package ibctesting

import (
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/stretchr/testify/require"
)

// MsgSendPacket sends a packet on the associated endpoint. The constructed packet is returned.
func (endpoint *Endpoint) MsgSendPacket(timeoutTimestamp uint64, packetData channeltypesv2.PacketData) (channeltypesv2.Packet, error) {
	msgSendPacket := channeltypesv2.NewMsgSendPacket(endpoint.ChannelID, timeoutTimestamp, endpoint.Chain.SenderAccount.GetAddress().String(), packetData)

	_, err := endpoint.Chain.SendMsgs(msgSendPacket)
	if err != nil {
		return channeltypesv2.Packet{}, err
	}

	if err := endpoint.Counterparty.UpdateClient(); err != nil {
		return channeltypesv2.Packet{}, err
	}

	// TODO: parse the packet from events instead of manually constructing it. https://github.com/cosmos/ibc-go/issues/7459
	nextSequenceSend, ok := endpoint.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2.GetNextSequenceSend(endpoint.Chain.GetContext(), endpoint.ChannelID)
	require.True(endpoint.Chain.TB, ok)
	packet := channeltypesv2.NewPacket(nextSequenceSend-1, endpoint.ChannelID, endpoint.Counterparty.ChannelID, timeoutTimestamp, packetData)

	return packet, nil
}

//// RecvPacketWithResult receives a packet on the associated endpoint and the result
//// of the transaction is returned. The counterparty client is updated.
//func (endpoint *Endpoint) RecvPacketWithResult(packet channeltypes.Packet) (*abci.ExecTxResult, error) {
//	// get proof of packet commitment on source
//	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
//	proof, proofHeight := endpoint.Counterparty.Chain.QueryProof(packetKey)
//
//	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())
//
//	// receive on counterparty and update source client
//	res, err := endpoint.Chain.SendMsgs(recvMsg)
//	if err != nil {
//		return nil, err
//	}
//
//	if err := endpoint.Counterparty.UpdateClient(); err != nil {
//		return nil, err
//	}
//
//	return res, nil
//}
