package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

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
	//TODO:
	// check version string
	if order != channeltypes.ORDERED {
		return sdkerrors.Wrapf(channeltypes.ErrInvalidChannelOrdering, "invalid channel ordering: %s, expected %s", order.String(), channeltypes.ORDERED.String())
	}

	// Claim channel capability passed back by IBC module
	if err := k.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, err.Error())
	}

	return nil
}

// register account (if it doesn't exist)
// check if counterpary version is the same
// TODO: remove ics27-1 hardcoded
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

	// TODO: Check counterparty version
	// if counterpartyVersion != types.Version {
	// 	return sdkerrors.Wrapf(porttypes.ErrInvalidPort, "invalid counterparty version: %s, expected %s", counterpartyVersion, "ics20-1")
	// }

	// Claim channel capability passed back by IBC module
	if err := k.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, err.Error())
	}

	// Register interchain account if it does not already exist
	_, _ = k.RegisterInterchainAccount(ctx, counterparty.PortId)
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
