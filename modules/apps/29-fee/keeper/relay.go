package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
)

// SendPacket wraps IBC ChannelKeeper's SendPacket function
func (k Keeper) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
	return k.ics4Wrapper.SendPacket(ctx, chanCap, packet)
}

// WriteAcknowledgement wraps IBC ChannelKeeper's WriteAcknowledgement function
// ICS29 WriteAcknowledgement is used for asynchronous acknowledgements
func (k Keeper) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, acknowledgement []byte) error {
	// retrieve the forward relayer that was stored in `onRecvPacket`
	packetId := channeltypes.NewPacketId(packet.GetSourceChannel(), packet.GetSourcePort(), packet.GetSequence())
	relayer, found := k.GetForwardRelayerAddress(ctx, packetId)

	if found {
		// delete the forward relayer address as it is no longer used
		k.DeleteForwardRelayerAddress(ctx, packetId)
	}

	ack := types.NewIncentivizedAcknowledgement(relayer, acknowledgement)
	bz := ack.Acknowledgement()

	// ics4Wrapper may be core IBC or higher-level middleware
	return k.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, bz)
}
