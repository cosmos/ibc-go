package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// NewLegacyMultiAck returns an implementation of the exported.Acknowledgement interface which will be forwards
// compatible with the new MultiAck structure for PacketV2.
func NewLegacyMultiAck(cdc codec.BinaryCodec, ack exported.Acknowledgement, appName string) exported.Acknowledgement {
	var multiAck channeltypes.MultiAcknowledgement
	recvPacketResult := channeltypes.RecvPacketResult{
		Acknowledgement: ack.Acknowledgement(),
	}
	if ack.Success() {
		recvPacketResult.Status = channeltypes.PacketStatus_Success
	} else {
		recvPacketResult.Status = channeltypes.PacketStatus_Failure
	}
	multiAck.AcknowledgementResults = append(multiAck.AcknowledgementResults, channeltypes.AcknowledgementResult{
		AppName:          appName,
		RecvPacketResult: recvPacketResult,
	})

	return &legacyMultiAck{
		cdc:      cdc,
		multiAck: multiAck,
	}
}

type legacyMultiAck struct {
	cdc      codec.BinaryCodec
	multiAck channeltypes.MultiAcknowledgement
}

func (l *legacyMultiAck) Acknowledgement() []byte {
	return l.cdc.MustMarshal(&l.multiAck)
}

func (l *legacyMultiAck) Success() bool {
	return l.multiAck.AcknowledgementResults[0].RecvPacketResult.Status == channeltypes.PacketStatus_Success
}
