package keeper

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

// SendTx takes pre-built packet data containing messages to be executed on the host chain from an authentication module and attempts to send the packet.
// The packet sequence for the outgoing packet is returned as a result.
// If the base application has the capability to send on the provided portID. An appropriate
// absolute timeoutTimestamp must be provided. If the packet is timed out, the channel will be closed.
// In the case of channel closure, a new channel may be reopened to reconnect to the host chain.
//
// Deprecated: this is a legacy API that is only intended to function correctly in workflows where an underlying application has been set.
// Prior to v6.x.x of ibc-go, the controller module was only functional as middleware, with authentication performed
// by the underlying application. For a full summary of the changes in v6.x.x, please see ADR009.
// This API will be removed in later releases.
func (k Keeper) SendTx(ctx sdk.Context, _ *capabilitytypes.Capability, connectionID, portID string, icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64) (uint64, error) {
	return k.sendTx(ctx, connectionID, portID, icaPacketData, timeoutTimestamp, "")
}

func (k Keeper) sendTx(ctx sdk.Context, connectionID, portID string, icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64, owner string) (uint64, error) {
	activeChannelID, found := k.GetOpenActiveChannel(ctx, connectionID, portID)
	if !found {
		return 0, errorsmod.Wrapf(icatypes.ErrActiveChannelNotFound, "failed to retrieve active channel on connection %s for port %s", connectionID, portID)
	}

	if strings.TrimSpace(owner) == "" {
		owner = strings.TrimPrefix(portID, icatypes.ControllerPortPrefix)
	}

	msgSendPacket := &channeltypes.MsgSendPacket{
		PortId:           portID,
		ChannelId:        activeChannelID,
		TimeoutHeight:    clienttypes.ZeroHeight(),
		TimeoutTimestamp: timeoutTimestamp,
		Data:             icaPacketData.GetBytes(),
		Signer:           owner,
	}

	handler := k.msgRouter.Handler(msgSendPacket)
	res, err := handler(ctx, msgSendPacket)
	if err != nil {
		return 0, err
	}

	sendPacketResp, ok := res.MsgResponses[0].GetCachedValue().(*channeltypes.MsgSendPacketResponse)
	if !ok {
		return 0, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to convert %T message response to %T", res.MsgResponses[0].GetCachedValue(), &channeltypes.MsgSendPacketResponse{})
	}

	// NOTE: The sdk msg handler creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(res.GetEvents())

	return sendPacketResp.Sequence, nil
}

// OnTimeoutPacket removes the active channel associated with the provided packet, the underlying channel end is closed
// due to the semantics of ORDERED channels
func (Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	return nil
}
