package events

import (
	"encoding/base64"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

// EmitSendCall emits a GMP send call event.
func EmitSendCall(
	ctx sdk.Context,
	packetData types.GMPPacketData,
	sourceClient,
	destinationClient,
	sourcePort,
	destinationPort string,
	sequence uint64,
) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSendCall,
			packetAttributes(packetData, sourceClient, destinationClient, sourcePort, destinationPort, sequence)...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})
}

// EmitOnRecvPacketEvent emits a GMP packet event in the OnRecvPacket callback.
func EmitOnRecvPacketEvent(
	ctx sdk.Context,
	packetData types.GMPPacketData,
	sourceClient,
	destinationClient,
	sourcePort,
	destinationPort string,
	sequence uint64,
	ackErr error,
) {
	attributes := packetAttributes(packetData, sourceClient, destinationClient, sourcePort, destinationPort, sequence)
	attributes = append(attributes, sdk.NewAttribute(types.AttributeKeyAckSuccess, strconv.FormatBool(ackErr == nil)))
	if ackErr != nil {
		attributes = append(attributes, sdk.NewAttribute(types.AttributeKeyAckError, ackErr.Error()))
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecvPacket,
			attributes...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})
}

func packetAttributes(
	packetData types.GMPPacketData,
	sourceClient,
	destinationClient,
	sourcePort,
	destinationPort string,
	sequence uint64,
) []sdk.Attribute {
	return []sdk.Attribute{
		sdk.NewAttribute(types.AttributeKeySender, packetData.Sender),
		sdk.NewAttribute(types.AttributeKeyReceiver, packetData.Receiver),
		sdk.NewAttribute(types.AttributeKeySalt, base64.StdEncoding.EncodeToString(packetData.Salt)),
		sdk.NewAttribute(types.AttributeKeyPayload, base64.StdEncoding.EncodeToString(packetData.Payload)),
		sdk.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
		sdk.NewAttribute(types.AttributeKeySourceClient, sourceClient),
		sdk.NewAttribute(types.AttributeKeyDestinationClient, destinationClient),
		sdk.NewAttribute(types.AttributeKeySourcePort, sourcePort),
		sdk.NewAttribute(types.AttributeKeyDestinationPort, destinationPort),
		sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(sequence, 10)),
	}
}
