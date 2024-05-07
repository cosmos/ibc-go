package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
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
	logger := k.Logger(ctx)
	if portID != icatypes.HostPortID {
		return "", sdkerrors.Wrapf(icatypes.ErrInvalidHostPort, "expected %s, got %s", icatypes.HostPortID, portID)
	}

<<<<<<< HEAD
	var metadata icatypes.Metadata
	if err := icatypes.ModuleCdc.UnmarshalJSON([]byte(counterpartyVersion), &metadata); err != nil {
		return "", sdkerrors.Wrapf(icatypes.ErrUnknownDataType, "cannot unmarshal ICS-27 interchain accounts metadata")
=======
	metadata, err := icatypes.MetadataFromVersion(counterpartyVersion)
	if err != nil {
		// Propose the default metadata if the counterparty version is invalid
		connection, err := k.channelKeeper.GetConnection(ctx, connectionHops[0])
		if err != nil {
			return "", errorsmod.Wrapf(err, "failed to retrieve connection %s", connectionHops[0])
		}

		logger.Debug("counterparty version is invalid, proposing default metadata")
		metadata = icatypes.NewDefaultMetadata(connection.Counterparty.ConnectionId, connectionHops[0])
>>>>>>> 3b3ecc5a (imp(apps): allow one sided fee middleware handshakes to complete (#6253))
	}

	if err := icatypes.ValidateHostMetadata(ctx, k.channelKeeper, connectionHops, metadata); err != nil {
		return "", err
	}

	activeChannelID, found := k.GetActiveChannelID(ctx, connectionHops[0], counterparty.PortId)
	if found {
		channel, found := k.channelKeeper.GetChannel(ctx, portID, activeChannelID)
		if !found {
			panic(fmt.Sprintf("active channel mapping set for %s but channel does not exist in channel store", activeChannelID))
		}

		if channel.State != channeltypes.CLOSED {
			return "", errorsmod.Wrapf(icatypes.ErrActiveChannelAlreadySet, "existing active channel %s for portID %s must be %s", activeChannelID, portID, channeltypes.CLOSED)
		}

		// if a channel is being reopened, we allow the controller to propose new fields
		// which are not exactly the same as the previous. The provided address will
		// be overwritten with the correct one before the metadata is returned.
	}

	// On the host chain the capability may only be claimed during the OnChanOpenTry
	// The capability being claimed in OpenInit is for a controller chain (the port is different)
	if err := k.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", sdkerrors.Wrapf(err, "failed to claim capability for channel %s on port %s", channelID, portID)
	}

	var (
		accAddress sdk.AccAddress
		err        error
	)

	interchainAccAddr, found := k.GetInterchainAccountAddress(ctx, metadata.HostConnectionId, counterparty.PortId)
	if found {
		// reopening an interchain account
		k.Logger(ctx).Info("reopening existing interchain account", "address", interchainAccAddr)
		accAddress = sdk.MustAccAddressFromBech32(interchainAccAddr)
		if _, ok := k.accountKeeper.GetAccount(ctx, accAddress).(*icatypes.InterchainAccount); !ok {
			return "", sdkerrors.Wrapf(icatypes.ErrInvalidAccountReopening, "existing account address %s, does not have interchain account type", accAddress)
		}

	} else {
		accAddress, err = k.createInterchainAccount(ctx, metadata.HostConnectionId, counterparty.PortId)
		if err != nil {
			return "", err
		}
		k.Logger(ctx).Info("successfully created new interchain account", "host-connection-id", metadata.HostConnectionId, "port-id", counterparty.PortId, "address", accAddress)
	}

	metadata.Address = accAddress.String()
	versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
	if err != nil {
		return "", err
	}

	return string(versionBytes), nil
}

// OnChanOpenConfirm completes the handshake process by setting the active channel in state on the host chain
func (k Keeper) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	channel, found := k.channelKeeper.GetChannel(ctx, portID, channelID)
	if !found {
		return sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "failed to retrieve channel %s on port %s", channelID, portID)
	}

	// It is assumed the controller chain will not allow multiple active channels to be created for the same connectionID/portID
	// If the controller chain does allow multiple active channels to be created for the same connectionID/portID,
	// disallowing overwriting the current active channel guarantees the channel can no longer be used as the controller
	// and host will disagree on what the currently active channel is
	k.SetActiveChannelID(ctx, channel.ConnectionHops[0], channel.Counterparty.PortId, channelID)

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
