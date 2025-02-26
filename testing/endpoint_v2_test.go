package ibctesting

import (
	"encoding/hex"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

func TestParsePacketFromEventsV2(t *testing.T) {
	testCases := []struct {
		name          string
		events        []abci.Event
		expectedError string
	}{
		{
			name: "successful packet parsing",
			events: []abci.Event{
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key: channeltypesv2.AttributeKeyEncodedPacketHex,
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
											Value:           []byte("test data"),
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
			expectedError: "",
		},
		{
			name:          "empty events",
			events:        []abci.Event{},
			expectedError: "packet not found in events",
		},
		{
			name: "invalid hex encoding",
			events: []abci.Event{
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypesv2.AttributeKeyEncodedPacketHex,
							Value: "invalid hex",
						},
					},
				},
			},
			expectedError: "failed to decode packet bytes",
		},
		{
			name: "invalid proto encoding",
			events: []abci.Event{
				{
					Type: channeltypesv2.EventTypeSendPacket,
					Attributes: []abci.EventAttribute{
						{
							Key:   channeltypesv2.AttributeKeyEncodedPacketHex,
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
			packet, err := ParsePacketFromEventsV2(tc.events)
			if tc.expectedError == "" {
				require.NoError(t, err)
				require.NotEmpty(t, packet)
				require.Equal(t, "client-0", packet.SourceClient)
				require.Equal(t, "client-1", packet.DestinationClient)
				require.Equal(t, uint64(1), packet.Sequence)
				require.Equal(t, uint64(100), packet.TimeoutTimestamp)
				require.Len(t, packet.Payloads, 1)
				require.Equal(t, "transfer", packet.Payloads[0].SourcePort)
				require.Equal(t, "transfer", packet.Payloads[0].DestinationPort)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestMsgSendPacketWithSender(t *testing.T) {
	coordinator := NewCoordinator(t, 2)
	chainA := coordinator.GetChain(GetChainID(1))
	chainB := coordinator.GetChain(GetChainID(2))
	coordinator.CommitBlock(chainA, chainB)

	path := NewPath(chainA, chainB)
	coordinator.SetupConnections(path)

	// Create a mock payload
	payload := channeltypesv2.Payload{
		SourcePort:      "sourcePort",
		DestinationPort: "destPort",
		Version:         "1.0",
		Encoding:        "proto3",
		Value:           []byte("test data"),
	}

	// Create sender account
	senderAccount := SenderAccount{
		SenderPrivKey: chainA.SenderPrivKey,
		SenderAccount: chainA.SenderAccount,
	}

	t.Run("successful packet send", func(t *testing.T) {
		// Send packet
		timeoutTimestamp := chainA.GetTimeoutTimestamp()
		packet, err := path.EndpointA.MsgSendPacketWithSender(timeoutTimestamp, payload, senderAccount)
		require.NoError(t, err)

		// Verify packet fields
		require.Equal(t, path.EndpointA.ClientID, packet.SourceClient)
		require.Equal(t, path.EndpointB.ClientID, packet.DestinationClient)
		require.Equal(t, timeoutTimestamp, packet.TimeoutTimestamp)
		require.Len(t, packet.Payloads, 1)
		require.Equal(t, payload, packet.Payloads[0])
	})

	t.Run("packet not found in events", func(t *testing.T) {
		// Mock a failed send by using an invalid payload that will cause the event to be missing
		invalidPayload := channeltypesv2.Payload{
			SourcePort:      "", // Invalid empty source port
			DestinationPort: "", // Invalid empty destination port
			Version:         "",
			Encoding:        "",
			Value:           nil,
		}

		timeoutTimestamp := chainA.GetTimeoutTimestamp()
		_, err := path.EndpointA.MsgSendPacketWithSender(timeoutTimestamp, invalidPayload, senderAccount)
		require.Error(t, err)
		require.Contains(t, err.Error(), "packet not found in events")
	})
}
