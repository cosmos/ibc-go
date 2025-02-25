package ibctesting

import (
	"testing"

	"github.com/stretchr/testify/require"

	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

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
