package types_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

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
			name: "valid custom state",
			genesisState: types.GenesisState{
				WhitelistedAddressPairs: []types.WhitelistedAddressPair{
					{Sender: "senderA", Receiver: "receiverA"},
					{Sender: "senderB", Receiver: "receiverB"},
				},
				BlacklistedDenoms:                []string{"denomA", "denomB"},
				PendingSendPacketSequenceNumbers: []string{"channel-0/1/denomA", "channel-2/3/denomB"},
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
				PendingSendPacketSequenceNumbers: []string{"channel-0/1/denomA", "channel-2|3"},
			},
			expectedError: "invalid pending packet (channel-2|3), must be of form: {channelId}/{sequenceNumber}/{denom}",
		},
		{
			name: "invalid packet sequence - invalid sequence",
			genesisState: types.GenesisState{
				PendingSendPacketSequenceNumbers: []string{"channel-0/1/denomA", "channel-2/X/denomB"},
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
				PendingRecvPacketSequenceNumbers: []string{"channel-0/1/denomA", "channel-2|3"},
			},
			expectedError: "invalid pending packet (channel-2|3), must be of form: {channelId}/{sequenceNumber}/{denom}",
		},
		{
			name: "invalid receive packet sequence - invalid sequence",
			genesisState: types.GenesisState{
				PendingRecvPacketSequenceNumbers: []string{"channel-0/1/denomA", "channel-2/X/denomB"},
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
