package keeper

import (
	"reflect"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
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
	proposedUpgradeChannel types.Channel,
	counterpartyTimeoutHeight clienttypes.Height,
	counterpartyTimeoutTimestamp uint64,
) (uint64, error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.OPEN {
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.OPEN, channel.State)
	}

	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)) {
		return 0, errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", portID, channelID)
	}

	if proposedUpgradeChannel.Counterparty.PortId != channel.Counterparty.PortId ||
		proposedUpgradeChannel.Counterparty.ChannelId != channel.Counterparty.ChannelId {
		return 0, errorsmod.Wrap(types.ErrInvalidCounterparty, "counterparty port ID and channel ID cannot be upgraded")
	}

	if !channel.Ordering.SubsetOf(proposedUpgradeChannel.Ordering) {
		return 0, errorsmod.Wrap(types.ErrInvalidChannelOrdering, "channel ordering must be a subset of the new ordering")
	}

	upgradeSequence := uint64(1)
	if seq, found := k.GetUpgradeSequence(ctx, portID, channelID); found {
		upgradeSequence = seq + 1
	}

	upgradeTimeout := types.UpgradeTimeout{
		TimeoutHeight:    counterpartyTimeoutHeight,
		TimeoutTimestamp: counterpartyTimeoutTimestamp,
	}

	k.SetUpgradeRestoreChannel(ctx, portID, channelID, channel)
	k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)
	k.SetUpgradeTimeout(ctx, portID, channelID, upgradeTimeout)

	return upgradeSequence, nil
}

// WriteUpgradeInitChannel writes a channel which has successfully passed the UpgradeInit handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeInitChannel(
	ctx sdk.Context,
	portID,
	channelID string,
	upgradeSequence uint64,
	channelUpgrade types.Channel,
) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-init")

	k.SetChannel(ctx, portID, channelID, channelUpgrade)
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.INITUPGRADE.String())

	emitChannelUpgradeInitEvent(ctx, portID, channelID, upgradeSequence, channelUpgrade)
}

func (k Keeper) ChanUpgradeTry(
	ctx sdk.Context,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterpartyChannel types.Channel,
	counterpartyUpgradeSequence uint64,
	proposedUpgradeChannel types.Channel,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	channelProof []byte,
	upgradeTimeoutProof []byte,
	upgradeSequenceProof []byte,
	proofHeight clienttypes.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if !collections.Contains(channel.State, []types.State{types.OPEN, types.INITUPGRADE}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)) {
		return errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", portID, channelID)
	}

	if proposedUpgradeChannel.State != types.TRYUPGRADE || proposedUpgradeChannel.Counterparty.PortId != channel.Counterparty.PortId ||
		proposedUpgradeChannel.Counterparty.ChannelId != channel.Counterparty.ChannelId {
		return errorsmod.Wrap(types.ErrInvalidChannel, "proposed channel upgrade is invalid")
	}

	if strings.TrimSpace(proposedUpgradeChannel.Version) == "" {
		return errorsmod.Wrap(types.ErrInvalidChannelVersion, "channel version must be not be empty")
	}

	if !channel.Ordering.SubsetOf(proposedUpgradeChannel.Ordering) {
		return errorsmod.Wrap(types.ErrInvalidChannelOrdering, "channel ordering must be a subset of the new ordering")
	}

	if counterpartyChannel.Ordering != proposedUpgradeChannel.Ordering {
		return errorsmod.Wrapf(types.ErrInvalidChannelOrdering, "channel ordering of counterparty channel and proposed channel must be equal")
	}

	connection, err := k.GetConnection(ctx, proposedUpgradeChannel.ConnectionHops[0])
	if err != nil {
		return err
	}

	if err := k.connectionKeeper.VerifyChannelState(ctx, connection, proofHeight, channelProof, portID, channelID, counterpartyChannel); err != nil {
		return err
	}

	upgradeTimeout := types.UpgradeTimeout{
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
	}

	// TODO
	_ = upgradeTimeout
	//abortTransactionUnless(verifyChannelUpgradeTimeout(connection, proofHeight, proofUpgradeTimeout, currentChannel.counterpartyPortIdentifier, currentChannel.counterpartyChannelIdentifier, upgradeTimeout))

	if err := k.connectionKeeper.VerifyChannelUpgradeSequence(ctx, connection, proofHeight, upgradeSequenceProof, portID, channelID, counterpartyUpgradeSequence); err != nil {
		return err
	}

	currentSequence, found := k.GetUpgradeSequence(ctx, portID, channelID)
	if counterpartyUpgradeSequence > currentSequence {
		currentSequence = counterpartyUpgradeSequence
		k.SetUpgradeSequence(ctx, portID, channelID, counterpartyUpgradeSequence)
	} else {
		errorReceipt := types.ErrorReceipt{
			Sequence: currentSequence,
			Error:    errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "upgrade sequence %d was not smaller than the counter party chain upgrade sequence %d", currentSequence, counterpartyUpgradeSequence).Error(),
		}

		// TODO: provableStore.set(channelUpgradeErrorPath(portIdentifier, channelIdentifier), errorReceipt)
		_ = errorReceipt

		k.SetUpgradeSequence(ctx, portID, channelID, currentSequence+1)
	}

	switch channel.State {
	case types.OPEN:
		// this is first message in upgrade handshake on this chain so we must store original channel in restore channel path
		// in case we need to restore channel later.
		k.SetUpgradeRestoreChannel(ctx, portID, channelID, channel)
	case types.INITUPGRADE:
		// if there is a crossing hello, ie an UpgradeInit has been called on both channelEnds,
		// then we must ensure that the proposedUpgrade by the counterparty is the same as the currentChannel
		// except for the channel state (upgrade channel will be in TRYUPGRADE and current channel will be in INITUPGRADE)
		// if the proposed upgrades on either side are incompatible, then we will restore the channel and cancel the upgrade.
		channel.State = types.TRYUPGRADE
		k.SetChannel(ctx, portID, channelID, channel)

		if !reflect.DeepEqual(channel, proposedUpgradeChannel) {
			// TODO: restore channel
			// log that the channel was restored?
			return nil
		}
	default:
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s] but got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	return nil
}
