package types_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

const pendingGenesisPacketID = "channel-0/1/denomA"

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
					{Sender: "senderA", Receiver: "receiverA"},
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
			expectedError: "hour epoch duration must be specified",
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
			name: "invalid hour epoch - no epoch height",
			genesisState: types.GenesisState{
				HourEpoch: types.HourEpoch{
					EpochNumber:    1,
					EpochStartTime: blockTime,
					Duration:       time.Minute,
				},
			},
			expectedError: "if hour epoch number is non-empty, epoch height must be initialized",
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
