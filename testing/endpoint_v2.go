package ibctesting

import (
	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
)

// CreateChannel will construct and execute a new MsgCreateChannel on the associated endpoint.
func (endpoint *Endpoint) CreateChannel() (err error) {
	endpoint.IncrementNextChannelSequence()
	msg := channeltypesv2.NewMsgCreateChannel(endpoint.ClientID, endpoint.MerklePathPrefix, endpoint.Chain.SenderAccount.GetAddress().String())

	// create channel
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ChannelID, err = ParseChannelIDFromEvents(res.Events)
	if err != nil {
		return err
	}

	return nil
}

// RegisterCounterparty will construct and execute a MsgRegisterCounterparty on the associated endpoint.
func (endpoint *Endpoint) RegisterCounterparty() (err error) {
	msg := channeltypesv2.NewMsgRegisterCounterparty(endpoint.ChannelID, endpoint.Counterparty.ChannelID, endpoint.Chain.SenderAccount.GetAddress().String())

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
	msgSendPacket := channeltypesv2.NewMsgSendPacket(endpoint.ChannelID, timeoutTimestamp, sender.SenderAccount.GetAddress().String(), payload)

	res, err := endpoint.Chain.SendMsgsWithSender(sender, msgSendPacket)
	if err != nil {
		return channeltypesv2.Packet{}, err
	}

	if err := endpoint.Counterparty.UpdateClient(); err != nil {
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
	packet := channeltypesv2.NewPacket(sendResponse.Sequence, endpoint.ChannelID, endpoint.Counterparty.ChannelID, timeoutTimestamp, payload)

	return packet, nil
}

// MsgRecvPacket sends a MsgRecvPacket on the associated endpoint with the provided packet.
func (endpoint *Endpoint) MsgRecvPacket(packet channeltypesv2.Packet) error {
	// get proof of packet commitment from chainA
	packetKey := hostv2.PacketCommitmentKey(packet.SourceChannel, packet.Sequence)
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgRecvPacket(packet, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	if err := endpoint.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return endpoint.Counterparty.UpdateClient()
}

// MsgAcknowledgePacket sends a MsgAcknowledgement on the associated endpoint with the provided packet and ack.
func (endpoint *Endpoint) MsgAcknowledgePacket(packet channeltypesv2.Packet, ack channeltypesv2.Acknowledgement) error {
	packetKey := hostv2.PacketAcknowledgementKey(packet.DestinationChannel, packet.Sequence)
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgAcknowledgement(packet, ack, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	if err := endpoint.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return endpoint.Counterparty.UpdateClient()
}

// MsgTimeoutPacket sends a MsgTimeout on the associated endpoint with the provided packet.
func (endpoint *Endpoint) MsgTimeoutPacket(packet channeltypesv2.Packet) error {
	packetKey := hostv2.PacketReceiptKey(packet.DestinationChannel, packet.Sequence)
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	msg := channeltypesv2.NewMsgTimeout(packet, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	if err := endpoint.Chain.sendMsgs(msg); err != nil {
		return err
	}

	return endpoint.Counterparty.UpdateClient()
}
