package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

type PacketHandler interface {
	RecvPacket(
		ctx sdk.Context,
		packet channeltypes.Packet,
		proof []byte,
		proofHeight exported.Height) (string, error)

	WriteAcknowledgement(
		ctx sdk.Context,
		packet exported.PacketI,
		acknowledgement exported.Acknowledgement,
	) error

	AcknowledgePacket(
		ctx sdk.Context,
		packet channeltypes.Packet,
		acknowledgement []byte,
		proof []byte,
		proofHeight exported.Height,
	) (string, error)

	TimeoutPacket(
		ctx sdk.Context,
		packet channeltypes.Packet,
		proof []byte,
		proofHeight exported.Height,
		nextSequenceRecv uint64,
	) (string, error)
}
