package keeper

import (
	"fmt"
	"reflect"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// ChanUpgradeInit is called by a module to initiate a channel upgrade handshake with
// a module on another chain.
func (k Keeper) ChanUpgradeInit(
	ctx sdk.Context,
	portID string,
	channelID string,
	upgrade types.Upgrade,
) (upgradeSequence uint64, previousVersion string, err error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return 0, "", errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.OPEN {
		return 0, "", errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.OPEN, channel.State)
	}

	if err := k.ValidateProposedUpgradeFields(ctx, upgrade.UpgradeFields, channel); err != nil {
		return 0, "", err
	}

	channel.UpgradeSequence++
	k.SetChannel(ctx, portID, channelID, channel)

	return channel.UpgradeSequence, channel.Version, nil
}

// WriteUpgradeInitChannel writes a channel which has successfully passed the UpgradeInit handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeInitChannel(ctx sdk.Context, portID, channelID string, upgrade types.Upgrade) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-init")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("failed to retrieve channel %s on port %s", channelID, portID))
	}

	channel.State = types.INITUPGRADE

	k.SetChannel(ctx, portID, channelID, channel)
	k.SetUpgrade(ctx, portID, channelID, upgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.INITUPGRADE.String())

	emitChannelUpgradeInitEvent(ctx, portID, channelID, channel, upgrade)
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
	proofCounterpartyChannel,
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

	// validate the proposed upgrade fields against the existing channel
	if err = k.ValidateProposedUpgradeFields(ctx, counterpartyProposedUpgrade.UpgradeFields, channel); err != nil {
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

	// verify that the counterparty channel has correctly entered into the upgrade process
	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connectionEnd,
		proofHeight,
		proofCounterpartyChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		expectedCounterpartyChannel,
	); err != nil {
		return 0, err
	}

	// verifies the proof that a particular proposed upgrade has been stored in the upgrade path.
	if err := k.connectionKeeper.VerifyChannelUpgrade(ctx, connectionEnd, proofHeight, proofUpgrade, channel.Counterparty.PortId,
		channel.Counterparty.ChannelId, counterpartyProposedUpgrade); err != nil {
		return 0, err
	}

	// verify that the timeout set in UpgradeInit has not passed on this chain
	if hasPassed, err := counterpartyProposedUpgrade.Timeout.HasPassed(ctx); hasPassed {
		errorReceipt := types.NewErrorReceipt(channel.UpgradeSequence, err)
		// TODO: emit error receipt events
		k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
		return 0, errorsmod.Wrapf(types.ErrInvalidUpgrade, "upgrade aborted, error receipt written for upgrade sequence: %d", channel.UpgradeSequence)
	}

	// if the counterparty sequence is less than or equal to the current sequence, then either the counterparty chain is out-of-sync or
	// the message is out-of-sync and we write an error receipt with our own sequence so that the counterparty can update
	// their sequence as well.
	// We must then increment our sequence so both sides start the next upgrade with a fresh sequence.
	if counterpartyUpgradeSequence <= channel.UpgradeSequence {
		errorReceipt := types.NewErrorReceipt(channel.UpgradeSequence, errorsmod.Wrapf(types.ErrInvalidUpgrade, "counterparty chain upgrade sequence <= upgrade sequence (%d <= %d)", counterpartyUpgradeSequence, channel.UpgradeSequence))
		channel.UpgradeSequence++
		// TODO: emit error receipt events
		k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
		k.SetChannel(ctx, portID, channelID, channel)

		return 0, errorsmod.Wrapf(types.ErrInvalidUpgrade, "upgrade aborted, error receipt written for upgrade sequence: %d", channel.UpgradeSequence)
	}

	// happy path case
	// increment upgrade sequence appropriately
	if channel.State == types.OPEN {
		channel.UpgradeSequence++
		k.SetChannel(ctx, portID, channelID, channel)
		k.SetUpgrade(ctx, portID, channelID, proposedUpgrade)
	}

	// crossing hellos case
	if channel.State == types.INITUPGRADE {
		currentUpgrade, found := k.GetUpgrade(ctx, portID, channelID)
		if !found {
			return 0, errorsmod.Wrap(types.ErrInvalidUpgrade, "failed to retrieve upgrade")
		}

		if !reflect.DeepEqual(currentUpgrade.UpgradeFields, proposedUpgrade.UpgradeFields) {
			return 0, errorsmod.Wrap(types.ErrInvalidUpgrade, "proposed upgrade fields have changed since UpgradeInit")
		}

		// if the counterparty sequence is greater than the current sequence, we fast forward to the counterparty sequence
		// so that both channel ends are using the same sequence for the current upgrade
		if counterpartyUpgradeSequence > channel.UpgradeSequence {
			channel.UpgradeSequence = counterpartyUpgradeSequence
			k.SetChannel(ctx, portID, channelID, channel)
		}
	}

	return channel.UpgradeSequence, nil
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

	// TODO: previous state will not be OPEN in the case of crossing hellos. Determine this state correctly.
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.TRYUPGRADE.String())
	emitChannelUpgradeTryEvent(ctx, portID, channelID, channel, upgrade)
}
