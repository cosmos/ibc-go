package ibctesting

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
)

// RegisterCounterparty will construct and execute a MsgRegisterCounterparty on the associated endpoint.
func (endpoint *Endpoint) RegisterCounterparty() (err error) {
	msg := clientv2types.NewMsgRegisterCounterparty(endpoint.ClientID, endpoint.Counterparty.MerklePathPrefix.KeyPath, endpoint.Counterparty.ClientID, endpoint.Chain.SenderAccount.GetAddress().String())

	// setup counterparty
	_, err = endpoint.Chain.SendMsgs(msg)

	return err
}

// MsgSendPacket sends a packet on the associated endpoint using a predefined sender. The constructed packet is returned.
func (endpoint *Endpoint) MsgSendPacket(timeoutTimestamp uint64, payload channeltypesv2.Payload) (channeltypesv2.Packet, error) {
	senderAccount := SenderAccount{
		SenderPrivKey: endpoint.Chain.SenderPrivKey,
		SenderAccount: endpoint.Chain.SenderAccount,
	}

	return endpoint.MsgSendPacketWithSender(timeoutTimestamp, payload, senderAccount)
}

// MsgSendPacketWithSender sends a packet on the associated endpoint using the provided sender. The constructed packet is returned.
func (endpoint *Endpoint) MsgSendPacketWithSender(timeoutTimestamp uint64, payload channeltypesv2.Payload, sender SenderAccount) (channeltypesv2.Packet, error) {
	msgSendPacket := channeltypesv2.NewMsgSendPacket(endpoint.ClientID, timeoutTimestamp, sender.SenderAccount.GetAddress().String(), payload)

	res, err := endpoint.Chain.SendMsgsWithSender(sender, msgSendPacket)
	if err != nil {
		return channeltypesv2.Packet{}, err
	}

	if err := endpoint.Counterparty.UpdateClient(); err != nil {
		return channeltypesv2.Packet{}, err
	}

	// Parse the packet from events
	for _, event := range res.Events {
		if event.Type == channeltypesv2.EventTypeSendPacket {
			var packet channeltypesv2.Packet
			for _, attr := range event.Attributes {
				if string(attr.Key) == channeltypesv2.AttributeKeyEncodedPacketHex {
					packetBytes, err := hex.DecodeString(string(attr.Value))
					if err != nil {
						return channeltypesv2.Packet{}, err
					}
					if err := proto.Unmarshal(packetBytes, &packet); err != nil {
						return channeltypesv2.Packet{}, err
					}
					return packet, nil
				}
			}
		}
	}

	return channeltypesv2.Packet{}, fmt.Errorf("packet not found in events")
}

// MsgRecvPacket sends a MsgRecvPacket on the associated endpoint with the provided packet.
func (endpoint *Endpoint) MsgRecvPacket(packet channeltypesv2.Packet) error {
	// get proof of packet commitment from chainA
	packetKey := hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence)
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgRecvPacket(packet, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	if err := endpoint.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return endpoint.Counterparty.UpdateClient()
}

// MsgAcknowledgePacket sends a MsgAcknowledgement on the associated endpoint with the provided packet and ack.
func (endpoint *Endpoint) MsgAcknowledgePacket(packet channeltypesv2.Packet, ack channeltypesv2.Acknowledgement) error {
	packetKey := hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence)
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgAcknowledgement(packet, ack, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	if err := endpoint.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return endpoint.Counterparty.UpdateClient()
}

// MsgTimeoutPacket sends a MsgTimeout on the associated endpoint with the provided packet.
func (endpoint *Endpoint) MsgTimeoutPacket(packet channeltypesv2.Packet) error {
	packetKey := hostv2.PacketReceiptKey(packet.DestinationClient, packet.Sequence)
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgTimeout(packet, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	if err := endpoint.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return endpoint.Counterparty.UpdateClient()
}
