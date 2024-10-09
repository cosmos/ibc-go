package types

import (
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
)

// NewMsgSendPacket creates a new MsgSendPacket instance.
func NewMsgSendPacket(sourceID string, timeoutTimestamp uint64, signer string, packetData ...PacketData) *MsgSendPacket {
	return &MsgSendPacket{
		SourceId:         sourceID,
		TimeoutTimestamp: timeoutTimestamp,
		PacketData:       packetData,
		Signer:           signer,
	}
}

// NewMsgRecvPacket creates a new MsgRecvPacket instance.
func NewMsgRecvPacket(packet Packet, proofCommitment []byte, proofHeight clienttypes.Height, signer string) *MsgRecvPacket {
	return &MsgRecvPacket{
		Packet:          packet,
		ProofCommitment: proofCommitment,
		ProofHeight:     proofHeight,
		Signer:          signer,
	}
}
