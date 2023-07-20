package keeper

import (
	"fmt"
	"reflect"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
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

	if err := k.validateSelfUpgradeFields(ctx, upgradeFields, channel); err != nil {
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
	upgradeTimeout types.Timeout,
	counterpartyUpgrade types.Upgrade,
	counterpartyUpgradeSequence uint64,
	proofCounterpartyChannel,
	proofCounterpartyUpgrade []byte,
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

	connection, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return types.Upgrade{}, errorsmod.Wrap(err, "failed to retrieve connection using the channel connection hops")
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return types.Upgrade{}, errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	if hasPassed, err := counterpartyUpgrade.Timeout.HasPassed(ctx); hasPassed {
		// abort here and let counterparty timeout the upgrade
		return types.Upgrade{}, errorsmod.Wrap(err, "upgrade timeout has passed")
	}

	// construct counterpartyChannel from existing information and provided counterpartyUpgradeSequence
	// create upgrade fields from counterparty proposed upgrade and own verified connection hops
	proposedUpgradeFields := types.UpgradeFields{
		Ordering:       counterpartyUpgrade.Fields.Ordering,
		ConnectionHops: proposedConnectionHops,
		Version:        counterpartyUpgrade.Fields.Version,
	}

	var upgrade types.Upgrade

	switch channel.State {
	case types.OPEN:
		// initialize handshake with upgrade fields
		upgrade, err = k.ChanUpgradeInit(ctx, portID, channelID, proposedUpgradeFields, upgradeTimeout)
		if err != nil {
			return types.Upgrade{}, errorsmod.Wrap(err, "failed to initialize upgrade")
		}

		// TODO: add fast forward feature
		// https://github.com/cosmos/ibc-go/issues/3794

		// NOTE: OnChanUpgradeInit will not be executed by the application

		k.WriteUpgradeInitChannel(ctx, portID, channelID, upgrade)

	case types.INITUPGRADE:
		// crossing hellos
		// assert that the upgrade fields are the same as the upgrade already in progress
		upgrade, found = k.GetUpgrade(ctx, portID, channelID)
		if !found {
			return types.Upgrade{}, errorsmod.Wrapf(types.ErrUpgradeNotFound, "current upgrade not found despite channel state being in %s", types.INITUPGRADE)
		}

		if !reflect.DeepEqual(upgrade.Fields, proposedUpgradeFields) {
			return types.Upgrade{}, errorsmod.Wrapf(
				types.ErrInvalidUpgrade, "upgrade fields are not equal to current upgrade fields in crossing hellos case, expected %s", upgrade.Fields)
		}

	default:
		panic(fmt.Sprintf("channel state should be asserted to be in OPEN or INITUPGRADE before reaching this check; state is %s", channel.State))
	}

	// construct expected counterparty channel from information in state
	// only the counterpartyUpgradeSequence is provided by the relayer
	counterpartyConnectionHops := []string{connection.GetCounterparty().GetConnectionID()}
	counterpartyChannel := types.Channel{
		State:           types.INITUPGRADE,
		Ordering:        channel.Ordering,
		Counterparty:    types.NewCounterparty(portID, channelID),
		ConnectionHops:  counterpartyConnectionHops,
		Version:         channel.Version,
		UpgradeSequence: counterpartyUpgradeSequence, // provided by the relayer
		FlushStatus:     types.NOTINFLUSH,
	}

	if err := k.startFlushUpgradeHandshake(
		ctx,
		portID, channelID,
		proposedUpgradeFields,
		counterpartyChannel,
		counterpartyUpgrade,
		proofCounterpartyChannel, proofCounterpartyUpgrade,
		proofHeight,
	); err != nil {
		return types.Upgrade{}, err
	}

	return upgrade, nil
}

// WriteUpgradeTryChannel writes the channel end and upgrade to state after successfully passing the UpgradeTry handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTryChannel(ctx sdk.Context, portID, channelID string, upgrade types.Upgrade, upgradeVersion string, counterpartyLastSequenceSend uint64) (types.Channel, types.Upgrade) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-try")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeTry step, channelID: %s, portID: %s", channelID, portID))
	}

	previousState := channel.State
	channel.State = types.TRYUPGRADE
	channel.FlushStatus = types.FLUSHING

	if !k.HasInflightPackets(ctx, portID, channelID) {
		channel.FlushStatus = types.FLUSHCOMPLETE
	}

	upgrade.Fields.Version = upgradeVersion

	k.SetCounterpartyLastPacketSequence(ctx, portID, channelID, counterpartyLastSequenceSend)
	k.SetChannel(ctx, portID, channelID, channel)
	k.SetUpgrade(ctx, portID, channelID, upgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState, "new-state", types.TRYUPGRADE.String())
	emitChannelUpgradeTryEvent(ctx, portID, channelID, channel, upgrade)

	return channel, upgrade
}

// ChanUpgradeAck is called by a module to accept the ACKUPGRADE handshake step of the channel upgrade protocol.
// This method should only be called by the IBC core msg server.
// This method will verify that the counterparty has entered TRYUPGRADE
// and that its own upgrade is compatible with the selected counterparty version.
func (k Keeper) ChanUpgradeAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyFlushStatus types.FlushStatus,
	counterpartyUpgrade types.Upgrade,
	proofChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if !collections.Contains(channel.State, []types.State{types.INITUPGRADE, types.TRYUPGRADE}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.INITUPGRADE, types.TRYUPGRADE, channel.State)
	}

	if !collections.Contains(counterpartyFlushStatus, []types.FlushStatus{types.FLUSHING, types.FLUSHCOMPLETE}) {
		return errorsmod.Wrapf(types.ErrInvalidFlushStatus, "expected one of [%s, %s], got %s", types.FLUSHING, types.FLUSHCOMPLETE, counterpartyFlushStatus)
	}

	connection, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return errorsmod.Wrap(err, "failed to retrieve connection using the channel connection hops")
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String())
	}

	counterpartyHops := []string{connection.GetCounterparty().GetConnectionID()}
	counterpartyChannel := types.Channel{
		State:           types.TRYUPGRADE,
		Ordering:        channel.Ordering,
		ConnectionHops:  counterpartyHops,
		Counterparty:    types.NewCounterparty(portID, channelID),
		Version:         channel.Version,
		UpgradeSequence: channel.UpgradeSequence,
		FlushStatus:     counterpartyFlushStatus, // provided by the relayer
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", portID, channelID)
	}

	if err := k.startFlushUpgradeHandshake(ctx, portID, channelID, upgrade.Fields, counterpartyChannel, counterpartyUpgrade,
		proofChannel, proofUpgrade, proofHeight); err != nil {
		return err
	}

	// in the crossing hellos case, the versions returned by both on TRY must be the same
	if channel.State == types.TRYUPGRADE {
		if upgrade.Fields.Version != counterpartyUpgrade.Fields.Version {
			return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrap(types.ErrIncompatibleCounterpartyUpgrade, "both channel ends must agree on the same version"))
		}
	}

	return nil
}

// WriteUpgradeAckChannel writes a channel which has successfully passed the UpgradeAck handshake step as well as
// setting the upgrade for that channel.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeAckChannel(ctx sdk.Context, portID, channelID, upgradeVersion string, counterpartyLastSequenceSend uint64) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-ack")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeAck step, channelID: %s, portID: %s", channelID, portID))
	}

	previousState := channel.State
	channel.State = types.ACKUPGRADE
	channel.FlushStatus = types.FLUSHING

	if !k.HasInflightPackets(ctx, portID, channelID) {
		channel.FlushStatus = types.FLUSHCOMPLETE
	}

	k.SetCounterpartyLastPacketSequence(ctx, portID, channelID, counterpartyLastSequenceSend)
	k.SetChannel(ctx, portID, channelID, channel)

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("cound not find existing upgrade when updating channel state in successful ChanUpgradeAck step, channelID: %s, portID: %s", channelID, portID))
	}

	upgrade.Fields.Version = upgradeVersion

	k.SetUpgrade(ctx, portID, channelID, upgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState, "new-state", types.ACKUPGRADE.String())
	emitChannelUpgradeAckEvent(ctx, portID, channelID, channel, upgrade)
}

// ChanUpgradeOpen is called by a module to complete the channel upgrade handshake and move the channel back to an OPEN state.
// This method should only be called after both channels have flushed any in-flight packets.
// This method should only be called directly by the core IBC message server.
func (k Keeper) ChanUpgradeOpen(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelState types.State,
	proofCounterpartyChannel []byte,
	proofHeight clienttypes.Height,
) error {
	if k.HasInflightPackets(ctx, portID, channelID) {
		return errorsmod.Wrapf(types.ErrPendingInflightPackets, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if !collections.Contains(channel.State, []types.State{types.TRYUPGRADE, types.ACKUPGRADE}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.TRYUPGRADE, types.ACKUPGRADE, channel.State)
	}

	if channel.FlushStatus != types.FLUSHCOMPLETE {
		return errorsmod.Wrapf(types.ErrInvalidFlushStatus, "expected %s, got %s", types.FLUSHCOMPLETE, channel.FlushStatus)
	}

	connection, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return errorsmod.Wrap(err, "failed to retrieve connection using the channel connection hops")
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String())
	}

	var counterpartyChannel types.Channel
	switch counterpartyChannelState {
	case types.OPEN:
		upgrade, found := k.GetUpgrade(ctx, portID, channelID)
		if !found {
			return errorsmod.Wrapf(types.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", portID, channelID)
		}
		// If counterparty has reached OPEN, we must use the upgraded connection to verify the counterparty channel
		upgradeConnection, err := k.GetConnection(ctx, upgrade.Fields.ConnectionHops[0])
		if err != nil {
			return errorsmod.Wrap(err, "failed to retrieve connection using the upgrade connection hops")
		}

		if upgradeConnection.GetState() != int32(connectiontypes.OPEN) {
			return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(upgradeConnection.GetState()).String())
		}

		counterpartyChannel = types.Channel{
			State:           types.OPEN,
			Ordering:        upgrade.Fields.Ordering,
			ConnectionHops:  []string{upgradeConnection.GetCounterparty().GetConnectionID()},
			Counterparty:    types.NewCounterparty(portID, channelID),
			Version:         upgrade.Fields.Version,
			UpgradeSequence: channel.UpgradeSequence,
			FlushStatus:     types.NOTINFLUSH,
		}

	case types.TRYUPGRADE:
		// If the counterparty is in TRYUPGRADE, then we must have gone through the ACKUPGRADE step.
		if channel.State != types.ACKUPGRADE {
			return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.ACKUPGRADE, channel.State)
		}

		counterpartyChannel = types.Channel{
			State:           types.TRYUPGRADE,
			Ordering:        channel.Ordering,
			ConnectionHops:  []string{connection.GetCounterparty().GetConnectionID()},
			Counterparty:    types.NewCounterparty(portID, channelID),
			Version:         channel.Version,
			UpgradeSequence: channel.UpgradeSequence,
			FlushStatus:     types.FLUSHCOMPLETE,
		}

	case types.ACKUPGRADE:
		counterpartyChannel = types.Channel{
			State:           types.ACKUPGRADE,
			Ordering:        channel.Ordering,
			ConnectionHops:  []string{connection.GetCounterparty().GetConnectionID()},
			Counterparty:    types.NewCounterparty(portID, channelID),
			Version:         channel.Version,
			UpgradeSequence: channel.UpgradeSequence,
			FlushStatus:     types.FLUSHCOMPLETE,
		}

	default:
		panic(fmt.Sprintf("counterparty channel state should be in one of [%s, %s, %s]; got %s", types.TRYUPGRADE, types.ACKUPGRADE, types.OPEN, counterpartyChannelState))
	}

	if err = k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight, proofCounterpartyChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return errorsmod.Wrapf(err, "failed to verify counterparty channel, expected counterparty channel state: %s", counterpartyChannel.String())
	}

	return nil
}

// WriteUpgradeOpenChannel writes the agreed upon upgrade fields to the channel, sets the channel flush status to NOTINFLUSH and sets the channel state back to OPEN. This can be called in one of two cases:
// - In the UpgradeAck step of the handshake if both sides have already flushed all in-flight packets.
// - In the UpgradeOpen step of the handshake.
func (k Keeper) WriteUpgradeOpenChannel(ctx sdk.Context, portID, channelID string) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state, channelID: %s, portID: %s", channelID, portID))
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find upgrade when updating channel state, channelID: %s, portID: %s", channelID, portID))
	}

	// Switch channel fields to upgrade fields and set channel state to OPEN
	previousState := channel.State
	channel.Ordering = upgrade.Fields.Ordering
	channel.Version = upgrade.Fields.Version
	channel.ConnectionHops = upgrade.Fields.ConnectionHops
	channel.State = types.OPEN
	channel.FlushStatus = types.NOTINFLUSH

	k.SetChannel(ctx, portID, channelID, channel)

	// delete state associated with upgrade which is no longer required.
	k.deleteUpgrade(ctx, portID, channelID)
	k.deleteCounterpartyLastPacketSequence(ctx, portID, channelID)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState.String(), "new-state", types.OPEN.String())
	emitChannelUpgradeOpenEvent(ctx, portID, channelID, channel)
}

// ChanUpgradeCancel is called by a module to cancel a channel upgrade that is in progress.
func (k Keeper) ChanUpgradeCancel(ctx sdk.Context, portID, channelID string, errorReceipt types.ErrorReceipt, errorReceiptProof []byte, proofHeight clienttypes.Height) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	// the channel state must be in INITUPGRADE or TRYUPGRADE
	if !collections.Contains(channel.State, []types.State{types.INITUPGRADE, types.TRYUPGRADE}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.INITUPGRADE, types.TRYUPGRADE, channel.State)
	}

	// get underlying connection for proof verification
	connection, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return errorsmod.Wrap(err, "failed to retrieve connection using the channel connection hops")
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	if err := k.connectionKeeper.VerifyChannelUpgradeError(
		ctx,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		connection,
		errorReceipt,
		errorReceiptProof,
		proofHeight,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty error receipt")
	}

	// If counterparty sequence is less than the current sequence, abort the transaction since this error receipt is from a previous upgrade.
	// Otherwise, set our upgrade sequence to the counterparty's error sequence + 1 so that both sides start with a fresh sequence.
	currentSequence := channel.UpgradeSequence
	counterpartySequence := errorReceipt.Sequence
	if counterpartySequence < currentSequence {
		return errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "error receipt sequence (%d) must be greater than or equal to current sequence (%d)", counterpartySequence, currentSequence)
	}

	return nil
}

// WriteUpgradeCancelChannel writes a channel which has canceled the upgrade process.Auxiliary upgrade state is
// also deleted.
func (k Keeper) WriteUpgradeCancelChannel(ctx sdk.Context, portID, channelID string, newUpgradeSequence uint64) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-cancel")

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find upgrade when updating channel state, channelID: %s, portID: %s", channelID, portID))
	}

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state, channelID: %s, portID: %s", channelID, portID))
	}

	previousState := channel.State

	k.restoreChannel(ctx, portID, channelID, newUpgradeSequence, channel)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState, "new-state", types.OPEN.String())
	emitChannelUpgradeCancelEvent(ctx, portID, channelID, channel, upgrade)
}

// ChanUpgradeTimeout times out an outstanding upgrade.
// This should be used by the initialising chain when the counterparty chain has not responded to an upgrade proposal within the specified timeout period.
func (k Keeper) ChanUpgradeTimeout(
	ctx sdk.Context,
	portID, channelID string,
	counterpartyChannel types.Channel,
	prevErrorReceipt *types.ErrorReceipt,
	proofCounterpartyChannel,
	proofErrorReceipt []byte,
	proofHeight exported.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.INITUPGRADE {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "channel state is not INITUPGRADE (got %s)", channel.State)
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	connection, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(
			connectiontypes.ErrConnectionNotFound,
			channel.ConnectionHops[0],
		)
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	// proof must be from a height after timeout has elapsed. Either timeoutHeight or timeoutTimestamp must be defined.
	// if timeoutHeight is defined and proof is from before timeout height, abort transaction
	proofTimestamp, err := k.connectionKeeper.GetTimestampAtHeight(ctx, connection, proofHeight)
	if err != nil {
		return err
	}

	timeout := upgrade.Timeout
	proofHeightIsInvalid := timeout.Height.IsZero() || proofHeight.LT(timeout.Height)
	proofTimestampIsInvalid := timeout.Timestamp == 0 || proofTimestamp < timeout.Timestamp
	if proofHeightIsInvalid && proofTimestampIsInvalid {
		return errorsmod.Wrap(types.ErrInvalidUpgradeTimeout, "timeout has not yet passed on counterparty chain")
	}

	// counterparty channel must be proved to still be in OPEN state or INITUPGRADE state (crossing hellos)
	if !collections.Contains(counterpartyChannel.State, []types.State{types.OPEN, types.INITUPGRADE}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.INITUPGRADE, counterpartyChannel.State)
	}

	// verify the counterparty channel state
	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight, proofCounterpartyChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty channel state")
	}

	// Error receipt passed in is either nil or it is a stale error receipt from a previous upgrade
	if prevErrorReceipt == nil {
		if err := k.connectionKeeper.VerifyChannelUpgradeErrorAbsence(
			ctx,
			channel.Counterparty.PortId, channel.Counterparty.ChannelId,
			connection,
			proofErrorReceipt,
			proofHeight,
		); err != nil {
			return errorsmod.Wrap(err, "failed to verify absence of counterparty channel upgrade error receipt")
		}

		return nil
	}
	// timeout for this sequence can only succeed if the error receipt written into the error path on the counterparty
	// was for a previous sequence by the timeout deadline.
	upgradeSequence := channel.UpgradeSequence
	if upgradeSequence <= prevErrorReceipt.Sequence {
		return errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "previous counterparty error receipt sequence is greater than or equal to our current upgrade sequence: %d > %d", prevErrorReceipt.Sequence, upgradeSequence)
	}

	if err := k.connectionKeeper.VerifyChannelUpgradeError(
		ctx,
		channel.Counterparty.PortId, channel.Counterparty.ChannelId,
		connection,
		*prevErrorReceipt,
		proofErrorReceipt,
		proofHeight,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty channel upgrade error receipt")
	}

	return nil
}

// WriteUpgradeTimeoutChannel restores the channel state of an initialising chain in the event that the counterparty chain has passed the timeout set in ChanUpgradeInit to the state before the upgrade was proposed.
// Auxiliary upgrade state is also deleted.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTimeoutChannel(
	ctx sdk.Context,
	portID, channelID string,
) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-timeout")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeTimeout step, channelID: %s, portID: %s", channelID, portID))
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing upgrade when cancelling channel upgrade, channelID: %s, portID: %s", channelID, portID))
	}

	k.restoreChannel(ctx, portID, channelID, channel.UpgradeSequence, channel)

	k.Logger(ctx).Info("channel state restored", "port-id", portID, "channel-id", channelID)
	emitChannelUpgradeTimeoutEvent(ctx, portID, channelID, channel, upgrade)
}

// startFlushUpgradeHandshake will verify the counterparty proposed upgrade and the current channel state.
// Once the counterparty information has been verified, it will be validated against the self proposed upgrade.
// If any of the proposed upgrade fields are incompatible, an upgrade error will be returned resulting in an
// aborted upgrade.
func (k Keeper) startFlushUpgradeHandshake(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedUpgradeFields types.UpgradeFields,
	counterpartyChannel types.Channel,
	counterpartyUpgrade types.Upgrade,
	proofCounterpartyChannel,
	proofCounterpartyUpgrade []byte,
	proofHeight clienttypes.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	connection, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return errorsmod.Wrap(err, "failed to retrieve connection using the channel connection hops")
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String())
	}

	// verify the counterparty channel state containing the upgrade sequence
	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight, proofCounterpartyChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty channel state")
	}

	// verifies the proof that a particular proposed upgrade has been stored in the upgrade path of the counterparty
	if err := k.connectionKeeper.VerifyChannelUpgrade(
		ctx,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		connection,
		counterpartyUpgrade,
		proofCounterpartyUpgrade, proofHeight,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty upgrade")
	}

	// the current upgrade handshake must only continue if both channels are using the same upgrade sequence,
	// otherwise an error receipt must be written so that the upgrade handshake may be attempted again with synchronized sequences
	if counterpartyChannel.UpgradeSequence != channel.UpgradeSequence {
		// save the previous upgrade sequence for the error message
		prevUpgradeSequence := channel.UpgradeSequence

		// error on the higher sequence so that both chains synchronize on a fresh sequence
		channel.UpgradeSequence = sdkmath.Max(counterpartyChannel.UpgradeSequence, channel.UpgradeSequence)
		k.SetChannel(ctx, portID, channelID, channel)

		return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrapf(
			types.ErrIncompatibleCounterpartyUpgrade, "expected upgrade sequence (%d) to match counterparty upgrade sequence (%d)", prevUpgradeSequence, counterpartyChannel.UpgradeSequence),
		)
	}

	if err := k.checkForUpgradeCompatibility(ctx, proposedUpgradeFields, counterpartyUpgrade); err != nil {
		return types.NewUpgradeError(channel.UpgradeSequence, err)
	}

	return nil
}

// checkForUpgradeCompatibility checks performs stateful validation of self upgrade fields relative to counterparty upgrade.
func (k Keeper) checkForUpgradeCompatibility(ctx sdk.Context, proposedUpgradeFields types.UpgradeFields, counterpartyUpgrade types.Upgrade) error {
	// assert that both sides propose the same channel ordering
	if proposedUpgradeFields.Ordering != counterpartyUpgrade.Fields.Ordering {
		return errorsmod.Wrapf(types.ErrIncompatibleCounterpartyUpgrade, "expected upgrade ordering (%s) to match counterparty upgrade ordering (%s)", proposedUpgradeFields.Ordering, counterpartyUpgrade.Fields.Ordering)
	}

	proposedConnection, err := k.GetConnection(ctx, proposedUpgradeFields.ConnectionHops[0])
	if err != nil {
		// NOTE: this error is expected to be unreachable as the proposed upgrade connectionID should have been
		// validated in the upgrade INIT and TRY handlers
		return errorsmod.Wrap(err, "expected proposed connection to be found")
	}

	if proposedConnection.GetState() != int32(connectiontypes.OPEN) {
		// NOTE: this error is expected to be unreachable as the proposed upgrade connectionID should have been
		// validated in the upgrade INIT and TRY handlers
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "expected proposed connection to be OPEN (got %s)", connectiontypes.State(proposedConnection.GetState()).String())
	}

	// connectionHops can change in a channelUpgrade, however both sides must still be each other's counterparty.
	if counterpartyUpgrade.Fields.ConnectionHops[0] != proposedConnection.GetCounterparty().GetConnectionID() {
		return errorsmod.Wrapf(
			types.ErrIncompatibleCounterpartyUpgrade, "counterparty upgrade connection end is not a counterparty of self proposed connection end (%s != %s)", counterpartyUpgrade.Fields.ConnectionHops[0], proposedConnection.GetCounterparty().GetConnectionID())
	}

	return nil
}

// validateSelfUpgradeFields validates the proposed upgrade fields against the existing channel.
// It returns an error if the following constraints are not met:
// - there exists at least one valid proposed change to the existing channel fields
// - the proposed order is a subset of the existing order
// - the proposed connection hops do not exist
// - the proposed version is non-empty (checked in UpgradeFields.ValidateBasic())
// - the proposed connection hops are not open
func (k Keeper) validateSelfUpgradeFields(ctx sdk.Context, proposedUpgrade types.UpgradeFields, currentChannel types.Channel) error {
	currentFields := extractUpgradeFields(currentChannel)

	if reflect.DeepEqual(proposedUpgrade, currentFields) {
		return errorsmod.Wrapf(types.ErrChannelExists, "existing channel end is identical to proposed upgrade channel end: got %s", proposedUpgrade)
	}

	connectionID := proposedUpgrade.ConnectionHops[0]
	connection, found := k.connectionKeeper.GetConnection(ctx, connectionID)
	if !found {
		return errorsmod.Wrapf(connectiontypes.ErrConnectionNotFound, "failed to retrieve connection: %s", connectionID)
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	getVersions := connection.GetVersions()
	if len(getVersions) != 1 {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"single version must be negotiated on connection before opening channel, got: %v",
			getVersions,
		)
	}

	if !connectiontypes.VerifySupportedFeature(getVersions[0], proposedUpgrade.Ordering.String()) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"connection version %s does not support channel ordering: %s",
			getVersions[0], proposedUpgrade.Ordering.String(),
		)
	}

	return nil
}

// extractUpgradeFields returns the upgrade fields from the provided channel.
func extractUpgradeFields(channel types.Channel) types.UpgradeFields {
	return types.UpgradeFields{
		Ordering:       channel.Ordering,
		ConnectionHops: channel.ConnectionHops,
		Version:        channel.Version,
	}
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

// MustAbortUpgrade will restore the channel state and flush status to their pre-upgrade state so that upgrade is aborted.
// Any unnecessary state is deleted and an error receipt is written.
// This function is expected to always succeed, a panic will occur if an error occurs.
func (k Keeper) MustAbortUpgrade(ctx sdk.Context, portID, channelID string, err error) {
	if err := k.abortUpgrade(ctx, portID, channelID, err); err != nil {
		panic(err)
	}
}

// abortUpgrade will restore the channel state and flush status to their pre-upgrade state so that upgrade is aborted.
// Any unnecessary state is delete and an error receipt is written.
func (k Keeper) abortUpgrade(ctx sdk.Context, portID, channelID string, err error) error {
	if err == nil {
		return errorsmod.Wrap(types.ErrInvalidUpgradeError, "cannot abort upgrade handshake with nil error")
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	// the channel upgrade sequence has already been updated in ChannelUpgradeTry, so we can pass
	// its updated value.
	k.restoreChannel(ctx, portID, channelID, channel.UpgradeSequence, channel)

	// in the case of application callbacks, the error may not be an upgrade error.
	// in this case we need to construct one in order to write the error receipt.
	upgradeError, ok := err.(*types.UpgradeError)
	if !ok {
		upgradeError = types.NewUpgradeError(channel.UpgradeSequence, err)
	}

	if err := k.writeErrorReceipt(ctx, portID, channelID, upgrade, upgradeError); err != nil {
		return err
	}

	return nil
}

// restoreChannel will restore the channel state and flush status to their pre-upgrade state so that upgrade is aborted.
func (k Keeper) restoreChannel(ctx sdk.Context, portID, channelID string, upgradeSequence uint64, currentChannel types.Channel) {
	currentChannel.State = types.OPEN
	currentChannel.FlushStatus = types.NOTINFLUSH
	currentChannel.UpgradeSequence = upgradeSequence

	k.SetChannel(ctx, portID, channelID, currentChannel)

	// delete state associated with upgrade which is no longer required.
	k.deleteUpgrade(ctx, portID, channelID)
	k.deleteCounterpartyLastPacketSequence(ctx, portID, channelID)
}

// writeErrorReceipt will write an error receipt from the provided UpgradeError.
func (k Keeper) writeErrorReceipt(ctx sdk.Context, portID, channelID string, upgrade types.Upgrade, upgradeError *types.UpgradeError) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	k.SetUpgradeErrorReceipt(ctx, portID, channelID, upgradeError.GetErrorReceipt())
	emitErrorReceiptEvent(ctx, portID, channelID, channel, upgrade, upgradeError)
	return nil
}
