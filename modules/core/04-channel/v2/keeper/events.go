package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// EmitSendPacketEvents emits events for the SendPacket handler.
func EmitSendPacketEvents(ctx context.Context, packet channeltypesv2.Packet) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitRecvPacketEvents emits events for the RecvPacket handler.
func EmitRecvPacketEvents(ctx context.Context, packet channeltypesv2.Packet) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitAcknowledgePacketEvents emits events for the AcknowledgePacket handler.
func EmitAcknowledgePacketEvents(ctx context.Context, packet channeltypesv2.Packet) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitTimeoutPacketEvents emits events for the TimeoutPacket handler.
func EmitTimeoutPacketEvents(ctx context.Context, packet channeltypesv2.Packet) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitWriteAcknowledgementEvents emits events for WriteAcknowledgement.
func EmitWriteAcknowledgementEvents(ctx context.Context, packet channeltypesv2.Packet, ack channeltypesv2.Acknowledgement) {
	// TODO: https://github.com/cosmos/ibc-go/issues/7386
}

// EmitCreateChannelEvent emits a channel create event.
func (*Keeper) EmitCreateChannelEvent(ctx context.Context, channelID string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			channeltypesv2.EventTypeCreateChannel,
			sdk.NewAttribute(channeltypesv2.AttributeKeyChannelID, channelID),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, channeltypesv2.AttributeValueCategory),
		),
	})
}

