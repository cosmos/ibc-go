package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// emitSendPacketEvents emits events for the SendPacket handler.
func emitSendPacketEvents(ctx context.Context, packet types.Packet) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSendPacket,
			sdk.NewAttribute(types.AttributeKeySrcChannel, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeyDstChannel, packet.DestinationChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyPayloadLength, fmt.Sprintf("%d", len(packet.Payloads))),
			sdk.NewAttribute(types.AttributeKeyVersion, packet.Payloads[0].Version),
			sdk.NewAttribute(types.AttributeKeyEncoding, packet.Payloads[0].Encoding),
			sdk.NewAttribute(types.AttributeKeyData, hex.EncodeToString(packet.Payloads[0].Value)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})

	for i, payload := range packet.Payloads {
		sdkCtx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				types.EventTypeSendPayload,
				sdk.NewAttribute(types.AttributeKeySrcChannel, packet.SourceChannel),
				sdk.NewAttribute(types.AttributeKeyDstChannel, packet.DestinationChannel),
				sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
				sdk.NewAttribute(types.AttributeKeyPayloadSequence, fmt.Sprintf("%d", i)),
				sdk.NewAttribute(types.AttributeKeyVersion, payload.Version),
				sdk.NewAttribute(types.AttributeKeyEncoding, payload.Encoding),
				sdk.NewAttribute(types.AttributeKeyData, hex.EncodeToString(payload.Value)),
			),
			sdk.NewEvent(
				sdk.EventTypeMessage,
				sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			),
		})
	}
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
