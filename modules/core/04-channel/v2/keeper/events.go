package keeper

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

// emitSendPacketEvents emits events for the SendPacket handler.
func emitSendPacketEvents(ctx sdk.Context, packet types.Packet) {
	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSendPacket,
			sdk.NewAttribute(types.AttributeKeySrcClient, packet.SourceClient),
			sdk.NewAttribute(types.AttributeKeyDstClient, packet.DestinationClient),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyEncodedPacketHex, hex.EncodeToString(encodedPacket)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitRecvPacketEvents emits events for the RecvPacket handler.
func emitRecvPacketEvents(ctx sdk.Context, packet types.Packet) {
	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecvPacket,
			sdk.NewAttribute(types.AttributeKeySrcClient, packet.SourceClient),
			sdk.NewAttribute(types.AttributeKeyDstClient, packet.DestinationClient),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyEncodedPacketHex, hex.EncodeToString(encodedPacket)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitWriteAcknowledgementEvents emits events for WriteAcknowledgement.
func emitWriteAcknowledgementEvents(ctx sdk.Context, packet types.Packet, ack types.Acknowledgement) {
	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	encodedAck, err := proto.Marshal(&ack)
	if err != nil {
		panic(err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeWriteAck,
			sdk.NewAttribute(types.AttributeKeySrcClient, packet.SourceClient),
			sdk.NewAttribute(types.AttributeKeyDstClient, packet.DestinationClient),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyEncodedPacketHex, hex.EncodeToString(encodedPacket)),
			sdk.NewAttribute(types.AttributeKeyEncodedAckHex, hex.EncodeToString(encodedAck)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitAcknowledgePacketEvents emits events for the AcknowledgePacket handler.
func emitAcknowledgePacketEvents(ctx sdk.Context, packet types.Packet) {
	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAcknowledgePacket,
			sdk.NewAttribute(types.AttributeKeySrcClient, packet.SourceClient),
			sdk.NewAttribute(types.AttributeKeyDstClient, packet.DestinationClient),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyEncodedPacketHex, hex.EncodeToString(encodedPacket)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitTimeoutPacketEvents emits events for the TimeoutPacket handler.
func emitTimeoutPacketEvents(ctx sdk.Context, packet types.Packet) {
	encodedPacket, err := proto.Marshal(&packet)
	if err != nil {
		panic(err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTimeoutPacket,
			sdk.NewAttribute(types.AttributeKeySrcClient, packet.SourceClient),
			sdk.NewAttribute(types.AttributeKeyDstClient, packet.DestinationClient),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.TimeoutTimestamp)),
			sdk.NewAttribute(types.AttributeKeyEncodedPacketHex, hex.EncodeToString(encodedPacket)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}
