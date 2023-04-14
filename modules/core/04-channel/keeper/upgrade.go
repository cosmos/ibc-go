package keeper

import (
	"fmt"
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

	connectionEnd, err := k.GetConnection(ctx, proposedUpgradeChannel.ConnectionHops[0])
	if err != nil {
		return 0, "", err
	}

	if connectionEnd.GetState() != int32(connectiontypes.OPEN) {
		return 0, "", errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connectionEnd.GetState()).String(),
		)
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

	k.SetChannel(ctx, portID, channelID, channelUpgrade)
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.INITUPGRADE.String())

	emitChannelUpgradeInitEvent(ctx, portID, channelID, upgradeSequence, channelUpgrade)
}

// ChanUpgradeTry is called by a module to accept the first step of a channel upgrade
// handshake initiated by a module on another chain. If this function is successful, the upgrade sequence
// will be returned. If an error occurs in the callback, 0 will be returned but the upgrade sequence will
// be incremented.
func (k Keeper) ChanUpgradeTry(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedUpgrade,
	counterpartyProposedUpgrade types.Upgrade,
	counterpartyUpgradeSequence uint64,
	proofChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
) (upgradeSequence uint64, err error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	// the channel state could be in INITUPGRADE if we are in a crossing hellos situation
	if !collections.Contains(channel.State, []types.State{types.OPEN, types.INITUPGRADE}) {
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	if err = k.ValidateProposedUpgradeFields(ctx, proposedUpgrade.UpgradeFields, channel); err != nil {
		return 0, errorsmod.Wrapf(types.ErrInvalidUpgrade, "proposed upgrade fields are invalid: %s", err.Error())
	}

	connectionEnd, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return 0, err
	}

	counterpartyConnectionHops := []string{connectionEnd.GetCounterparty().GetConnectionID()}

	expectedCounterpartyChannel := types.Channel{
		State:           types.INITUPGRADE,
		Counterparty:    types.NewCounterparty(channel.Counterparty.PortId, channel.Counterparty.ChannelId),
		Ordering:        channel.Ordering,
		ConnectionHops:  counterpartyConnectionHops,
		Version:         channel.Version,
		UpgradeSequence: counterpartyUpgradeSequence,
	}

	// verify that the counterparty channel has entered into the upgrade process
	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connectionEnd,
		proofHeight,
		proofChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		expectedCounterpartyChannel,
	); err != nil {
		return 0, err
	}
	// verify the proposed upgrade information provided by the counterparty
	if err := k.connectionKeeper.VerifyChannelUpgrade(ctx, connectionEnd, proofHeight, proofUpgrade, channel.Counterparty.PortId,
		channel.Counterparty.ChannelId, counterpartyProposedUpgrade); err != nil {
		return 0, err
	}
	// verify that the timeout set in UpgradeInit has not passed on this chain
	if hasPassed, err := counterpartyProposedUpgrade.Timeout.HasPassed(ctx); hasPassed {
		return 0, err
	}

	// if the counterparty sequence is less than or equal to the current sequence, then either the counterparty chain is out-of-sync or
	// the message is out-of-sync and we write an error receipt with our own sequence so that the counterparty can update
	// their sequence as well.
	// We must then increment our sequence so both sides start the next upgrade with a fresh sequence.
	if counterpartyUpgradeSequence <= upgradeSequence {
		errorReceipt := types.NewErrorReceipt(upgradeSequence, errorsmod.Wrapf(types.ErrInvalidUpgrade, "counterparty chain upgrade sequence <= upgrade sequence (%d <= %d)", counterpartyUpgradeSequence, upgradeSequence))
		upgradeSequence++
		k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
		k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)

		return 0, errorsmod.Wrapf(types.ErrInvalidUpgrade, "upgrade aborted, error receipt written for upgrade sequence: %d", upgradeSequence)
	}

	// if the counterparty sequence is greater than the current sequence, we fast forward to the counterparty sequence
	// so that both channel ends are using the same sequence for the current upgrade
	if counterpartyUpgradeSequence > upgradeSequence {
		channel.UpgradeSequence = counterpartyUpgradeSequence
		k.SetChannel(ctx, portID, channelID, channel)
	}

	// happy path case
	// increment upgrade sequence appropriately
	if channel.State == types.OPEN {
		upgradeSequence++
		k.SetUpgradeSequence(ctx, portID, channelID, upgradeSequence)
	}

	// crossing hellos case
	if channel.State == types.INITUPGRADE {
		if !reflect.DeepEqual(proposedUpgrade.UpgradeFields, counterpartyProposedUpgrade.UpgradeFields) {
			return 0, errorsmod.Wrap(types.ErrInvalidUpgrade, "crossing hellos, but proposed upgrade fields do not match")
		}
	}
	// TODO: emit error receipt events

	// this is first message in upgrade handshake on this chain so we must store original channel in restore channel path
	// in case we need to restore channel later.
	k.SetUpgradeRestoreChannel(ctx, portID, channelID, channel)
	return upgradeSequence, nil
}

// WriteUpgradeTryChannel writes a channel which has successfully passed the UpgradeTry handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTryChannel(
	ctx sdk.Context,
	portID, channelID string,
	upgrade types.Upgrade,
) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-try")
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("failed to retrieve channel %s on port %s", channelID, portID))
	}

	channel.State = types.TRYUPGRADE

	k.SetChannel(ctx, portID, channelID, channel)
	k.SetUpgrade(ctx, portID, channelID, upgrade)

	// TODO: previous state will not be OPEN in the case of crossing hellos. Determine this state correctly.
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.TRYUPGRADE.String())
	emitChannelUpgradeTryEvent(ctx, portID, channelID, channel, upgrade)
}

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

	return cbs.OnChanUpgradeRestore(ctx, portID, channelID)
}
