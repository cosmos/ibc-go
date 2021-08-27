package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// SendPacket is called by a module in order to send an IBC packet on a channel
// end owned by the calling module to the corresponding module on the counterparty
// chain.
func (k Keeper) SendPacket(
	ctx sdk.Context,
	channelCap *capabilitytypes.Capability,
	packet exported.PacketI,
) error {
	// will this work this ics20? SendTransfer
	// if channelKeeper === ics20 then sendTransfer, otherwise SendPacket
	return k.channelKeeper.SendPacket(ctx, channelCap, packet)
}

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
