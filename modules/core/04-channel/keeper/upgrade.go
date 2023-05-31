package keeper

import (
	"fmt"

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

// ChanUpgradeAck is called by a module to accept the ACKUPGRADE handshake step of the channel upgrade protocol.
func (k Keeper) ChanUpgradeAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyFlushStatus types.Order,
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

	// TODO: rebase with FlushStatus and update args and error return
	// abortTransactionUnless(counterpartyFlushStatus == FLUSHING || counterpartyFlushStatus == FLUSHCOMPLETE)
	if !collections.Contains(counterpartyFlushStatus, []types.Order{types.UNORDERED, types.ORDERED}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.INITUPGRADE, types.TRYUPGRADE, channel.State)
	}

	connectionEnd, err := k.GetConnection(ctx, channel.ConnectionHops[0])
	if err != nil {
		return errorsmod.Wrapf(err, "failed to retrieve connection: %s", channel.ConnectionHops[0])
	}

	counterpartyHops := connectionEnd.GetCounterparty().GetConnectionID()
	counterpartyChannel := types.Channel{
		State:           types.TRYUPGRADE,
		Ordering:        channel.Ordering,
		ConnectionHops:  []string{counterpartyHops},
		Counterparty:    types.NewCounterparty(portID, channelID),
		Version:         channel.Version,
		UpgradeSequence: channel.UpgradeSequence,
		// FlushStatus: counterpartyFlushStatus,
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", portID, channelID)
	}

	// in the crossing hellos case, the versions returned by both on TRY must be the same
	if channel.State == types.TRYUPGRADE {
		if upgrade.Fields.Version != counterpartyUpgrade.Fields.Version {
			// restoreChannel(portID, channelID)
			return errorsmod.Wrap(types.ErrInvalidUpgrade, "todo: return types.UpgradeError(channel.UpgradeSequence, err)")
		}
	}

	if err := k.startFlushUpgradeHandshake(ctx, portID, channelID, upgrade.Fields, counterpartyChannel, counterpartyUpgrade,
		proofChannel, proofUpgrade, proofHeight); err != nil {
		return err
	}

	return nil
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

func (k Keeper) startFlushUpgradeHandshake(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedUpgradeFields types.UpgradeFields,
	counterpartyChannel types.Channel,
	counterpartyUpgrade types.Upgrade,
	proofCounterpartyChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
) error {
	return nil
}
