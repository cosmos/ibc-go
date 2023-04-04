package keeper

import (
	"reflect"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	portkeeper "github.com/cosmos/ibc-go/v7/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
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
) (upgradeSequence uint64, previousVersion string, err error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, "", errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.OPEN {
		return 0, "", errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.OPEN, channel.State)
	}

	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)) {
		return 0, "", errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", portID, channelID)
	}

	// set the restore channel to the current channel and reassign channel state to INITUPGRADE,
	// if the channel == proposedUpgradeChannel then fail fast as no upgradable fields have been modified.
	restoreChannel := channel
	channel.State = types.INITUPGRADE
	if reflect.DeepEqual(channel, proposedUpgradeChannel) {
		return 0, "", errorsmod.Wrap(types.ErrChannelExists, "existing channel end is identical to proposed upgrade channel end")
	}

	if !k.connectionKeeper.HasConnection(ctx, proposedUpgradeChannel.ConnectionHops[0]) {
		return 0, "", errorsmod.Wrapf(connectiontypes.ErrConnectionNotFound, "failed to retrieve connection: %s", proposedUpgradeChannel.ConnectionHops[0])
	}

	if proposedUpgradeChannel.Counterparty.PortId != channel.Counterparty.PortId ||
		proposedUpgradeChannel.Counterparty.ChannelId != channel.Counterparty.ChannelId {
		return 0, "", errorsmod.Wrap(types.ErrInvalidCounterparty, "counterparty port ID and channel ID cannot be upgraded")
	}

	if !channel.Ordering.SubsetOf(proposedUpgradeChannel.Ordering) {
		return 0, "", errorsmod.Wrap(types.ErrInvalidChannelOrdering, "channel ordering must be a subset of the new ordering")
	}

	upgradeSequence = uint64(1)
	if seq, found := k.GetUpgradeSequence(ctx, portID, channelID); found {
		upgradeSequence = seq + 1
	}

	upgradeTimeout := types.UpgradeTimeout{
		TimeoutHeight:    counterpartyTimeoutHeight,
		TimeoutTimestamp: counterpartyTimeoutTimestamp,
	}

	k.SetUpgradeRestoreChannel(ctx, portID, channelID, restoreChannel)
	k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)
	k.SetUpgradeTimeout(ctx, portID, channelID, upgradeTimeout)

	return upgradeSequence, channel.Version, nil
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

	// TODO: don't set the channel directly, instead modify the existing channel and re-assign the modifiable fields.
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
) (upgradeSequence uint64, previousVersion string, err error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, "", errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if !collections.Contains(channel.State, []types.State{types.OPEN, types.INITUPGRADE}) {
		return 0, "", errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)) {
		return 0, "", errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", portID, channelID)
	}

	if proposedUpgradeChannel.Counterparty.PortId != channel.Counterparty.PortId ||
		proposedUpgradeChannel.Counterparty.ChannelId != channel.Counterparty.ChannelId {
		return 0, "", errorsmod.Wrap(types.ErrInvalidChannel, "proposed channel upgrade is invalid")
	}

	if !channel.Ordering.SubsetOf(proposedUpgradeChannel.Ordering) {
		return 0, "", errorsmod.Wrap(types.ErrInvalidChannelOrdering, "channel ordering must be a subset of the new ordering")
	}

	if counterpartyChannel.Ordering != proposedUpgradeChannel.Ordering {
		return 0, "", errorsmod.Wrapf(types.ErrInvalidChannelOrdering, "channel ordering of counterparty channel and proposed channel must be equal")
	}

	connectionEnd, err := k.getUpgradeTryConnectionEnd(ctx, portID, channelID, channel)
	if err != nil {
		return 0, "", err
	}

	if connectionEnd.GetState() != int32(connectiontypes.OPEN) {
		return 0, "", errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connectionEnd.GetState()).String(),
		)
	}

	if connectionEnd.GetCounterparty().GetConnectionID() != counterpartyChannel.ConnectionHops[0] {
		return 0, "", errorsmod.Wrapf(connectiontypes.ErrInvalidConnection, "unexpected counterparty channel connection hops, expected %s but got %s", connectionEnd.GetCounterparty().GetConnectionID(), counterpartyChannel.ConnectionHops[0])
	}

	if err := k.connectionKeeper.VerifyChannelState(ctx, connectionEnd, proofHeight, proofChannel, proposedUpgradeChannel.Counterparty.PortId,
		proposedUpgradeChannel.Counterparty.ChannelId, counterpartyChannel); err != nil {
		return 0, "", err
	}

	upgradeTimeout := types.UpgradeTimeout{TimeoutHeight: timeoutHeight, TimeoutTimestamp: timeoutTimestamp}
	if err := k.connectionKeeper.VerifyChannelUpgradeTimeout(ctx, connectionEnd, proofHeight, proofUpgradeTimeout, proposedUpgradeChannel.Counterparty.PortId,
		proposedUpgradeChannel.Counterparty.ChannelId, upgradeTimeout); err != nil {
		return 0, "", err
	}

	if err := k.connectionKeeper.VerifyChannelUpgradeSequence(ctx, connectionEnd, proofHeight, proofUpgradeSequence, proposedUpgradeChannel.Counterparty.PortId,
		proposedUpgradeChannel.Counterparty.ChannelId, counterpartyUpgradeSequence); err != nil {
		return 0, "", err
	}

	// check if upgrade timed out by comparing it with the latest height of the chain
	selfHeight := clienttypes.GetSelfHeight(ctx)
	if !timeoutHeight.IsZero() && selfHeight.GTE(timeoutHeight) {
		return 0, "", k.RestoreChannel(ctx, portID, channelID, upgradeSequence, types.ErrUpgradeTimeout)
	}

	// check if upgrade timed out by comparing it with the latest timestamp of the chain
	if timeoutTimestamp != 0 && uint64(ctx.BlockTime().UnixNano()) >= timeoutTimestamp {
		return 0, "", k.RestoreChannel(ctx, portID, channelID, upgradeSequence, types.ErrUpgradeTimeout)
	}

	switch channel.State {
	case types.OPEN:
		upgradeSequence = uint64(0)
		if seq, found := k.GetUpgradeSequence(ctx, portID, channelID); found {
			upgradeSequence = seq
		}

		// if the counterparty upgrade sequence is ahead then fast forward so both channel ends are using the same sequence for the current upgrade
		if counterpartyUpgradeSequence > upgradeSequence {
			upgradeSequence = counterpartyUpgradeSequence
			k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)
		} else {
			errorReceipt := types.NewErrorReceipt(upgradeSequence, errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "upgrade sequence %d was not smaller than the counter party chain upgrade sequence %d", upgradeSequence, counterpartyUpgradeSequence))
			// the upgrade sequence is incremented so both sides start the next upgrade with a fresh sequence.
			upgradeSequence++

			k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
			k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)

			// TODO: emit error receipt events

			// do we want to return upgrade sequence here to include in response??
			return 0, "", errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "upgrade aborted, error receipt written for upgrade sequence: %d", errorReceipt.GetSequence())
		}

		// this is first message in upgrade handshake on this chain so we must store original channel in restore channel path
		// in case we need to restore channel later.
		k.SetUpgradeRestoreChannel(ctx, portID, channelID, channel)
	case types.INITUPGRADE:
		upgradeSequence, found = k.GetUpgradeSequence(ctx, portID, channelID)
		if !found {
			errorReceipt := types.NewErrorReceipt(upgradeSequence, errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "upgrade sequence %d was not smaller than the counter party chain upgrade sequence %d", upgradeSequence, counterpartyUpgradeSequence))
			k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)

			return 0, "", errorsmod.Wrapf(types.ErrUpgradeAborted, "upgrade aborted, error receipt written for upgrade sequence: %d", upgradeSequence)
		}

		if upgradeSequence != counterpartyUpgradeSequence {
			errorReceipt := types.NewErrorReceipt(upgradeSequence, errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "upgrade sequence %d was not equal to the counter party chain upgrade sequence %d", upgradeSequence, counterpartyUpgradeSequence))

			// set to the max of the two
			if counterpartyUpgradeSequence > upgradeSequence {
				upgradeSequence = counterpartyUpgradeSequence
				k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)
			}

			k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
			return 0, "", errorsmod.Wrapf(types.ErrUpgradeAborted, "upgrade aborted, error receipt written for upgrade sequence: %d", upgradeSequence)
		}

		// if there is a crossing hello, i.e an UpgradeInit has been called on both channelEnds,
		// then we must ensure that the proposedUpgrade by the counterparty is the same as the currentChannel
		// except for the channel state (upgrade channel will be in TRYUPGRADE and current channel will be in INITUPGRADE)
		// if the proposed upgrades on either side are incompatible, then we will restore the channel and cancel the upgrade.
		channel.State = types.TRYUPGRADE
		k.SetChannel(ctx, portID, channelID, channel)

		if !reflect.DeepEqual(channel, proposedUpgradeChannel) {
			// TODO: log and emit events
			if err := k.RestoreChannel(ctx, portID, channelID, upgradeSequence, types.ErrInvalidChannel); err != nil {
				return 0, "", errorsmod.Wrap(types.ErrUpgradeAborted, err.Error())
			}
			return 0, "", errorsmod.Wrap(types.ErrUpgradeAborted, "proposed upgrade channel did not equal expected channel")
		}

		// retrieve restore channel to return previous version
		restoreChannel, found := k.GetUpgradeRestoreChannel(ctx, portID, channelID)
		if !found {
			// TODO(this case should be unreachable): write error receipt for upgrade sequence and abort / cancel upgrade
			return 0, "", errorsmod.Wrapf(types.ErrChannelNotFound, "upgrade aborted, error receipt written for upgrade sequence: %d", upgradeSequence)
		}

		return upgradeSequence, restoreChannel.Version, nil
	default:
		return 0, "", errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s] but got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	return upgradeSequence, channel.Version, nil
}

// WriteUpgradeTryChannel writes a channel which has successfully passed the UpgradeTry handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTryChannel(
	ctx sdk.Context,
	portID,
	channelID string,
	upgradeSequence uint64,
	channelUpgrade types.Channel,
) error {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-try")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "failed to retrieve channel %s on port %s", channelID, portID)
	}

	// assign directly the fields that are modifiable.
	// counterparty fields may not be changed.
	channel.Version = channelUpgrade.Version
	channel.Ordering = channelUpgrade.Ordering
	channel.ConnectionHops = channelUpgrade.ConnectionHops

	// prevent silent failures if fields that are not allowed to be modified are changed in the upgrade channel
	// request.
	if !reflect.DeepEqual(channel, channelUpgrade) {
		return errorsmod.Wrapf(types.ErrInvalidChannel, "expected: %v but got: %v", channel, channelUpgrade)
	}

	k.SetChannel(ctx, portID, channelID, channel)

	// TODO: previous state will not be OPEN in the case of crossing hellos. Determine this state correctly.
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.TRYUPGRADE.String())

	emitChannelUpgradeTryEvent(ctx, portID, channelID, upgradeSequence, channelUpgrade)
	return nil
}

// TODO: should we pull out the error receipt logic from this function? They seem like two discrete operations.

// RestoreChannel restores the given channel to the state prior to upgrade.
func (k Keeper) RestoreChannel(ctx sdk.Context, portID, channelID string, upgradeSequence uint64, err error) error {
	errorReceipt := types.NewErrorReceipt(upgradeSequence, err)
	k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)

	channel, found := k.GetUpgradeRestoreChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "channel-id: %s", channelID)
	}

	k.SetChannel(ctx, portID, channelID, channel)
	k.DeleteUpgradeRestoreChannel(ctx, portID, channelID)
	k.DeleteUpgradeTimeout(ctx, portID, channelID)

	module, _, err := k.LookupModuleByChannel(ctx, portID, channelID)
	if err != nil {
		return errorsmod.Wrap(err, "could not retrieve module from port-id")
	}

	portKeeper, ok := k.portKeeper.(*portkeeper.Keeper)
	if !ok {
		panic("todo: handle this situation")
	}

	cbs, found := portKeeper.Router.GetRoute(module)
	if !found {
		return errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to module: %s", module)
	}

	cbs.OnChanUpgradeRestore(ctx, portID, channelID)
	return nil
}

// getUpgradeTryConnectionEnd returns the connection end that should be used. During crossing hellos, the restore
// channel connection end is used, while in a regular flow the current channel connection end is used.
func (k Keeper) getUpgradeTryConnectionEnd(ctx sdk.Context, portID string, channelID string, currentChannel types.Channel) (exported.ConnectionI, error) {
	isCrossingHellos := currentChannel.State == types.INITUPGRADE
	if isCrossingHellos {
		// fetch restore channel
		restoreChannel, found := k.GetUpgradeRestoreChannel(ctx, portID, channelID)
		if !found {
			return nil, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
		}
		connectionEnd, err := k.GetConnection(ctx, restoreChannel.ConnectionHops[0])
		if err != nil {
			return nil, err
		}
		return connectionEnd, nil
	}

	// use current channel
	connectionEnd, err := k.GetConnection(ctx, currentChannel.ConnectionHops[0])
	if err != nil {
		return nil, err
	}
	return connectionEnd, nil
}
