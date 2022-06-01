package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
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
	if order != channeltypes.UNORDERED {
		return sdkerrors.Wrapf(channeltypes.ErrInvalidChannelOrdering, "expected %s channel, got %s", channeltypes.UNORDERED, order)
	}

	if !strings.HasPrefix(portID, types.PortPrefix) {
		return sdkerrors.Wrapf(types.ErrInvalidControllerPort, "expected %s{owner-account-address}, got %s", types.PortPrefix, portID)
	}

	if counterparty.PortId != types.PortID {
		return sdkerrors.Wrapf(types.ErrInvalidHostPort, "expected %s, got %s", types.PortID, counterparty.PortId)
	}

	return nil
}

// OnChanOpenAck sets the active channel if it's valid.
func (k Keeper) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyVersion string,
) error {
	if portID == types.PortID {
		return sdkerrors.Wrapf(types.ErrInvalidControllerPort, "portID cannot be host chain port ID: %s", types.PortID)
	}

	if !strings.HasPrefix(portID, types.PortPrefix) {
		return sdkerrors.Wrapf(types.ErrInvalidControllerPort, "expected %s{owner-account-address}, got %s", types.PortPrefix, portID)
	}

	return nil
}

// OnChanOpenTry performs basic validation of the ICQ channel.
func (k Keeper) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if order != channeltypes.UNORDERED {
		return "", sdkerrors.Wrapf(channeltypes.ErrInvalidChannelOrdering, "expected %s channel, got %s", channeltypes.UNORDERED, order)
	}

	if portID != types.PortID {
		return "", sdkerrors.Wrapf(types.ErrInvalidHostPort, "expected %s, got %s", types.PortID, portID)
	}

	if !strings.HasPrefix(counterparty.PortId, types.PortPrefix) {
		return "", sdkerrors.Wrapf(types.ErrInvalidControllerPort, "expected %s{owner-account-address}, got %s", types.PortPrefix, counterparty.PortId)
	}

	// On the host chain the capability may only be claimed during the OnChanOpenTry
	// The capability being claimed in OpenInit is for a controller chain (the port is different)
	if err := k.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", sdkerrors.Wrapf(err, "failed to claim capability for channel %s on port %s", channelID, portID)
	}

	return string(types.Version), nil
}

// OnChanOpenConfirm completes the handshake process by setting the active channel in state on the host chain
func (k Keeper) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {

	return nil
}

// OnChanCloseConfirm removes the active channel stored in state
func (k Keeper) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {

	return nil
}
