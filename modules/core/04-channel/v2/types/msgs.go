package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	v1types "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

var _ sdk.Msg = (*MsgSendPacket)(nil)

// NewMsgSendPacket constructs and returns a new MsgSendPacket.
func NewMsgSendPacket(sourceID string, timeoutTimestamp uint64, signer string, packetData ...v1types.PacketData) *MsgSendPacket {
	return &MsgSendPacket{
		SourceId:         sourceID,
		TimeoutTimestamp: timeoutTimestamp,
		PacketData:       packetData,
		Signer:           signer,
	}
}

// NewMsgRecvPacket constructs and returns a new MsgRecvPacket
func NewMsgRecvPacket(packet v1types.PacketV2, proofCommitment []byte, proofHeight clienttypes.Height, signer string) *MsgRecvPacket {
	return &MsgRecvPacket{
		Packet:          packet,
		ProofCommitment: proofCommitment,
		ProofHeight:     proofHeight,
		Signer:          signer,
	}
}

// NewMsgAcknowledgement constructs and returns a new MsgAcknowledgement
func NewMsgAcknoweldgement(packet v1types.PacketV2, multiAck v1types.MultiAcknowledgement, proofAcked []byte, proofHeight clienttypes.Height, signer string) *MsgAcknowledgement {
	return &MsgAcknowledgement{
		Packet:               packet,
		MultiAcknowledgement: multiAck,
		ProofAcked:           proofAcked,
		ProofHeight:          proofHeight,
		Signer:               signer,
	}
}
