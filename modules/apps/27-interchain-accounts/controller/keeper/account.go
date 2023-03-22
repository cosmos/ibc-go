package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// RegisterInterchainAccount is the entry point to registering an interchain account:
// - It generates a new port identifier using the provided owner string, binds to the port identifier and claims the associated capability.
// - Callers are expected to provide the appropriate application version string.
// - For example, this could be an ICS27 encoded metadata type or an ICS29 encoded metadata type with a nested application version.
// - A new MsgChannelOpenInit is routed through the MsgServiceRouter, executing the OnOpenChanInit callback stack as configured.
// - An error is returned if the port identifier is already in use. Gaining access to interchain accounts whose channels
// have closed cannot be done with this function. A regular MsgChannelOpenInit must be used.
func (k Keeper) RegisterInterchainAccount(ctx sdk.Context, connectionID, owner, version string) error {
	portID, err := icatypes.NewControllerPortID(owner)
	if err != nil {
		return err
	}

	// if there is an active channel for this portID / connectionID return an error
	activeChannelID, found := k.GetOpenActiveChannel(ctx, connectionID, portID)
	if found {
		return sdkerrors.Wrapf(icatypes.ErrActiveChannelAlreadySet, "existing active channel %s for portID %s on connection %s for owner %s", activeChannelID, portID, connectionID, owner)
	}

	switch {
	case k.portKeeper.IsBound(ctx, portID) && !k.IsBound(ctx, portID):
		return sdkerrors.Wrapf(icatypes.ErrPortAlreadyBound, "another module has claimed capability for and bound port with portID: %s", portID)
	case !k.portKeeper.IsBound(ctx, portID):
		cap := k.BindPort(ctx, portID)
		if err := k.ClaimCapability(ctx, cap, host.PortPath(portID)); err != nil {
			return sdkerrors.Wrapf(err, "unable to bind to newly generated portID: %s", portID)
		}
	}

	msg := channeltypes.NewMsgChannelOpenInit(portID, version, channeltypes.ORDERED, []string{connectionID}, icatypes.PortID, authtypes.NewModuleAddress(icatypes.ModuleName).String())
	handler := k.msgRouter.Handler(msg)

	res, err := handler(ctx, msg)
	if err != nil {
		return err
	}

	// NOTE: The sdk msg handler creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(res.GetEvents())

	return nil
}
<<<<<<< HEAD
=======

// registerInterchainAccount registers an interchain account, returning the channel id of the MsgChannelOpenInitResponse
// and an error if one occurred.
func (k Keeper) registerInterchainAccount(ctx sdk.Context, connectionID, portID, version string) (string, error) {
	// if there is an active channel for this portID / connectionID return an error
	activeChannelID, found := k.GetOpenActiveChannel(ctx, connectionID, portID)
	if found {
		return "", errorsmod.Wrapf(icatypes.ErrActiveChannelAlreadySet, "existing active channel %s for portID %s on connection %s", activeChannelID, portID, connectionID)
	}

	switch {
	case k.portKeeper.IsBound(ctx, portID) && !k.HasCapability(ctx, portID):
		return "", errorsmod.Wrapf(icatypes.ErrPortAlreadyBound, "another module has claimed capability for and bound port with portID: %s", portID)
	case !k.portKeeper.IsBound(ctx, portID):
		capability := k.BindPort(ctx, portID)
		if err := k.ClaimCapability(ctx, capability, host.PortPath(portID)); err != nil {
			return "", errorsmod.Wrapf(err, "unable to bind to newly generated portID: %s", portID)
		}
	}

	msg := channeltypes.NewMsgChannelOpenInit(portID, version, channeltypes.ORDERED, []string{connectionID}, icatypes.HostPortID, authtypes.NewModuleAddress(icatypes.ModuleName).String())
	handler := k.msgRouter.Handler(msg)
	res, err := handler(ctx, msg)
	if err != nil {
		return "", err
	}

	events := res.GetEvents()
	k.Logger(ctx).Debug("emitting interchain account registration events", logging.SdkEventsToLogArguments(events))

	// NOTE: The sdk msg handler creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(events)

	firstMsgResponse := res.MsgResponses[0]
	channelOpenInitResponse, ok := firstMsgResponse.GetCachedValue().(*channeltypes.MsgChannelOpenInitResponse)
	if !ok {
		return "", errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to covert %T message response to %T", firstMsgResponse.GetCachedValue(), &channeltypes.MsgChannelOpenInitResponse{})
	}

	return channelOpenInitResponse.ChannelId, nil
}
>>>>>>> 5a67efc4 (chore: fix linter warnings (#3311))
