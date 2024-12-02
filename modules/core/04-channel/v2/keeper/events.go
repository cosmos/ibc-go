package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// emitSendPacketEvents emits events for the SendPacket handler.
func emitSendPacketEvents(ctx context.Context, packet types.Packet) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSendPacket,
			sdk.NewAttribute(types.AttributeKeySrcChannel, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeyDstChannel, packet.DestinationChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyPacketData, hex.EncodeToString(encodedPacket)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitRecvPacketEvents emits events for the RecvPacket handler.
func emitRecvPacketEvents(ctx context.Context, packet types.Packet) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecvPacket,
			sdk.NewAttribute(types.AttributeKeySrcChannel, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeyDstChannel, packet.DestinationChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyPacketData, hex.EncodeToString(encodedPacket)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// EmitWriteAcknowledgementEvents emits events for WriteAcknowledgement.
func EmitWriteAcknowledgementEvents(ctx context.Context, packet types.Packet, ack types.Acknowledgement) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	encodedAck, err := proto.Marshal(&ack)
	if err != nil {
		panic(err)
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeWriteAck,
			sdk.NewAttribute(types.AttributeKeySrcChannel, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeyDstChannel, packet.DestinationChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyPacketData, hex.EncodeToString(encodedPacket)),
			sdk.NewAttribute(types.AttributeKeyAckData, hex.EncodeToString(encodedAck)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// EmitAcknowledgePacketEvents emits events for the AcknowledgePacket handler.
func EmitAcknowledgePacketEvents(ctx context.Context, packet types.Packet) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAcknowledgePacket,
			sdk.NewAttribute(types.AttributeKeySrcChannel, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeyDstChannel, packet.DestinationChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyPacketData, hex.EncodeToString(encodedPacket)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// EmitTimeoutPacketEvents emits events for the TimeoutPacket handler.
func EmitTimeoutPacketEvents(ctx context.Context, packet types.Packet) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTimeoutPacket,
			sdk.NewAttribute(types.AttributeKeySrcChannel, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeyDstChannel, packet.DestinationChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyPacketData, hex.EncodeToString(encodedPacket)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
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
