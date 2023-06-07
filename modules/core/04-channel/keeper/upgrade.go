package keeper

import (
	"fmt"
	"reflect"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

	if err := k.validateUpgradeFields(ctx, upgradeFields, channel); err != nil {
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

// upgradeTry
func (k Keeper) ChanUpgradeTry(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedConnectionHops []string,
	upgradeTimeout types.Timeout,
	counterpartyProposedUpgrade types.Upgrade,
	counterpartyUpgradeSequence uint64,
	proofCounterpartyChannel,
	proofCounterpartyUpgrade []byte,
	proofHeight clienttypes.Height,
) (types.Upgrade, error) {
	// TODO
	return types.Upgrade{}, nil
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

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeTry step, channelID: %s, portID: %s", channelID, portID))
	}

	previousState := channel.State
	channel.State = types.TRYUPGRADE
	channel.FlushStatus = flushStatus

	k.SetChannel(ctx, portID, channelID, channel)
	k.SetUpgrade(ctx, portID, channelID, proposedUpgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState, "new-state", types.TRYUPGRADE.String())
	emitChannelUpgradeTryEvent(ctx, portID, channelID, channel, proposedUpgrade)
}

// startFlushUpgradeHandshake will verify the counterparty proposed upgrade and the current channel state.
// Once the counterparty information has been verified, it will be validated against the self proposed upgrade.
// If any of the proposed upgrade fields are incompatible, an upgrade error will be returned resulting in an
// aborted upgrade.
//
//lint:ignore U1000 Ignore unused function temporarily for debugging
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
		connection,
		proofHeight, proofCounterpartyUpgrade,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyUpgrade,
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

	// assert that both sides propose the same channel ordering
	if proposedUpgradeFields.Ordering != counterpartyUpgrade.Fields.Ordering {
		return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrapf(
			types.ErrIncompatibleCounterpartyUpgrade, "expected upgrade ordering (%s) to match counterparty upgrade ordering (%s)", proposedUpgradeFields.Ordering, counterpartyUpgrade.Fields.Ordering),
		)
	}

	proposedConnection, err := k.GetConnection(ctx, proposedUpgradeFields.ConnectionHops[0])
	if err != nil {
		// NOTE: this error is expected to be unreachable as the proposed upgrade connectionID should have been
		// validated in the upgrade INIT and TRY handlers
		return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrap(
			err, "expected proposed connection to be found"),
		)
	}

	if proposedConnection.GetState() != int32(connectiontypes.OPEN) {
		// NOTE: this error is expected to be unreachable as the proposed upgrade connectionID should have been
		// validated in the upgrade INIT and TRY handlers
		return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState, "expected proposed connection to be OPEN (got %s)", connectiontypes.State(proposedConnection.GetState()).String()),
		)
	}

	// connectionHops can change in a channelUpgrade, however both sides must still be each other's counterparty.
	if counterpartyUpgrade.Fields.ConnectionHops[0] != proposedConnection.GetCounterparty().GetConnectionID() {
		return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrapf(
			types.ErrIncompatibleCounterpartyUpgrade, "counterparty upgrade connection end is not a counterparty of self proposed connection end (%s != %s)", counterpartyUpgrade.Fields.ConnectionHops[0], proposedConnection.GetCounterparty().GetConnectionID()),
		)
	}
	return nil
}

// validateUpgradeFields validates the proposed upgrade fields against the existing channel.
// It returns an error if the following constraints are not met:
// - there exists at least one valid proposed change to the existing channel fields
// - the proposed order is a subset of the existing order
// - the proposed connection hops do not exist
// - the proposed version is non-empty (checked in UpgradeFields.ValidateBasic())
// - the proposed connection hops are not open
func (k Keeper) validateUpgradeFields(ctx sdk.Context, proposedUpgrade types.UpgradeFields, currentChannel types.Channel) error {
	currentFields := extractUpgradeFields(currentChannel)

	if reflect.DeepEqual(proposedUpgrade, currentFields) {
		return errorsmod.Wrap(types.ErrChannelExists, "existing channel end is identical to proposed upgrade channel end")
	}

	connectionID := proposedUpgrade.ConnectionHops[0]
	connection, err := k.GetConnection(ctx, connectionID)
	if err != nil {
		return errorsmod.Wrapf(connectiontypes.ErrConnectionNotFound, "failed to retrieve connection: %s", connectionID)
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
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
	seq, found := k.GetNextSequenceSend(ctx, portID, channelID)
	if !found {
		return types.Upgrade{}, types.ErrSequenceSendNotFound
	}
	return types.Upgrade{
		Fields:             fields,
		Timeout:            upgradeTimeout,
		LatestSequenceSend: seq - 1,
	}, nil
}

// abortHandshake will restore the channel state and flush status to their pre-upgrade state so that upgrade is aborted.
// any unnecessary state is deleted. An error receipt is written, and the OnChanUpgradeRestore callback is called.
func (k Keeper) abortHandshake(ctx sdk.Context, portID, channelID string, upgradeError *types.UpgradeError) error {
	if upgradeError == nil {
		return errorsmod.Wrap(types.ErrInvalidUpgradeError, "cannot abort upgrade handshake with nil error")
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if err := k.restoreChannel(ctx, portID, channelID); err != nil {
		return err
	}

	if err := k.writeErrorReceipt(ctx, portID, channelID, upgrade, upgradeError); err != nil {
		return err
	}

	// TODO: callback execution
	// cbs.OnChanUpgradeRestore()

	return nil
}

// restoreChannel will restore the channel state and flush status to their pre-upgrade state so that upgrade is aborted
// It will write an error receipt to state so that the counterparty can restore as well.
func (k Keeper) restoreChannel(ctx sdk.Context, portID, channelID string) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	channel.State = types.OPEN
	channel.FlushStatus = types.NOTINFLUSH

	k.SetChannel(ctx, portID, channelID, channel)

	// delete state associated with upgrade which is no longer required.
	k.deleteUpgrade(ctx, portID, channelID)
	k.deleteCounterpartyLastPacketSequence(ctx, portID, channelID)

	return nil
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
