package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
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

// WriteUpgradeTryChannel writes a channel which has successfully passed the UpgradeTry step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTryChannel(
	ctx sdk.Context,
	portID, channelID string,
	proposedUpgrade types.Upgrade,
) {
	// TODO
	// grab channel inside this function to get most current channel status
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

// startFlushUpgradeSequence will verify that the channel is in a valid precondition for calling the startFlushUpgradeHandshake
// and that the desiredChannelState is valid
// it will verify the proofs of the counterparty channel and upgrade
// it will verify that the upgrades on both ends are mutually compatible
// it will set the channel to desiredChannel state and move to flushing mode
// if flush is already complete, it will automatically set flushStatus to FLUSHCOMPLETE
//
//lint:ignore U1000 Ignore unused function temporarily for debugging
func (k Keeper) startFlushUpgradeHandshake(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedUpgradeFields types.UpgradeFields,
	counterpartyChannel types.Channel,
	counterpartyUpgrade types.Upgrade,
	desiredChannelState types.State,
	// TODO: add flush state here when enum is present
	proofCounterpartyChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
) error {

	if !collections.Contains(desiredChannelState, []types.State{types.TRYUPGRADE, types.ACKUPGRADE}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.TRYUPGRADE, types.ACKUPGRADE, desiredChannelState)
	}

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	connection, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return errorsmod.Wrap(err, "failed to retrieve connection using the channel connection hops")
	}

	// make sure connection is OPEN
	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	// verify the counterparty channel state containing the upgrade sequence
	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight,
		proofCounterpartyChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return err
	}

	// verifies the proof that a particular proposed upgrade has been stored in the upgrade path of the counterparty
	if err := k.connectionKeeper.VerifyChannelUpgrade(ctx, connection, proofHeight, proofUpgrade, channel.Counterparty.PortId,
		channel.Counterparty.ChannelId, counterpartyUpgrade); err != nil {
		return err
	}

	// ensure that the upgrade sequences match.
	if counterpartyChannel.UpgradeSequence != channel.UpgradeSequence {
		// error on the higher sequence so that both chains move to a fresh sequence.
		channel.UpgradeSequence = math.Max(counterpartyChannel.UpgradeSequence, channel.UpgradeSequence)
		k.SetChannel(ctx, portID, channelID, channel)
		k.restoreChannel(portID, channelID)
		// TODO: return error receipt here, not an error.
		return errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "expected upgrade sequence (%d) to match counterparty upgrade sequence (%d)", channel.UpgradeSequence, counterpartyChannel.UpgradeSequence)
	}

	//if !reflect.DeepEqual(proposedUpgradeFields, counterpartyUpgrade.Fields) {
	//	k.restoreChannel(portID, channelID)
	//	// TODO: return error receipt
	//}

	// ensure upgrade fields ordering is the same.
	if proposedUpgradeFields.Ordering != counterpartyUpgrade.Fields.Ordering {
		k.restoreChannel(portID, channelID)
		// TODO: return error receipt
	}

	// connectionHops can change in a channelUpgrade, however both sides must still be each other's counterparty.
	proposedConnection, found := k.connectionKeeper.GetConnection(ctx, proposedUpgradeFields.ConnectionHops[0])
	if !found {
		k.restoreChannel(portID, channelID)
		// TODO: return error receipt
	}

	if proposedConnection.GetState() != int32(connectiontypes.OPEN) {
		k.restoreChannel(portID, channelID)
		// TODO: return error receipt
	}

	if counterpartyUpgrade.Fields.ConnectionHops[0] != proposedConnection.Counterparty.ConnectionId {
		k.restoreChannel(portID, channelID)
		// TODO: return error receipt
	}

	// set the channel to the desired state
	channel.State = desiredChannelState
	// TODO: channel.FlushState = FLUSHING

	if len(k.pendingInflightPackets(portID, channelID)) == 0 {
		// if there are no packets in flight, then flush is complete
		// TODO: channel.FlushState = FLUSHCOMPLETE
	}

	k.SetChannel(ctx, portID, channelID, channel)
	// TODO: k.SetChannelCounterpartyLastPacketSequenceSend(portID, channelID, counterpartyUpgrade.LatestSequenceSend)

	return nil
}

// restoreChannel will write an error receipt, set the channel back to its original state and
// delete upgrade information when the executing channel needs to abort the upgrade handshake and return to the original parameters.
func (k Keeper) restoreChannel(portID, channelID string) {
	// TODO
}

// pendingInflightPackets returns the packet sequences sent on this end that have not had their lifecycle completed
//
//lint:ignore U1000 Ignore unused function temporarily for debugging
func (k Keeper) pendingInflightPackets(portID, channelID string) []uint64 {
	// TODO
	return []uint64{}
}
