package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

// ChanUpgradeInit is called by a module to initiate a channel upgrade handshake with
// a module on another chain.
func (k Keeper) ChanUpgradeInit(
	ctx sdk.Context,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	propsedUpgradeChannel types.Channel,
	counterpartyTimeoutHeight clienttypes.Height,
	counterpartyTimeoutTimestamp uint64,
) (uint64, error) {
	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)) {
		return 0, errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", portID, channelID)
	}

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.OPEN {
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.OPEN, channel.State)
	}

	if propsedUpgradeChannel.State != types.INITUPGRADE &&
		propsedUpgradeChannel.Counterparty.PortId != channel.Counterparty.PortId &&
		propsedUpgradeChannel.Counterparty.ChannelId != channel.Counterparty.ChannelId {
		return 0, errorsmod.Wrap(types.ErrInvalidChannel, "TODO: update error")
	}

	if propsedUpgradeChannel.Version == "" {
		return 0, errorsmod.Wrap(types.ErrInvalidChannelVersion, "channel version must be not be empty")
	}

	if !channel.Ordering.SubsetOf(propsedUpgradeChannel.Ordering) {
		return 0, errorsmod.Wrap(types.ErrInvalidChannelOrdering, "channel ordering must be a subset of the new ordering")
	}

	sequence, found := k.GetUpgradeSequence(ctx, portID, channelID)
	if !found {
		sequence = 1
		k.SetUpgradeSequence(ctx, portID, channelID, sequence)
	} else {
		sequence = sequence + 1
		k.SetUpgradeSequence(ctx, portID, channelID, sequence)
	}

	upgradeTimeout := types.UpgradeTimeout{
		TimeoutHeight:    counterpartyTimeoutHeight,
		TimeoutTimestamp: counterpartyTimeoutTimestamp,
	}

	k.SetUpgradeTimeout(ctx, portID, channelID, upgradeTimeout)
	k.SetUpgradeRestoreChannel(ctx, portID, channelID, channel)

	return sequence, nil
}

// WriteUpgradeInitChannel writes a channel which has successfully passed the UpgradeInit handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeInitChannel(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedUpgradeChannel types.Channel,
) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-init")

	k.SetChannel(ctx, portID, channelID, proposedUpgradeChannel)
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", "NONE", "new-state", "INITUPGRADE")

	// emitChannelOpenInitEvent(ctx, portID, channelID, channel)
}
