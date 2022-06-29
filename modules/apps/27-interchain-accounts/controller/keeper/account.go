package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	icatypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v4/modules/core/24-host"
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

	msg := channeltypes.NewMsgChannelOpenInit(portID, version, channeltypes.ORDERED, []string{connectionID}, icatypes.PortID, icatypes.ModuleName)
	handler := k.msgRouter.Handler(msg)

	res, err := handler(ctx, msg)
	if err != nil {
		return err
	}

	// NOTE: The sdk msg handler creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(res.GetEvents())

	return nil
}
