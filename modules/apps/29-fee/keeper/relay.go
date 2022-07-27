package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"
)

// SendPacket wraps IBC ChannelKeeper's SendPacket function
func (k Keeper) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
	return k.ics4Wrapper.SendPacket(ctx, chanCap, packet)
}

// WriteAcknowledgement wraps IBC ChannelKeeper's WriteAcknowledgement function
// ICS29 WriteAcknowledgement is used for asynchronous acknowledgements
func (k Keeper) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, acknowledgement ibcexported.Acknowledgement) error {
	if !k.IsFeeEnabled(ctx, packet.GetDestPort(), packet.GetDestChannel()) {
		// ics4Wrapper may be core IBC or higher-level middleware
		return k.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, acknowledgement)
	}

	packetID := channeltypes.NewPacketId(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

	// retrieve the forward relayer that was stored in `onRecvPacket`
	relayer, found := k.GetRelayerAddressForAsyncAck(ctx, packetID)
	if !found {
		return sdkerrors.Wrapf(types.ErrRelayerNotFoundForAsyncAck, "no relayer address stored for async acknowledgement for packet with portID: %s, channelID: %s, sequence: %d", packetID.PortId, packetID.ChannelId, packetID.Sequence)
	}

	// it is possible that a relayer has not registered a counterparty address.
	// if there is no registered counterparty address then write acknowledgement with empty relayer address and refund recv_fee.
	forwardRelayer, _ := k.GetCounterpartyPayeeAddress(ctx, relayer, packet.GetDestChannel())

	ack := types.NewIncentivizedAcknowledgement(forwardRelayer, acknowledgement.Acknowledgement(), acknowledgement.Success())

	k.DeleteForwardRelayerAddress(ctx, packetID)

	// ics4Wrapper may be core IBC or higher-level middleware
	return k.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

// GetAppVersion returns the underlying application version.
func (k Keeper) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	version, found := k.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
	if !found {
		return "", false
	}

	if !k.IsFeeEnabled(ctx, portID, channelID) {
		return version, true
	}

	var metadata types.Metadata
	if err := types.ModuleCdc.UnmarshalJSON([]byte(version), &metadata); err != nil {
		panic(fmt.Errorf("unable to unmarshal metadata for fee enabled channel: %w", err))
	}

	return metadata.AppVersion, true
}
