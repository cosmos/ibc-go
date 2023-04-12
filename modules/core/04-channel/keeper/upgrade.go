package keeper

import (
	"fmt"
	"reflect"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	portkeeper "github.com/cosmos/ibc-go/v7/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
)

// ChanUpgradeInit is called by a module to initiate a channel upgrade handshake with
// a module on another chain.
func (k Keeper) ChanUpgradeInit(ctx sdk.Context, portID string, channelID string, upgrade types.Upgrade) (upgradeSequence uint64, err error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.OPEN {
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.OPEN, channel.State)
	}

	if err := k.validateProposedUpgradeFields(ctx, channel, upgrade.ProposedUpgrade); err != nil {
		return 0, err
	}

	channel.UpgradeSequence++
	k.SetChannel(ctx, portID, channelID, channel)

	return channel.UpgradeSequence, nil
}

// WriteUpgradeInitChannel writes a channel which has successfully passed the UpgradeInit handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeInit(ctx sdk.Context, portID, channelID string, upgrade types.Upgrade) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-init")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("failed to retrieve channel %s on port %s", channelID, portID))
	}

	channel.State = types.INITUPGRADE
	k.SetChannel(ctx, portID, channelID, channel)
	k.SetUpgrade(ctx, portID, channelID, upgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.INITUPGRADE.String())

	// emitChannelUpgradeInitEvent(ctx, portID, channelID, upgradeSequence, upgradeChannel)
}

// ChanUpgradeTry is called by a module to accept the first step of a channel upgrade
// handshake initiated by a module on another chain. If this function is successful, the upgrade sequence
// will be returned. If an error occurs in the callback, 0 will be returned but the upgrade sequence will
// be incremented.
func (k Keeper) ChanUpgradeTry(ctx sdk.Context, portID string, channelID string, proposedUpgrade, counterpartyProposedUpgrade types.Upgrade, counterpartyUpgradeSequence uint64, proofChannel []byte, proofUpgrade []byte, proofHeight clienttypes.Height) (upgradeSequence uint64, err error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	// the channel state could be in INITUPGRADE if we are in a crossing hellos situation
	if !collections.Contains(channel.State, []types.State{types.OPEN, types.INITUPGRADE}) {
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	// TODO: add check that if crossing hellos case, the currently set proposed upgrade == the passed in proposed upgrade fields (ie they did not change)

	// proposed upgrade fields must be equal on both sides of the upgrade
	if reflect.DeepEqual(proposedUpgrade.ProposedUpgrade, counterpartyProposedUpgrade.ProposedUpgrade) {
		return 0, errorsmod.Wrap(types.ErrChannelExists, "existing channel end is identical to proposed upgrade channel end")
	}

	if err := k.validateProposedUpgradeFields(ctx, channel, proposedUpgrade.ProposedUpgrade); err != nil {
		return 0, err
	}

	connectionEnd, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return 0, err
	}

	if connectionEnd.GetState() != int32(connectiontypes.OPEN) {
		return 0, errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connectionEnd.GetState()).String(),
		)
	}

	counterpartyConnectionHops := []string{connectionEnd.GetCounterparty().GetConnectionID()}
	expectedCounterpartyChannel := types.Channel{
		State:           types.INITUPGRADE,
		Counterparty:    types.NewCounterparty(portID, channelID),
		Ordering:        channel.Ordering,
		ConnectionHops:  counterpartyConnectionHops,
		Version:         channel.Version,
		UpgradeSequence: counterpartyUpgradeSequence,
	}

	// verify that the counterparty channel has entered into the upgrade process
	if err := k.connectionKeeper.VerifyChannelState(ctx, connectionEnd, proofHeight, proofChannel, channel.Counterparty.PortId,
		channel.Counterparty.ChannelId, expectedCounterpartyChannel); err != nil {
		return 0, err
	}

	// verify the proposed upgrade information provided by the counterparty
	if err := k.connectionKeeper.VerifyChannelUpgrade(ctx, connectionEnd, proofHeight, proofUpgrade, channel.Counterparty.PortId,
		channel.Counterparty.ChannelId, counterpartyProposedUpgrade); err != nil {
		return 0, err
	}

	/*
		replace the following with:

		selfHeight := clienttypes.GetSelfHeight(ctx)
		selfTime := uint64(ctx.BlockTime.UnixNano())
		if info, hasPassed := counterpartyProposedUpgrade.Timeout.Status(selfHeight, selfTime); hasPassed {
			return 0, types.NewErrorReceipt(upgradeSequence, info)
		}
	*/

	// check if upgrade timed out by comparing it with the latest height of the chain
	selfHeight := clienttypes.GetSelfHeight(ctx)
	timeoutHeight := counterpartyProposedUpgrade.Timeout.TimeoutHeight
	if !timeoutHeight.IsZero() && selfHeight.GTE(timeoutHeight) {
		if err := k.WriteErrorReceipt(ctx, portID, channelID, upgradeSequence, types.ErrUpgradeTimeout); err != nil {
			return 0, errorsmod.Wrap(types.ErrUpgradeAborted, err.Error())
		}
		return 0, errorsmod.Wrapf(types.ErrUpgradeAborted, "block height >= upgrade timeout height (%s >= %s)", selfHeight, timeoutHeight)
	}

	// check if upgrade timed out by comparing it with the latest timestamp of the chain
	timeoutTimestamp := counterpartyProposedUpgrade.Timeout.TimeoutTimestamp
	if timeoutTimestamp != 0 && uint64(ctx.BlockTime().UnixNano()) >= timeoutTimestamp {
		upgradeSequence = uint64(0)
		if err := k.WriteErrorReceipt(ctx, portID, channelID, upgradeSequence, types.ErrUpgradeTimeout); err != nil {
			return 0, errorsmod.Wrap(types.ErrUpgradeAborted, err.Error())
		}
		return 0, errorsmod.Wrapf(types.ErrUpgradeAborted, "block timestamp >= upgrade timeout timestamp (%s >= %s)", ctx.BlockTime(), time.Unix(0, int64(timeoutTimestamp)))
	}

	// increment upgrade sequence for new upgrade attempt
	if channel.State == types.OPEN {
		channel.UpgradeSequence = channel.UpgradeSequence + 1

		// if the counterparty upgrade sequence is ahead then fast forward so both channel ends are using the same sequence for the current upgrade
		if counterpartyUpgradeSequence > channel.UpgradeSequence {
			channel.UpgradeSequence = counterpartyUpgradeSequence
		}

		k.SetChannel(ctx, portID, channelID, channel)
	}

	// check that both sides have the same upgrade sequence
	if counterpartyUpgradeSequence != upgradeSequence {
		errorReceipt := types.NewErrorReceipt(upgradeSequence, errorsmod.Wrapf(types.ErrUpgradeAborted, "counterparty chain upgrade sequence <= upgrade sequence (%d <= %d)", counterpartyUpgradeSequence, upgradeSequence))
		k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)

		// fast forward sequence for crossing hellos case
		if channel.State == types.INITUPGRADE {
			if counterpartyUpgradeSequence > upgradeSequence {
				channel.UpgradeSequence = counterpartyUpgradeSequence
				k.SetChannel(ctx, portID, channelID, channel)
			}
		}

		// do we want to return upgrade sequence here to include in response??
		return 0, errorsmod.Wrapf(types.ErrUpgradeAborted, "upgrade aborted, error receipt written for upgrade sequence: %d", errorReceipt.GetSequence())
	}

	// TODO: emit error receipt events

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

	// assign directly the fields that are modifiable.
	// counterparty fields may not be changed.
	channel.State = types.TRYUPGRADE
	k.SetChannel(ctx, portID, channelID, channel)
	k.SetUpgrade(ctx, portID, channelID, upgrade)

	// TODO: previous state will not be OPEN in the case of crossing hellos. Determine this state correctly.
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.TRYUPGRADE.String())

	// emitChannelUpgradeTryEvent(ctx, portID, channelID, upgradeSequence, channelUpgrade)
}

// TODO: should we pull out the error receipt logic from this function? They seem like two discrete operations.

// WriteErrorReceipt restores the given channel to the state prior to upgrade.
func (k Keeper) WriteErrorReceipt(ctx sdk.Context, portID, channelID string, upgradeSequence uint64, err error) error {
	errorReceipt := types.NewErrorReceipt(upgradeSequence, err)
	k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
	// TODO: add abort function which call this function and sets the channel to OPEN

	/*
		TODO: This should still callback?

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
	*/
	return nil
}

// validateProposedUpgradeFields validates the proposed upgrade fields against the existing channel.
// It returns an error if the following constraints are not met:
// - there exists at least one valid proposed change to the existing channel fields
// - the proposed order is a subset of the existing order
// - the proposed connection hops do not exist
// - the proposed version is non empty (checked in ModifiableUpgradeFields.ValidateBasic())
func (k Keeper) validateProposedUpgradeFields(ctx sdk.Context, existingChannel types.Channel, proposedUpgrade types.ModifiableUpgradeFields) error {
	currentFields := types.ModifiableUpgradeFields{
		Ordering:       existingChannel.Ordering,
		ConnectionHops: existingChannel.ConnectionHops,
		Version:        existingChannel.Version,
	}
	if reflect.DeepEqual(proposedUpgrade, currentFields) {
		return errorsmod.Wrap(types.ErrChannelExists, "existing channel end is identical to proposed upgrade channel end")
	}

	if !currentFields.Ordering.SubsetOf(proposedUpgrade.Ordering) {
		return errorsmod.Wrap(types.ErrInvalidChannelOrdering, "channel ordering must be a subset of the new ordering")
	}

	if !k.connectionKeeper.HasConnection(ctx, proposedUpgrade.ConnectionHops[0]) {
		return errorsmod.Wrapf(connectiontypes.ErrConnectionNotFound, "failed to retrieve connection: %s", proposedUpgrade.ConnectionHops[0])
	}

	return nil
}
