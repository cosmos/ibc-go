package ibctesting

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	abci "github.com/cometbft/cometbft/abci/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
)

// ParsePacketFromEventsV2 parses a packet from events using the encoded packet hex attribute.
// It returns the parsed packet and an error if the packet cannot be found or decoded.
func ParsePacketFromEventsV2(events []abci.Event) (channeltypesv2.Packet, error) {
	for _, event := range events {
		if event.Type == channeltypesv2.EventTypeSendPacket {
			for _, attr := range event.Attributes {
				if string(attr.Key) == channeltypesv2.AttributeKeyEncodedPacketHex {
					packetBytes, err := hex.DecodeString(string(attr.Value))
					if err != nil {
						return channeltypesv2.Packet{}, fmt.Errorf("failed to decode packet bytes: %w", err)
					}
					var packet channeltypesv2.Packet
					if err := proto.Unmarshal(packetBytes, &packet); err != nil {
						return channeltypesv2.Packet{}, fmt.Errorf("failed to unmarshal packet: %w", err)
					}
					return packet, nil
				}
			}
		}
	}
	return channeltypesv2.Packet{}, fmt.Errorf("packet not found in events")
}

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

	// Convert sdk.Events to abci.Events
	abciEvents := make([]abci.Event, len(res.Events))
	for i, event := range res.Events {
		attributes := make([]abci.EventAttribute, len(event.Attributes))
		for j, attr := range event.Attributes {
			attributes[j] = abci.EventAttribute{
				Key:   attr.Key,
				Value: attr.Value,
			}
		}
		abciEvents[i] = abci.Event{
			Type:       event.Type,
			Attributes: attributes,
		}
	}

	return ParsePacketFromEventsV2(abciEvents)
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
