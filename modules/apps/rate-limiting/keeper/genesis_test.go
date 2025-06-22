package keeper_test

import (
	"strconv"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

func createRateLimits() []types.RateLimit {
	rateLimits := []types.RateLimit{}
	for i := int64(1); i <= 3; i++ {
		suffix := strconv.Itoa(int(i))
		rateLimit := types.RateLimit{
			Path:  &types.Path{Denom: "denom-" + suffix, ChannelOrClientId: "channel-" + suffix},
			Quota: &types.Quota{MaxPercentSend: sdkmath.NewInt(i), MaxPercentRecv: sdkmath.NewInt(i), DurationHours: uint64(i)}, //nolint:gosec
			Flow:  &types.Flow{Inflow: sdkmath.NewInt(i), Outflow: sdkmath.NewInt(i), ChannelValue: sdkmath.NewInt(i)},
		}

		rateLimits = append(rateLimits, rateLimit)
	}
	return rateLimits
}

func (s *KeeperTestSuite) TestGenesis() {
	currentHour := 13
	blockTime := time.Date(2024, 1, 1, currentHour, 55, 8, 0, time.UTC) // 13:55:08
	blockHeight := int64(10)

	testCases := []struct {
		name         string
		genesisState types.GenesisState
		firstEpoch   bool
		panicError   string
	}{
		{
			name:         "valid default state",
			genesisState: *types.DefaultGenesis(),
			firstEpoch:   true,
		},
		{
			name: "valid custom state",
			genesisState: types.GenesisState{
				RateLimits: createRateLimits(),
				WhitelistedAddressPairs: []types.WhitelistedAddressPair{
					{Sender: "senderA", Receiver: "receiverA"},
					{Sender: "senderB", Receiver: "receiverB"},
				},
				BlacklistedDenoms:                []string{"denomA", "denomB"},
				PendingSendPacketSequenceNumbers: []string{"channel-0/1", "channel-2/3"},
				HourEpoch: types.HourEpoch{
					EpochNumber:      1,
					EpochStartTime:   blockTime,
					Duration:         time.Minute,
					EpochStartHeight: 1,
				},
			},
			firstEpoch: false,
		},
		{
			name: "invalid packet sequence - wrong delimiter",
			genesisState: types.GenesisState{
				RateLimits:                       createRateLimits(),
				PendingSendPacketSequenceNumbers: []string{"channel-0/1", "channel-2|3"},
			},
			panicError: "invalid pending send packet (channel-2|3), must be of form: {channelId}/{sequenceNumber}",
		},
	}

	// Establish base height and time before the loop
	s.coordinator.CommitNBlocks(s.chainA, uint64(blockHeight-s.chainA.App.LastBlockHeight()+1))
	s.coordinator.SetTime(blockTime)

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.panicError != "" {
				s.Require().PanicsWithValue(tc.panicError, func() {
					s.chainA.GetSimApp().RateLimitKeeper.InitGenesis(s.chainA.GetContext(), tc.genesisState)
				})
				return
			}
			s.chainA.GetSimApp().RateLimitKeeper.InitGenesis(s.chainA.GetContext(), tc.genesisState)

			// If the hour epoch was not initialized in the raw genState,
			// it will be initialized during InitGenesis
			expectedGenesis := tc.genesisState

			// For the default genesis with firstEpoch=true, InitGenesis will set the HourEpoch fields
			// based on the current block time and height
			if tc.firstEpoch {
				// Get the context to retrieve current height
				ctx := s.chainA.GetContext()

				// For a new epoch, InitGenesis will:
				// - Set EpochNumber to current hour (13 from blockTime)
				// - Set EpochStartTime to the truncated hour (13:00:00)
				// - Set EpochStartHeight to current block height
				expectedGenesis.HourEpoch.EpochNumber = uint64(blockTime.Hour())
				expectedGenesis.HourEpoch.EpochStartTime = blockTime.Truncate(time.Hour)
				expectedGenesis.HourEpoch.EpochStartHeight = ctx.BlockHeight()
			}

			// Check that the exported state matches the imported state
			exportedState := s.chainA.GetSimApp().RateLimitKeeper.ExportGenesis(s.chainA.GetContext())
			s.Require().Equal(expectedGenesis, *exportedState, "exported genesis state")
		})
	}
}
