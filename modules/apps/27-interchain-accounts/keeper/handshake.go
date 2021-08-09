package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// OnChanOpenInit performs basic validation of channel initialization.
// The channel order must be ORDERED, the counterparty port identifier
// must be the host chain representation as defined in the types package,
// the channel version must be equal to the version in the types package,
// there must not be an active channel for the specfied port identifier,
// and the interchain accounts module must be able to claim the channel
// capability.
//
// Controller Chain
func (k Keeper) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) error {
	if order != channeltypes.ORDERED {
		return sdkerrors.Wrapf(channeltypes.ErrInvalidChannelOrdering, "invalid channel ordering: %s, expected %s", order.String(), channeltypes.ORDERED.String())
	}
	if counterparty.PortId != types.PortID {
		return sdkerrors.Wrapf(porttypes.ErrInvalidPort, "counterparty port-id must be '%s', (%s != %s)", types.PortID, counterparty.PortId, types.PortID)
	}
	if version != types.Version {
		return sdkerrors.Wrapf(channeltypes.ErrInvalidChannelVersion, "channel version must be '%s' (%s != %s)", types.Version, version, types.Version)
	}

	existingChannelID, found := k.GetActiveChannel(ctx, portID)
	if found {
		return sdkerrors.Wrapf(porttypes.ErrInvalidPort, "existing active channel (%s) for portID (%s)", existingChannelID, portID)
	}

	// Claim channel capability passed back by IBC module
	if err := k.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, err.Error())
	}

	return nil
}

// OnChanOpenTry performs basic validation of the ICA channel
// and registers a new interchain account (if it doesn't exist).
//
// Host Chain
func (k Keeper) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version,
	counterpartyVersion string,
) error {
	if order != channeltypes.ORDERED {
		return sdkerrors.Wrapf(channeltypes.ErrInvalidChannelOrdering, "invalid channel ordering: %s, expected %s", order.String(), channeltypes.ORDERED.String())
	}
	if version != types.Version {
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "got: %s, expected %s", version, types.Version)
	}
	if counterpartyVersion != types.Version {
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: %s, expected %s", counterpartyVersion, types.Version)
	}

	// On the host chain the capability may only be claimed during the OnChanOpenTry
	// The capability being claimed in OpenInit is for a controller chain (the port is different)
	if err := k.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return err
	}

	// Register interchain account if it does not already exist
	k.RegisterInterchainAccount(ctx, counterparty.PortId)
	return nil
}

func (k Keeper) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyVersion string,
) error {
	k.SetActiveChannel(ctx, portID, channelID)

	return nil
}

// Set active channel
func (k Keeper) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {

	return nil
}

// May want to use these for re-opening a channel when it is closed
//// OnChanCloseInit implements the IBCModule interface
//func (am AppModule) OnChanCloseInit(
//	ctx sdk.Context,
//	portID,
//	channelID string,
//) error {
//	// Disallow user-initiated channel closing for transfer channels
//	return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
//}

//// OnChanCloseConfirm implements the IBCModule interface
//func (am AppModule) OnChanCloseConfirm(
//	ctx sdk.Context,
//	portID,
//	channelID string,
//) error {
//	return nil
//}
