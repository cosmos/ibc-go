package keeper

import (
	"fmt"
	"reflect"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	portkeeper "github.com/cosmos/ibc-go/v7/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

const (
	// restoreErrorString defines a string constant included in error receipts.
	// NOTE: Changing this const is state machine breaking as it is written into state.
	restoreErrorString = "restored channel to pre-upgrade state"
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
) (uint64, string, error) {
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

// RestoreChannel restores the given channel to the state prior to upgrade.
func (k Keeper) RestoreChannel(ctx sdk.Context, portID, channelID string, upgradeSequence uint64) error {

	_, code, _ := errorsmod.ABCIInfo(types.ErrChannelRestored, false) // discard non-determinstic codespace and log values

	errorReceipt := types.ErrorReceipt{
		Sequence: upgradeSequence,
		Error:    fmt.Sprintf("ABCI code: %d: %s", code, restoreErrorString),
	}

	k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)

	channel, found := k.GetUpgradeRestoreChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrRestoreChannelNotFound, "channel-id: %s", channelID)
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
