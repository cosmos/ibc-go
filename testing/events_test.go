package ibctesting_test

import (
	"encoding/hex"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
			expectedError: "packet not found in events",
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
			expectedError: "packet not found in events",
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
			allPackets, err := ibctesting.ParsePacketsFromEvents(channeltypes.EventTypeSendPacket, tc.events)

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

func TestParsePacketsFromEventsV2(t *testing.T) {
	testCases := []struct {
		name            string
		events          []abci.Event
		expectedPackets []channeltypesv2.Packet
		expectedError   string
	}{
		{
			name: "success with multiple packets",
			events: []abci.Event{
				{
					Type: "xxx",
				},
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key: "packet_hex",
							Value: hex.EncodeToString(func() []byte {
								packet := channeltypesv2.Packet{
									SourceClient:      "client-0",
									DestinationClient: "client-1",
									Sequence:          1,
									TimeoutTimestamp:  100,
									Payloads: []channeltypesv2.Payload{
										{
											SourcePort:      "transfer",
											DestinationPort: "transfer",
											Version:         "1.0",
											Encoding:        "proto3",
											Value:           []byte("data1"),
										},
									},
								}
								bz, err := proto.Marshal(&packet)
								require.NoError(t, err)
								return bz
							}()),
						},
					},
				},
				{
					Type: "yyy",
				},
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key: "packet_hex",
							Value: hex.EncodeToString(func() []byte {
								packet := channeltypesv2.Packet{
									SourceClient:      "client-0",
									DestinationClient: "client-1",
									Sequence:          2,
									TimeoutTimestamp:  200,
									Payloads: []channeltypesv2.Payload{
										{
											SourcePort:      "transfer",
											DestinationPort: "transfer",
											Version:         "1.0",
											Encoding:        "proto3",
											Value:           []byte("data2"),
										},
									},
								}
								bz, err := proto.Marshal(&packet)
								require.NoError(t, err)
								return bz
							}()),
						},
					},
				},
			},
			expectedPackets: []channeltypesv2.Packet{
				{
					SourceClient:      "client-0",
					DestinationClient: "client-1",
					Sequence:          1,
					TimeoutTimestamp:  100,
					Payloads: []channeltypesv2.Payload{
						{
							SourcePort:      "transfer",
							DestinationPort: "transfer",
							Version:         "1.0",
							Encoding:        "proto3",
							Value:           []byte("data1"),
						},
					},
				},
				{
					SourceClient:      "client-0",
					DestinationClient: "client-1",
					Sequence:          2,
					TimeoutTimestamp:  200,
					Payloads: []channeltypesv2.Payload{
						{
							SourcePort:      "transfer",
							DestinationPort: "transfer",
							Version:         "1.0",
							Encoding:        "proto3",
							Value:           []byte("data2"),
						},
					},
				},
			},
		},
		{
			name:          "fail: no events",
			events:        []abci.Event{},
			expectedError: "packet not found in events",
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
			expectedError: "packet not found in events",
		},
		{
			name: "fail: invalid hex encoding",
			events: []abci.Event{
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   "packet_hex",
							Value: "invalid hex",
						},
					},
				},
			},
			expectedError: "failed to decode packet bytes",
		},
		{
			name: "fail: invalid proto encoding",
			events: []abci.Event{
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   "packet_hex",
							Value: hex.EncodeToString([]byte("invalid proto")),
						},
					},
				},
			},
			expectedError: "failed to unmarshal packet",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allPackets, err := ibctesting.ParsePacketsFromEventsV2(channeltypesv2.EventTypeSendPacket, tc.events)

			if tc.expectedError == "" {
				require.NoError(t, err)
				require.Equal(t, tc.expectedPackets, allPackets)

				// Test ParsePacketFromEventsV2 as well
				firstPacket, err := ibctesting.ParsePacketFromEventsV2(tc.events)
				require.NoError(t, err)
				require.Equal(t, tc.expectedPackets[0], firstPacket)
			} else {
				require.ErrorContains(t, err, tc.expectedError)

				// Test ParsePacketFromEventsV2 as well
				_, err = ibctesting.ParsePacketFromEventsV2(tc.events)
				require.ErrorContains(t, err, tc.expectedError)
			}
		})
	}
}
