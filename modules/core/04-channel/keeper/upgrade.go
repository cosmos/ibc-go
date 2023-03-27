package keeper

import (
	"reflect"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	portkeeper "github.com/cosmos/ibc-go/v7/modules/core/05-port/keeper"
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

// ChanUpgradeTry is called by a module to accept the first step of a channel upgrade
// handshake initiated by a module on another chain.
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
	proofChannel []byte,
	proofUpgradeTimeout []byte,
	proofUpgradeSequence []byte,
	proofHeight clienttypes.Height,
) (uint64, error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if !collections.Contains(channel.State, []types.State{types.OPEN, types.INITUPGRADE}) {
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)) {
		return 0, errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", portID, channelID)
	}

	if proposedUpgradeChannel.State != types.TRYUPGRADE || proposedUpgradeChannel.Counterparty.PortId != channel.Counterparty.PortId ||
		proposedUpgradeChannel.Counterparty.ChannelId != channel.Counterparty.ChannelId {
		return 0, errorsmod.Wrap(types.ErrInvalidChannel, "proposed channel upgrade is invalid")
	}

	if !channel.Ordering.SubsetOf(proposedUpgradeChannel.Ordering) {
		return 0, errorsmod.Wrap(types.ErrInvalidChannelOrdering, "channel ordering must be a subset of the new ordering")
	}

	if counterpartyChannel.Ordering != proposedUpgradeChannel.Ordering {
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelOrdering, "channel ordering of counterparty channel and proposed channel must be equal")
	}

	connection, err := k.GetConnection(ctx, proposedUpgradeChannel.ConnectionHops[0])
	if err != nil {
		return 0, err
	}

	if connection.GetCounterparty().GetConnectionID() != counterpartyChannel.ConnectionHops[0] {
		return 0, err
	}

	if err := k.connectionKeeper.VerifyChannelState(ctx, connection, proofHeight, proofChannel, proposedUpgradeChannel.Counterparty.PortId,
		proposedUpgradeChannel.Counterparty.ChannelId, counterpartyChannel); err != nil {
		return 0, err
	}

	// upgradeTimeout := types.UpgradeTimeout{TimeoutHeight: timeoutHeight, TimeoutTimestamp: timeoutTimestamp}
	// if err := k.connectionKeeper.VerifyChannelUpgradeTimeout(ctx, connection, proofHeight, proofUpgradeTimeout, proposedUpgradeChannel.Counterparty.PortId,
	// 	proposedUpgradeChannel.Counterparty.ChannelId, upgradeTimeout); err != nil {
	// 	return err
	// }

	if err := k.connectionKeeper.VerifyChannelUpgradeSequence(ctx, connection, proofHeight, proofUpgradeSequence, proposedUpgradeChannel.Counterparty.PortId,
		proposedUpgradeChannel.Counterparty.ChannelId, counterpartyUpgradeSequence); err != nil {
		return 0, err
	}

	upgradeSequence := uint64(1)
	if seq, found := k.GetUpgradeSequence(ctx, portID, channelID); found {
		upgradeSequence = seq + 1
	}

	// if the counterparty upgrade sequence is ahead then fast forward so both channel ends are using the same sequence for the current upgrade
	if counterpartyUpgradeSequence >= upgradeSequence {
		upgradeSequence = counterpartyUpgradeSequence
		k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)
	} else {
		errorReceipt := types.ErrorReceipt{
			Sequence: upgradeSequence,
			Error:    errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "upgrade sequence %d was not smaller than the counter party chain upgrade sequence %d", upgradeSequence, counterpartyUpgradeSequence).Error(),
		}

		// the upgrade sequence is incremented so both sides start the next upgrade with a fresh sequence.
		upgradeSequence++

		k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
		k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)

		// TODO: emit error receipt events

		return 0, errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "upgrade aborted, error receipt written for upgrade sequence: %d", errorReceipt.GetSequence())
	}

	switch channel.State {
	case types.OPEN:
		// this is first message in upgrade handshake on this chain so we must store original channel in restore channel path
		// in case we need to restore channel later.
		k.SetUpgradeRestoreChannel(ctx, portID, channelID, channel)
	case types.INITUPGRADE:
		// if there is a crossing hello, i.e an UpgradeInit has been called on both channelEnds,
		// then we must ensure that the proposedUpgrade by the counterparty is the same as the currentChannel
		// except for the channel state (upgrade channel will be in TRYUPGRADE and current channel will be in INITUPGRADE)
		// if the proposed upgrades on either side are incompatible, then we will restore the channel and cancel the upgrade.
		channel.State = types.TRYUPGRADE
		k.SetChannel(ctx, portID, channelID, channel)

		if !reflect.DeepEqual(channel, proposedUpgradeChannel) {
			// TODO: restore channel, log and emit events
			k.RestoreChannel(ctx, portID, channelID, upgradeSequence)
			return 0, nil
		}
	default:
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s] but got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	return upgradeSequence, nil
}

// WriteUpgradeTryChannel writes a channel which has successfully passed the UpgradeTry handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTryChannel(
	ctx sdk.Context,
	portID,
	channelID string,
	upgradeSequence uint64,
	channelUpgrade types.Channel,
) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-try")

	k.SetChannel(ctx, portID, channelID, channelUpgrade)
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.TRYUPGRADE.String())

	// TODO: add events
	// emitChannelUpgradeTryEvent(ctx, portID, channelID, upgradeSequence, channelUpgrade)
}

func (k Keeper) RestoreChannel(ctx sdk.Context, portID, channelID string, upgradeSequence uint64) {
	errorReceipt := types.ErrorReceipt{
		Sequence: upgradeSequence,
		Error:    "",
	}

	k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)

	channel, found := k.GetUpgradeRestoreChannel(ctx, portID, channelID)
	if !found {
		panic("todo unreachable state")
	}

	k.SetChannel(ctx, portID, channelID, channel)
	k.DeleteUpgradeRestoreChannel(ctx, portID, channelID)
	k.DeleteUpgradeTimeout(ctx, portID, channelID)

	module, _, err := k.LookupModuleByChannel(ctx, portID, channelID)
	if err != nil {
		panic("todo")
	}

	// TODO: add some methods to expected keeper
	portKeeper, ok := k.portKeeper.(portkeeper.Keeper)
	if !ok {
		panic("todo")
	}

	cbs, found := portKeeper.Router.GetRoute(module)
	if !found {
		panic("todo")
	}

	if err := cbs.OnChanUpgradeRestore(ctx, portID, channelID); err != nil {
		panic(err)
	}
}
