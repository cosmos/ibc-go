package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// ChanUpgradeInit is called by a module to initiate a channel upgrade handshake with
// a module on another chain.
func (k Keeper) ChanUpgradeInit(
	ctx sdk.Context,
	portID string,
	channelID string,
	upgradeFields types.UpgradeFields,
	upgradeTimeout types.Timeout,
) (types.Upgrade, error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return types.Upgrade{}, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.OPEN {
		return types.Upgrade{}, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.OPEN, channel.State)
	}

	if err := k.ValidateUpgradeFields(ctx, upgradeFields, channel); err != nil {
		return types.Upgrade{}, err
	}

	proposedUpgrade, err := k.constructProposedUpgrade(ctx, portID, channelID, upgradeFields, upgradeTimeout)
	if err != nil {
		return types.Upgrade{}, errorsmod.Wrap(err, "failed to construct proposed upgrade")
	}

	channel.UpgradeSequence++
	k.SetChannel(ctx, portID, channelID, channel)

	return proposedUpgrade, nil
}

// WriteUpgradeInitChannel writes a channel which has successfully passed the UpgradeInit handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeInitChannel(ctx sdk.Context, portID, channelID string, upgrade types.Upgrade) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-init")

	currentChannel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeInit step, channelID: %s, portID: %s", channelID, portID))
	}

	currentChannel.State = types.INITUPGRADE

	k.SetChannel(ctx, portID, channelID, currentChannel)
	k.SetUpgrade(ctx, portID, channelID, upgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN.String(), "new-state", types.INITUPGRADE.String())

	emitChannelUpgradeInitEvent(ctx, portID, channelID, currentChannel, upgrade)
}

// ChanUpgradeTry is called by a module to accept the first step of a channel upgrade handshake initiated by
// a module on another chain. If this function is successful, the proposed upgrade will be returned. If the upgrade fails, the upgrade sequence will still be incremented but an error will be returned.
func (k Keeper) ChanUpgradeTry(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedConnectionHops []string,
	proposedUpgradeTimeout types.Timeout,
	counterpartyProposedUpgrade types.Upgrade,
	counterpartyUpgradeSequence uint64,
	proofCounterpartyChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
) (types.Upgrade, error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return types.Upgrade{}, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	// the channel state must be in OPEN or INITUPGRADE if we are in a crossing hellos situation
	if !collections.Contains(channel.State, []types.State{types.OPEN, types.INITUPGRADE}) {
		return types.Upgrade{}, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.INITUPGRADE, channel.State)
	}

	// verify that the timeout set in UpgradeInit has not passed on this chain
	if hasPassed, err := counterpartyProposedUpgrade.Timeout.HasPassed(ctx); hasPassed {
		// abort here and let counterparty timeout the upgrade
		return types.Upgrade{}, errorsmod.Wrap(err, "upgrade timeout has passed")
	}

	connectionEnd, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return types.Upgrade{}, errorsmod.Wrap(err, "failed to retrieve connection using the channel connection hops")
	}

	// make sure connection is OPEN
	if connectionEnd.GetState() != int32(connectiontypes.OPEN) {
		return types.Upgrade{}, errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connectionEnd.GetState()).String(),
		)
	}

	// assert that the proposed connection hops are compatible with the counterparty connection hops
	// the proposed connections hops must have a counterparty which matches the counterparty connection hops
	proposedConnection, err := k.GetConnection(ctx, proposedConnectionHops[0])
	if err != nil {
		return types.Upgrade{}, err
	}

	counterpartyProposedHops := counterpartyProposedUpgrade.Fields.ConnectionHops
	if proposedConnection.GetCounterparty().GetConnectionID() != counterpartyProposedHops[0] {
		return types.Upgrade{}, errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnection,
			"proposed connection hops (%s) does not have counterparty proposed connection hops as counterparty, expected %s, got %s",
			proposedConnectionHops,
			counterpartyProposedHops,
			proposedConnection.GetCounterparty().GetConnectionID(),
		)
	}

	// construct counterpartyChannel from existing information and provided counterpartyUpgradeSequence
	// currentCounterpartyHops := connection.GetCounterparty().GetConnectionID()
	// counterpartyChannel := types.Channel{
	// 	State:           types.INITUPGRADE,
	// 	Counterparty:    types.NewCounterparty(portID, channelID),
	// 	Ordering:        channel.Ordering,
	// 	ConnectionHops:  []string{currentCounterpartyHops},
	// 	Version:         channel.Version,
	// 	UpgradeSequence: counterpartyUpgradeSequence,
	// }

	// create upgrade fields from counterparty proposed upgrade and own verified connection hops
	// upgradeFields := types.NewUpgradeFields(
	// 	counterpartyProposedUpgrade.Fields.Ordering,
	// 	proposedConnectionHops,
	// 	counterpartyProposedUpgrade.Fields.Version,
	// )

	// TODO: if OPEN, then initialize handshake with upgradeFields
	// this should validate the upgrade fields, set the upgrade path and set the final correct sequence.
	var proposedUpgrade types.Upgrade
	// if channel.State == types.OPEN {

	// TODO: otherwise, if the channel state is already in INITUPGRADE (crossing hellos case),
	// assert that the upgrade fields are the same as the upgrade already in progress
	// nolint:staticcheck
	// } else if channel.State == types.INITUPGRADE {

	// if the counterparty sequence is not equal to our own at this point, either the counterparty chain is out-of-sync or the message is out-of-sync
	// we write an error receipt with our own sequence so that the counterparty can update their sequence as well.
	// We must then increment our sequence so both sides start the next upgrade with a fresh sequence.

	// if counterpartyUpgradeSequence != channel.UpgradeSequence {
	// 	errorReceipt := types.NewErrorReceipt(channel.UpgradeSequence, errorsmod.Wrapf(types.ErrInvalidUpgrade, "counterparty chain upgrade sequence <= upgrade sequence (%d <= %d)", counterpartyUpgradeSequence, channel.UpgradeSequence))
	// 	channel.UpgradeSequence++
	// 	// TODO: emit error receipt events
	// 	k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
	// }
	// }

	return proposedUpgrade, nil
}

// WriteUpgradeTryChannel writes a channel which has successfully passed the UpgradeTry handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTryChannel(
	ctx sdk.Context,
	portID, channelID string,
	proposedUpgrade types.Upgrade,
	flushStatus types.FlushStatus,
) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-try")

	currentChannel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeTry step, channelID: %s, portID: %s", channelID, portID))
	}

	previousState := currentChannel.State
	currentChannel.State = types.TRYUPGRADE
	currentChannel.FlushStatus = flushStatus

	k.SetChannel(ctx, portID, channelID, currentChannel)
	k.SetUpgrade(ctx, portID, channelID, proposedUpgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState, "new-state", types.TRYUPGRADE.String())
	emitChannelUpgradeTryEvent(ctx, portID, channelID, currentChannel, proposedUpgrade)
}

// constructProposedUpgrade returns the proposed upgrade from the provided arguments.
func (k Keeper) constructProposedUpgrade(ctx sdk.Context, portID, channelID string, fields types.UpgradeFields, upgradeTimeout types.Timeout) (types.Upgrade, error) {
	nextSequenceSend, found := k.GetNextSequenceSend(ctx, portID, channelID)
	if !found {
		return types.Upgrade{}, types.ErrSequenceSendNotFound
	}

	return types.Upgrade{
		Fields:             fields,
		Timeout:            upgradeTimeout,
		LatestSequenceSend: nextSequenceSend - 1,
	}, nil
}
