package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// SendTx takes pre-built packet data containing messages to be executed on the host chain from an authentication module and attempts to send the packet.
// The packet sequence for the outgoing packet is returned as a result. An appropriate
// absolute timeoutTimestamp must be provided. If the packet is timed out, the channel will be closed.
// In the case of channel closure, a new channel may be reopened to reconnect to the host chain.
//
// Deprecated: this is a legacy API that is only intended to function correctly in workflows where an underlying application has been set.
// Prior to v6.x.x of ibc-go, the controller module was only functional as middleware, with authentication performed
// by the underlying application. For a full summary of the changes in v6.x.x, please see ADR009.
// This API will be removed in later releases.
func (k Keeper) SendTx(ctx context.Context, connectionID, portID string, icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64) (uint64, error) {
	return k.sendTx(ctx, connectionID, portID, icaPacketData, timeoutTimestamp)
}

func (k Keeper) sendTx(ctx context.Context, connectionID, portID string, icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64) (uint64, error) {
	if !k.GetParams(ctx).ControllerEnabled {
		return 0, types.ErrControllerSubModuleDisabled
	}

	activeChannelID, found := k.GetOpenActiveChannel(ctx, connectionID, portID)
	if !found {
		return 0, errorsmod.Wrapf(icatypes.ErrActiveChannelNotFound, "failed to retrieve active channel on connection %s for port %s", connectionID, portID)
	}

	blockTime := k.HeaderService.HeaderInfo(ctx).Time.UnixNano()
	if uint64(blockTime) >= timeoutTimestamp {
		return 0, icatypes.ErrInvalidTimeoutTimestamp
	}

	if err := icaPacketData.ValidateBasic(); err != nil {
		return 0, errorsmod.Wrap(err, "invalid interchain account packet data")
	}

	sequence, err := k.ics4Wrapper.SendPacket(ctx, portID, activeChannelID, clienttypes.ZeroHeight(), timeoutTimestamp, icaPacketData.GetBytes())
	if err != nil {
		return 0, err
	}

	return sequence, nil
}

// OnTimeoutPacket removes the active channel associated with the provided packet, the underlying channel end is closed
// due to the semantics of ORDERED channels
func (Keeper) OnTimeoutPacket(ctx context.Context, packet channeltypes.Packet) error {
	return nil
}
