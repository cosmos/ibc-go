package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	icqtypes "github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// OnChanOpenTry performs basic validation of the ICA channel
// and registers a new interchain account (if it doesn't exist).
// The version returned will include the registered interchain
// account address.
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

	if portID != icqtypes.PortID {
		return "", sdkerrors.Wrapf(icqtypes.ErrInvalidHostPort, "expected %s, got %s", icqtypes.PortID, portID)
	}

	if !strings.HasPrefix(counterparty.PortId, icqtypes.PortPrefix) {
		return "", sdkerrors.Wrapf(icqtypes.ErrInvalidControllerPort, "expected %s{owner-account-address}, got %s", icqtypes.PortPrefix, counterparty.PortId)
	}

	// On the host chain the capability may only be claimed during the OnChanOpenTry
	// The capability being claimed in OpenInit is for a controller chain (the port is different)
	if err := k.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", sdkerrors.Wrapf(err, "failed to claim capability for channel %s on port %s", channelID, portID)
	}

	return string(icqtypes.Version), nil
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
