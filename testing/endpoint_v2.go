package ibctesting

import (
	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
)

// RegisterCounterparty will construct and execute a MsgRegisterCounterparty on the associated ep.
func (ep *Endpoint) RegisterCounterparty() error {
	msg := clientv2types.NewMsgRegisterCounterparty(ep.ClientID, ep.Counterparty.MerklePathPrefix.KeyPath, ep.Counterparty.ClientID, ep.Chain.SenderAccount.GetAddress().String())

	// setup counterparty
	_, err := ep.Chain.SendMsgs(msg)

	return err
}

// MsgSendPacket sends a packet on the associated endpoint using a predefined sender. The constructed packet is returned.
func (ep *Endpoint) MsgSendPacket(timeoutTimestamp uint64, payloads ...channeltypesv2.Payload) (channeltypesv2.Packet, error) {
	senderAccount := SenderAccount{
		SenderPrivKey: ep.Chain.SenderPrivKey,
		SenderAccount: ep.Chain.SenderAccount,
	}

	return ep.MsgSendPacketWithSender(timeoutTimestamp, payloads, senderAccount)
}

// MsgSendPacketWithSender sends a packet on the associated endpoint using the provided sender. The constructed packet is returned.
func (ep *Endpoint) MsgSendPacketWithSender(timeoutTimestamp uint64, payloads []channeltypesv2.Payload, sender SenderAccount) (channeltypesv2.Packet, error) {
	msgSendPacket := channeltypesv2.NewMsgSendPacket(ep.ClientID, timeoutTimestamp, sender.SenderAccount.GetAddress().String(), payloads...)

	res, err := ep.Chain.SendMsgsWithSender(sender, msgSendPacket)
	if err != nil {
		return channeltypesv2.Packet{}, err
	}

	if err := ep.Counterparty.UpdateClient(); err != nil {
		return channeltypesv2.Packet{}, err
	}

	// TODO: parse the packet from events instead of from the response. https://github.com/cosmos/ibc-go/issues/7459
	// get sequence from msg response
	var msgData sdk.TxMsgData
	err = proto.Unmarshal(res.Data, &msgData)
	if err != nil {
		return channeltypesv2.Packet{}, err
	}
	msgResponse := msgData.MsgResponses[0]
	var sendResponse channeltypesv2.MsgSendPacketResponse
	err = proto.Unmarshal(msgResponse.Value, &sendResponse)
	if err != nil {
		return channeltypesv2.Packet{}, err
	}
	packet := channeltypesv2.NewPacket(sendResponse.Sequence, ep.ClientID, ep.Counterparty.ClientID, timeoutTimestamp, payloads...)

	err = ep.Counterparty.UpdateClient()
	if err != nil {
		return channeltypesv2.Packet{}, err
	}

	return packet, nil
}

// MsgRecvPacket sends a MsgRecvPacket on the associated endpoint with the provided packet.
func (ep *Endpoint) MsgRecvPacket(packet channeltypesv2.Packet) error {
	// get proof of packet commitment from chainA
	packetKey := hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence)
	proof, proofHeight := ep.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgRecvPacket(packet, proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String())

	if err := ep.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return ep.Counterparty.UpdateClient()
}

// MsgRecvPacketWithAck returns the acknowledgement for the given packet by sending a MsgRecvPacket on the associated endpoint.
func (ep *Endpoint) MsgRecvPacketWithAck(packet channeltypesv2.Packet) (channeltypesv2.Acknowledgement, error) {
	// get proof of packet commitment from chainA
	packetKey := hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence)
	proof, proofHeight := ep.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgRecvPacket(packet, proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String())

	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return channeltypesv2.Acknowledgement{}, err
	}

	ackBz, err := ParseAckV2FromEvents(res.Events)
	if err != nil {
		return channeltypesv2.Acknowledgement{}, err
	}
	var ack channeltypesv2.Acknowledgement
	err = proto.Unmarshal(ackBz, &ack)
	if err != nil {
		return channeltypesv2.Acknowledgement{}, err
	}

	err = ep.Counterparty.UpdateClient()
	if err != nil {
		return channeltypesv2.Acknowledgement{}, err
	}

	return ack, nil
}

// MsgAcknowledgePacket sends a MsgAcknowledgement on the associated endpoint with the provided packet and ack.
func (ep *Endpoint) MsgAcknowledgePacket(packet channeltypesv2.Packet, ack channeltypesv2.Acknowledgement) error {
	packetKey := hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence)
	proof, proofHeight := ep.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgAcknowledgement(packet, ack, proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String())

	if err := ep.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return ep.Counterparty.UpdateClient()
}

// MsgTimeoutPacket sends a MsgTimeout on the associated endpoint with the provided packet.
func (ep *Endpoint) MsgTimeoutPacket(packet channeltypesv2.Packet) error {
	packetKey := hostv2.PacketReceiptKey(packet.DestinationClient, packet.Sequence)
	proof, proofHeight := ep.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgTimeout(packet, proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String())

	if err := ep.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return ep.Counterparty.UpdateClient()
}

// RelayPacket relayes packet that was previously sent on the given endpoint.
func (ep *Endpoint) RelayPacket(packet channeltypesv2.Packet) error {
	// receive packet on counterparty
	ack, err := ep.Counterparty.MsgRecvPacketWithAck(packet)
	if err != nil {
		return err
	}

	// acknowledge packet on endpoint
	return ep.MsgAcknowledgePacket(packet, ack)
}
