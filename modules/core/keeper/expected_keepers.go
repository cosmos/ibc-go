package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type ChanKeeperI interface {
	RecvPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet channeltypes.Packet, proof []byte, proofHeight exported.Height) error
	WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet exported.PacketI, acknowledgement exported.Acknowledgement) error
	AcknowledgePacket(
		ctx sdk.Context,
		chanCap *capabilitytypes.Capability,
		packet channeltypes.Packet,
		acknowledgement []byte,
		proof []byte,
		proofHeight exported.Height,
	) error
	TimeoutPacket(
		ctx sdk.Context,
		packet channeltypes.Packet,
		proof []byte,
		proofHeight exported.Height,
		nextSequenceRecv uint64,
	) error
}
