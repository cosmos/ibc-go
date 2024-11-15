package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// EmitSendPacketEvents emits events for the SendPacket handler.
func EmitSendPacketEvents(ctx context.Context, packet types.Packet) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitRecvPacketEvents emits events for the RecvPacket handler.
func EmitRecvPacketEvents(ctx context.Context, packet types.Packet) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitAcknowledgePacketEvents emits events for the AcknowledgePacket handler.
func EmitAcknowledgePacketEvents(ctx context.Context, packet types.Packet) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitTimeoutPacketEvents emits events for the TimeoutPacket handler.
func EmitTimeoutPacketEvents(ctx context.Context, packet types.Packet) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitWriteAcknowledgementEvents emits events for WriteAcknowledgement.
func EmitWriteAcknowledgementEvents(ctx context.Context, packet types.Packet, ack types.Acknowledgement) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// emitCreateChannelEvent emits a channel create event.
func (*Keeper) emitCreateChannelEvent(ctx context.Context, channelID, clientID string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateChannel,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitRegisterCounterpartyEvent emits a register counterparty event.
func (*Keeper) emitRegisterCounterpartyEvent(ctx context.Context, channelID string, channel types.Channel) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRegisterCounterparty,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyClientID, channel.ClientId),
			sdk.NewAttribute(types.AttributeKeyCounterpartyChannelID, channel.CounterpartyChannelId),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}
