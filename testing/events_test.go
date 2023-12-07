package ibctesting_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestParsePacketsFromEvents(t *testing.T) {
	testCases := []struct {
		name            string
		events          []abci.Event
		expectedPackets []channeltypes.Packet
		expectedError   string
	}{
		{
			name: "success",
			events: []abci.Event{
				{
					Type: "xxx",
				},
				{
					Type: channeltypes.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypes.AttributeKeyData,
							Value: "data1",
						},
						{
							Key:   channeltypes.AttributeKeySequence,
							Value: "42",
						},
						{
							Key:   channeltypes.AttributeKeySrcPort,
							Value: "srcPort",
						},
						{
							Key:   channeltypes.AttributeKeySrcChannel,
							Value: "srcChannel",
						},
						{
							Key:   channeltypes.AttributeKeyDstPort,
							Value: "dstPort",
						},
						{
							Key:   channeltypes.AttributeKeyDstChannel,
							Value: "dstChannel",
						},
						{
							Key:   channeltypes.AttributeKeyTimeoutHeight,
							Value: "1-2",
						},
						{
							Key:   channeltypes.AttributeKeyTimeoutTimestamp,
							Value: "1000",
						},
					},
				},
				{
					Type: "yyy",
				},
				{
					Type: channeltypes.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypes.AttributeKeyData,
							Value: "data2",
						},
						{
							Key:   channeltypes.AttributeKeySequence,
							Value: "43",
						},
						{
							Key:   channeltypes.AttributeKeySrcPort,
							Value: "srcPort",
						},
						{
							Key:   channeltypes.AttributeKeySrcChannel,
							Value: "srcChannel",
						},
						{
							Key:   channeltypes.AttributeKeyDstPort,
							Value: "dstPort",
						},
						{
							Key:   channeltypes.AttributeKeyDstChannel,
							Value: "dstChannel",
						},
						{
							Key:   channeltypes.AttributeKeyTimeoutHeight,
							Value: "1-3",
						},
						{
							Key:   channeltypes.AttributeKeyTimeoutTimestamp,
							Value: "1001",
						},
					},
				},
			},
			expectedPackets: []channeltypes.Packet{
				{
					Sequence:           42,
					SourcePort:         "srcPort",
					SourceChannel:      "srcChannel",
					DestinationPort:    "dstPort",
					DestinationChannel: "dstChannel",
					Data:               []byte("data1"),
					TimeoutHeight: types.Height{
						RevisionNumber: 1,
						RevisionHeight: 2,
					},
					TimeoutTimestamp: 1000,
				},
				{
					Sequence:           43,
					SourcePort:         "srcPort",
					SourceChannel:      "srcChannel",
					DestinationPort:    "dstPort",
					DestinationChannel: "dstChannel",
					Data:               []byte("data2"),
					TimeoutHeight: types.Height{
						RevisionNumber: 1,
						RevisionHeight: 3,
					},
					TimeoutTimestamp: 1001,
				},
			},
		},

		{
			name:          "fail: no events",
			events:        []abci.Event{},
			expectedError: "acknowledgement event attribute not found",
		},
		{
			name: "fail: events without packet",
			events: []abci.Event{
				{
					Type: "xxx",
				},
				{
					Type: "yyy",
				},
			},
			expectedError: "acknowledgement event attribute not found",
		},
		{
			name: "fail: event packet with invalid AttributeKeySequence",
			events: []abci.Event{
				{
					Type: channeltypes.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypes.AttributeKeySequence,
							Value: "x",
						},
					},
				},
			},
			expectedError: "strconv.ParseUint: parsing \"x\": invalid syntax",
		},
		{
			name: "fail: event packet with invalid AttributeKeyTimeoutHeight",
			events: []abci.Event{
				{
					Type: channeltypes.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypes.AttributeKeyTimeoutHeight,
							Value: "x",
						},
					},
				},
			},
			expectedError: "expected height string format: {revision}-{height}. Got: x: invalid height",
		},
		{
			name: "fail: event packet with invalid AttributeKeyTimeoutTimestamp",
			events: []abci.Event{
				{
					Type: channeltypes.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypes.AttributeKeyTimeoutTimestamp,
							Value: "x",
						},
					},
				},
			},
			expectedError: "strconv.ParseUint: parsing \"x\": invalid syntax",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allPackets, err := ibctesting.ParsePacketsFromEvents(tc.events)

			if tc.expectedError == "" {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPackets, allPackets)
			} else {
				require.ErrorContains(t, err, tc.expectedError)
			}

			firstPacket, err := ibctesting.ParsePacketFromEvents(tc.events)

			if tc.expectedError == "" {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPackets[0], firstPacket)
			} else {
				require.ErrorContains(t, err, tc.expectedError)
			}
		})
	}
}
