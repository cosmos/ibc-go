package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// WriteAcknowledgement writes the packet execution acknowledgement to the state,
// which will be verified by the counterparty chain using AcknowledgePacket. This is for asynchronous acks

func (k Keeper) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
	acknowledgement []byte,
) error {
	// retrieve the forward relayer that was stored in `onRecvPacket`
	// relayer = privateStore.get(forwardRelayerPath(packet))
	// ack = constructIncentivizedAck(acknowledgment, relayer)
	// ack_bytes marshal(ack)
	return k.channelKeeper.WriteAcknowledgement(ctx, chanCap, packet, acknowledgement)
}
