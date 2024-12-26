package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

// RegisterInterchainAccount is the entry point to registering an interchain account:
// - It generates a new port identifier using the provided owner string.
// - Callers are expected to provide the appropriate application version string.
// - For example, this could be an ICS27 encoded metadata type or an ICS29 encoded metadata type with a nested application version.
// - A new MsgChannelOpenInit is routed through the MsgServiceRouter, executing the OnOpenChanInit callback stack as configured.
// - An error is returned if the port identifier is already in use. Gaining access to interchain accounts whose channels
// have closed cannot be done with this function. A regular MsgChannelOpenInit must be used.
//
// Deprecated: this is a legacy API that is only intended to function correctly in workflows where an underlying authentication application has been set.
// Calling this API will result in all packet callbacks being routed to the underlying application.

// Please use MsgRegisterInterchainAccount for use cases which do not need to route to an underlying application.

// Prior to v6.x.x of ibc-go, the controller module was only functional as middleware, with authentication performed
// by the underlying application. For a full summary of the changes in v6.x.x, please see ADR009.
// This API will be removed in later releases.
func (k Keeper) RegisterInterchainAccount(ctx context.Context, connectionID, owner, version string, ordering channeltypes.Order) error {
	portID, err := icatypes.NewControllerPortID(owner)
	if err != nil {
		return err
	}

	if k.IsMiddlewareDisabled(ctx, portID, connectionID) && !k.IsActiveChannelClosed(ctx, connectionID, portID) {
		return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel is already active or a handshake is in flight")
	}

	k.SetMiddlewareEnabled(ctx, portID, connectionID)

	// use ORDER_UNORDERED as default in case ordering is NONE
	if ordering == channeltypes.NONE {
		ordering = channeltypes.UNORDERED
	}

	_, err = k.registerInterchainAccount(ctx, connectionID, portID, version, ordering)
	if err != nil {
		return err
	}

	return nil
}

// registerInterchainAccount registers an interchain account, returning the channel id of the MsgChannelOpenInitResponse
// and an error if one occurred.
func (k Keeper) registerInterchainAccount(ctx context.Context, connectionID, portID, version string, ordering channeltypes.Order) (string, error) {
	// if there is an active channel for this portID / connectionID return an error
	activeChannelID, found := k.GetOpenActiveChannel(ctx, connectionID, portID)
	if found {
		return "", errorsmod.Wrapf(icatypes.ErrActiveChannelAlreadySet, "existing active channel %s for portID %s on connection %s", activeChannelID, portID, connectionID)
	}

	k.setPort(ctx, portID)

	msg := channeltypes.NewMsgChannelOpenInit(portID, version, ordering, []string{connectionID}, icatypes.HostPortID, authtypes.NewModuleAddress(icatypes.ModuleName).String())
	res, err := k.Environment.MsgRouterService.Invoke(ctx, msg)
	if err != nil {
		return "", err
	}

	chanOpenInitResp, ok := res.(*channeltypes.MsgChannelOpenInitResponse)
	if !ok {
		return "", errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to convert %T message response to %T", res, &channeltypes.MsgChannelOpenInitResponse{})
	}

	return chanOpenInitResp.ChannelId, nil
}
