package ibctesting_test

import (
	"encoding/hex"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv1 "github.com/cosmos/ibc-go/v10/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

func TestParseV1PacketsFromEvents(t *testing.T) {
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
							Key:   channeltypes.AttributeKeyDataHex,
							Value: hex.EncodeToString([]byte("data1")),
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
							Key:   channeltypes.AttributeKeyDataHex,
							Value: hex.EncodeToString([]byte("data2")),
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
			allPackets, err := ibctesting.ParseIBCV1Packets(channeltypes.EventTypeSendPacket, tc.events)

			if tc.expectedError == "" {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPackets, allPackets)
			} else {
				require.ErrorContains(t, err, tc.expectedError)
			}

			firstPacket, err := ibctesting.ParseV1PacketFromEvents(tc.events)

			if tc.expectedError == "" {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPackets[0], firstPacket)
			} else {
				require.ErrorContains(t, err, tc.expectedError)
			}
		})
	}
}

func TestParseV2PacketsFromEvents(t *testing.T) {
	testCases := []struct {
		name            string
		eventType       string
		events          []abci.Event
		expectedPackets []channeltypesv2.Packet
		expectedError   string
	}{
		{
			name:      "success: One v2 packet without payload + One v2 packet with payload",
			eventType: channeltypesv2.EventTypeRecvPacket,
			events: []abci.Event{
				{
					Type: "xxx",
				},
				{
					Type: channeltypesv2.EventTypeRecvPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypesv2.AttributeKeySequence,
							Value: "42",
						},
						{
							Key:   channeltypesv2.AttributeKeySrcClient,
							Value: "srcClient",
						},
						{
							Key:   channeltypesv2.AttributeKeyDstClient,
							Value: "destClient",
						},
						{
							Key:   channeltypesv2.AttributeKeyTimeoutTimestamp,
							Value: "1283798137",
						},
					},
				},
				{
					Type: "yyy",
				},
				{
					Type: channeltypesv2.EventTypeRecvPacket,
					// If AttributeKeyEncodedPacketHex is present, other attributes are ignored.
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypesv2.AttributeKeySequence,
							Value: "43", // Value Ignored
						},
						{
							Key:   channeltypesv2.AttributeKeySrcClient,
							Value: "srcClient-2", // Value Ignored
						},
						{
							Key:   channeltypesv2.AttributeKeyDstClient,
							Value: "destClient-2", // Value Ignored
						},
						{
							Key:   channeltypesv2.AttributeKeyTimeoutTimestamp,
							Value: "12837997475", // Value Ignored
						},
						{
							Key: channeltypesv2.AttributeKeyEncodedPacketHex,
							Value: func() string {
								payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)
								packet := channeltypesv2.NewPacket(44, "srcClient", "destClient", 19474197444, payload)
								encodedPacket, err := proto.Marshal(&packet)
								if err != nil {
									panic(err)
								}
								return hex.EncodeToString(encodedPacket)
							}(),
						},
					},
				},
			},
			expectedPackets: []channeltypesv2.Packet{
				{
					Sequence:          42,
					SourceClient:      "srcClient",
					DestinationClient: "destClient",
					TimeoutTimestamp:  1283798137,
				},
				{
					Sequence:          44,
					SourceClient:      "srcClient",
					DestinationClient: "destClient",
					TimeoutTimestamp:  19474197444,
					Payloads: []channeltypesv2.Payload{
						{
							SourcePort:      mockv2.ModuleNameA,
							DestinationPort: mockv2.ModuleNameB,
							Version:         mockv1.Version,
							Encoding:        transfertypes.EncodingProtobuf,
							Value:           mockv1.MockPacketData,
						},
					},
				},
			},
		},

		{
			name:          "fail: no events",
			eventType:     channeltypesv2.EventTypeSendPacket,
			events:        []abci.Event{},
			expectedError: "no IBC v2 packets found in events",
		},
		{
			name:      "fail: events without packet",
			eventType: channeltypesv2.EventTypeSendPacket,
			events: []abci.Event{
				{
					Type: "xxx",
				},
				{
					Type: "yyy",
				},
			},
			expectedError: "no IBC v2 packets found in events",
		},
		{
			name:      "fail: event packet with invalid AttributeKeySequence",
			eventType: channeltypesv2.EventTypeSendPacket,
			events: []abci.Event{
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypesv2.AttributeKeySequence,
							Value: "x",
						},
					},
				},
			},
			expectedError: "strconv.ParseUint: parsing \"x\": invalid syntax",
		},
		{
			name:      "fail: event packet with invalid AttributeKeyTimeoutHeight",
			eventType: channeltypesv2.EventTypeSendPacket,
			events: []abci.Event{
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypesv2.AttributeKeyTimeoutTimestamp,
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
			packets, err := ibctesting.ParseIBCV2Packets(tc.eventType, tc.events)
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedPackets, packets)
		})
	}
}
