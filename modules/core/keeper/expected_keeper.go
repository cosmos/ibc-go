package keeper

import (
	"context"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

type PacketHandler interface {
	RecvPacket(
		ctx context.Context,
		packet channeltypes.Packet,
		proof []byte,
		proofHeight exported.Height) (string, error)

	WriteAcknowledgement(
		ctx context.Context,
		packet exported.PacketI,
		acknowledgement exported.Acknowledgement,
	) error

	AcknowledgePacket(
		ctx context.Context,
		packet channeltypes.Packet,
		acknowledgement []byte,
		proof []byte,
		proofHeight exported.Height,
	) (string, error)

	TimeoutPacket(
		ctx context.Context,
		packet channeltypes.Packet,
		proof []byte,
		proofHeight exported.Height,
		nextSequenceRecv uint64,
	) (string, error)
}
