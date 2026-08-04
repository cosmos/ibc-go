package types_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

const (
	pendingGenesisPacketID = "channel-0/1/denomA"
	uatomDenom             = "uatom"
	senderA                = "senderA"
	receiverA              = "receiverA"
)

func genesisRateLimit(denom, channelOrClientID string) types.RateLimit {
	return types.RateLimit{
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelOrClientID},
		Quota: &types.Quota{
			MaxPercentSend: sdkmath.NewInt(10),
			MaxPercentRecv: sdkmath.NewInt(20),
			DurationHours:  24,
		},
		Flow: &types.Flow{
			Inflow:       sdkmath.ZeroInt(),
			Outflow:      sdkmath.NewInt(5),
			ChannelValue: sdkmath.NewInt(100),
		},
	}
}

func malleatedGenesisRateLimit(malleate func(*types.RateLimit)) types.RateLimit {
	rateLimit := genesisRateLimit(uatomDenom, "channel-1")
	malleate(&rateLimit)

	return rateLimit
}

func TestValidateGenesis(t *testing.T) {
	currentHour := 13
	blockTime := time.Date(2024, 1, 1, currentHour, 55, 8, 0, time.UTC) // 13:55:08

	testCases := []struct {
		name          string
		genesisState  types.GenesisState
		expectedError string
	}{
		{
			name:         "valid default state",
			genesisState: *types.DefaultGenesis(),
		},
		{
			name: "valid legacy pending packet sequences",
			genesisState: types.GenesisState{
				PendingSendPacketSequenceNumbers: []string{"channel-0/1"},
				PendingRecvPacketSequenceNumbers: []string{"channel-2/3"},
				HourEpoch: types.HourEpoch{
					Duration: time.Hour,
				},
			},
		},
		{
			name: "valid custom state",
			genesisState: types.GenesisState{
				WhitelistedAddressPairs: []types.WhitelistedAddressPair{
					{Sender: senderA, Receiver: receiverA},
					{Sender: "senderB", Receiver: "receiverB"},
				},
				BlacklistedDenoms:                []string{"denomA", "denomB"},
				PendingSendPacketSequenceNumbers: []string{pendingGenesisPacketID, "channel-2/3/denomB"},
				PendingRecvPacketSequenceNumbers: []string{"channel-4/5/transfer/channel-0/denomC", "channel-6/7/denomD"},
				HourEpoch: types.HourEpoch{
					EpochNumber:      1,
					EpochStartTime:   blockTime,
					Duration:         time.Minute,
					EpochStartHeight: 1,
				},
			},
		},
		{
			name: "valid rate limits",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{
					genesisRateLimit(uatomDenom, "channel-1"),
					genesisRateLimit(uatomDenom, "07-tendermint-0"),
					genesisRateLimit("uosmo", "channel-1"),
				},
				HourEpoch: types.HourEpoch{Duration: time.Hour},
			},
		},
		{
			name: "invalid rate limit - nil path",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{{}},
			},
			expectedError: "rate limit path must be specified",
		},
		{
			name: "invalid rate limit - empty denom",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{genesisRateLimit("", "channel-1")},
			},
			expectedError: "rate limit denom must be specified",
		},
		{
			name: "invalid rate limit - malformed channel or client ID",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{genesisRateLimit(uatomDenom, "channel-abc")},
			},
			expectedError: "invalid channel or client-id (channel-abc)",
		},
		{
			name: "invalid rate limit - nil quota",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{{Path: &types.Path{Denom: uatomDenom, ChannelOrClientId: "channel-1"}}},
			},
			expectedError: "rate limit quota must be specified for denom uatom on channel-1",
		},
		{
			name: "invalid rate limit - zero quota duration",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{malleatedGenesisRateLimit(func(rateLimit *types.RateLimit) {
					rateLimit.Quota.DurationHours = 0
				})},
			},
			expectedError: "duration can not be zero",
		},
		{
			name: "invalid rate limit - nil max percent recv",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{malleatedGenesisRateLimit(func(rateLimit *types.RateLimit) {
					rateLimit.Quota.MaxPercentRecv = sdkmath.Int{}
				})},
			},
			expectedError: "max-percent-send and max-percent-recv must be specified",
		},
		{
			name: "invalid rate limit - nil flow",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{malleatedGenesisRateLimit(func(rateLimit *types.RateLimit) {
					rateLimit.Flow = nil
				})},
			},
			expectedError: "rate limit flow must be specified for denom uatom on channel-1",
		},
		{
			name: "invalid rate limit - nil flow inflow",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{malleatedGenesisRateLimit(func(rateLimit *types.RateLimit) {
					rateLimit.Flow.Inflow = sdkmath.Int{}
				})},
			},
			expectedError: "inflow must be specified",
		},
		{
			name: "invalid rate limit - nil flow channel value",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{malleatedGenesisRateLimit(func(rateLimit *types.RateLimit) {
					rateLimit.Flow.ChannelValue = sdkmath.Int{}
				})},
			},
			expectedError: "channel value must be specified",
		},
		{
			name: "invalid rate limit - negative flow outflow",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{malleatedGenesisRateLimit(func(rateLimit *types.RateLimit) {
					rateLimit.Flow.Outflow = sdkmath.NewInt(-1)
				})},
			},
			expectedError: "outflow cannot be negative, provided: -1",
		},
		{
			name: "invalid rate limit - duplicate path",
			genesisState: types.GenesisState{
				RateLimits: []types.RateLimit{
					genesisRateLimit(uatomDenom, "channel-1"),
					genesisRateLimit(uatomDenom, "channel-1"),
				},
			},
			expectedError: "duplicate rate limit for denom uatom on channel-1",
		},
		{
			name: "invalid blacklist - empty denom",
			genesisState: types.GenesisState{
				BlacklistedDenoms: []string{"denomA", ""},
			},
			expectedError: "blacklisted denom must be specified",
		},
		{
			name: "invalid whitelist - empty sender",
			genesisState: types.GenesisState{
				WhitelistedAddressPairs: []types.WhitelistedAddressPair{{Sender: "", Receiver: receiverA}},
			},
			expectedError: "whitelisted address pair sender must be specified",
		},
		{
			name: "invalid whitelist - empty receiver",
			genesisState: types.GenesisState{
				WhitelistedAddressPairs: []types.WhitelistedAddressPair{{Sender: senderA, Receiver: ""}},
			},
			expectedError: "whitelisted address pair receiver must be specified",
		},
		{
			name: "invalid whitelist - duplicate pair",
			genesisState: types.GenesisState{
				WhitelistedAddressPairs: []types.WhitelistedAddressPair{
					{Sender: senderA, Receiver: receiverA},
					{Sender: senderA, Receiver: receiverA},
				},
			},
			expectedError: "duplicate whitelisted address pair for sender senderA and receiver receiverA",
		},
		{
			name: "invalid packet sequence - wrong delimiter",
			genesisState: types.GenesisState{
				PendingSendPacketSequenceNumbers: []string{pendingGenesisPacketID, "channel-2|3"},
			},
			expectedError: "invalid pending packet (channel-2|3), must be of form: {channelId}/{sequenceNumber}/{denom}",
		},
		{
			name: "invalid packet sequence - invalid sequence",
			genesisState: types.GenesisState{
				PendingSendPacketSequenceNumbers: []string{pendingGenesisPacketID, "channel-2/X/denomB"},
			},
			expectedError: "unable to parse sequence number (X) from pending packet",
		},
		{
			name: "invalid packet sequence - ID too long",
			genesisState: types.GenesisState{
				PendingSendPacketSequenceNumbers: []string{strings.Repeat("a", types.PendingSendPacketChannelLength+1) + "/1/denomA"},
			},
			expectedError: "greater than the allowed length 64",
		},
		{
			name: "invalid packet sequence - channel ID contains 0x00",
			genesisState: types.GenesisState{
				PendingSendPacketSequenceNumbers: []string{"channel-\x00/1/denomA"},
			},
			expectedError: "cannot contain 0x00",
		},
		{
			name: "invalid packet sequence - denom contains 0x00",
			genesisState: types.GenesisState{
				PendingSendPacketSequenceNumbers: []string{"channel-0/1/denom\x00A"},
			},
			expectedError: "pending packet denom cannot contain 0x00",
		},
		{
			name: "invalid receive packet sequence - wrong delimiter",
			genesisState: types.GenesisState{
				PendingRecvPacketSequenceNumbers: []string{pendingGenesisPacketID, "channel-2|3"},
			},
			expectedError: "invalid pending packet (channel-2|3), must be of form: {channelId}/{sequenceNumber}/{denom}",
		},
		{
			name: "invalid receive packet sequence - invalid sequence",
			genesisState: types.GenesisState{
				PendingRecvPacketSequenceNumbers: []string{pendingGenesisPacketID, "channel-2/X/denomB"},
			},
			expectedError: "unable to parse sequence number (X) from pending packet",
		},
		{
			name: "invalid receive packet sequence - ID too long",
			genesisState: types.GenesisState{
				PendingRecvPacketSequenceNumbers: []string{strings.Repeat("a", types.PendingSendPacketChannelLength+1) + "/1/denomA"},
			},
			expectedError: "greater than the allowed length 64",
		},
		{
			name: "invalid hour epoch - no duration",
			genesisState: types.GenesisState{
				HourEpoch: types.HourEpoch{},
			},
			expectedError: "hour epoch duration must be positive",
		},
		{
			name: "invalid hour epoch - negative duration",
			genesisState: types.GenesisState{
				HourEpoch: types.HourEpoch{Duration: -time.Hour},
			},
			expectedError: "hour epoch duration must be positive",
		},
		{
			name: "invalid hour epoch - no epoch time",
			genesisState: types.GenesisState{
				HourEpoch: types.HourEpoch{
					EpochNumber:      1,
					EpochStartHeight: 1,
					Duration:         time.Minute,
				},
			},
			expectedError: "if hour epoch number is non-empty, epoch time must be initialized",
		},
		{
			// InitGenesis seeds the epoch at InitChain, where the block height is 0,
			// so keeper-produced pre-first-rollover state has a zero start height
			name: "valid hour epoch - no epoch height",
			genesisState: types.GenesisState{
				HourEpoch: types.HourEpoch{
					EpochNumber:    1,
					EpochStartTime: blockTime,
					Duration:       time.Minute,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genesisState.Validate()
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsLegacyPendingPacketID(t *testing.T) {
	testCases := []struct {
		name       string
		packetID   string
		isLegacyID bool
	}{
		{
			name:       "valid legacy ID",
			packetID:   "channel-0/1",
			isLegacyID: true,
		},
		{
			name:     "invalid legacy sequence",
			packetID: "channel-0/X",
		},
		{
			name:     "invalid legacy channel ID contains 0x00",
			packetID: "channel-\x00/1",
		},
		{
			name:     "denom-scoped ID",
			packetID: pendingGenesisPacketID,
		},
		{
			name:     "wrong delimiter",
			packetID: "channel-0|1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isLegacyID, types.IsLegacyPendingPacketID(tc.packetID))
		})
	}
}

func TestParsePendingPacketID(t *testing.T) {
	testCases := []struct {
		name          string
		packetID      string
		expChannelID  string
		expSequence   uint64
		expDenom      string
		expectedError string
	}{
		{
			name:         "valid denom with slashes",
			packetID:     "channel-0/1/transfer/channel-1/uatom",
			expChannelID: "channel-0",
			expSequence:  1,
			expDenom:     "transfer/channel-1/uatom",
		},
		{
			name:          "empty denom",
			packetID:      "channel-0/1/",
			expectedError: "denom must be specified",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			channelID, sequence, denom, err := types.ParsePendingPacketID(tc.packetID)
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expChannelID, channelID)
			require.Equal(t, tc.expSequence, sequence)
			require.Equal(t, tc.expDenom, denom)
		})
	}
}
