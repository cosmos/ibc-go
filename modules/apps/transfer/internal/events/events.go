package events

import (
	"encoding/json"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// EmitTransferEvent emits a ibc transfer event on successful transfers.
func EmitTransferEvent(ctx sdk.Context, sender, receiver string, token types.Token, memo string) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTransfer,
			sdk.NewAttribute(types.AttributeKeySender, sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, receiver),
			sdk.NewAttribute(types.AttributeKeyDenom, token.Denom.Path()),
			sdk.NewAttribute(types.AttributeKeyAmount, token.Amount),
			sdk.NewAttribute(types.AttributeKeyMemo, memo),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})
}

// EmitOnRecvPacketEvent emits a fungible token packet event in the OnRecvPacket callback
func EmitOnRecvPacketEvent(ctx sdk.Context, packetData types.InternalTransferRepresentation, ack ibcexported.Acknowledgement, ackErr error) {
	eventAttributes := []sdk.Attribute{
		sdk.NewAttribute(types.AttributeKeySender, packetData.Sender),
		sdk.NewAttribute(types.AttributeKeyReceiver, packetData.Receiver),
		sdk.NewAttribute(types.AttributeKeyDenom, packetData.Token.Denom.Path()),
		sdk.NewAttribute(types.AttributeKeyAmount, packetData.Token.Amount),
		sdk.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
		sdk.NewAttribute(types.AttributeKeyAckSuccess, strconv.FormatBool(ack.Success())),
	}

	if ackErr != nil {
		eventAttributes = append(eventAttributes, sdk.NewAttribute(types.AttributeKeyAckError, ackErr.Error()))
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypePacket,
			eventAttributes...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})
}

// EmitOnAcknowledgementPacketEvent emits a fungible token packet event in the OnAcknowledgementPacket callback
func EmitOnAcknowledgementPacketEvent(ctx sdk.Context, packetData types.InternalTransferRepresentation, ack channeltypes.Acknowledgement) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypePacket,
			sdk.NewAttribute(sdk.AttributeKeySender, packetData.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, packetData.Receiver),
			sdk.NewAttribute(types.AttributeKeyDenom, packetData.Token.Denom.Path()),
			sdk.NewAttribute(types.AttributeKeyAmount, packetData.Token.Amount),
			sdk.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
			sdk.NewAttribute(types.AttributeKeyAck, ack.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})

	switch resp := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePacket,
				sdk.NewAttribute(types.AttributeKeyAckSuccess, string(resp.Result)),
			),
		)
	case *channeltypes.Acknowledgement_Error:
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePacket,
				sdk.NewAttribute(types.AttributeKeyAckError, resp.Error),
			),
		)
	}
}

// EmitOnTimeoutEvent emits a fungible token packet event in the OnTimeoutPacket callback
func EmitOnTimeoutEvent(ctx sdk.Context, packetData types.InternalTransferRepresentation) {
	tokenStr := mustMarshalJSON(packetData.Token)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTimeout,
			sdk.NewAttribute(types.AttributeKeyReceiver, packetData.Sender),
			sdk.NewAttribute(types.AttributeKeyRefundTokens, tokenStr),
			sdk.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})
}

// EmitDenomEvent emits a denomination event in the OnRecv callback.
func EmitDenomEvent(ctx sdk.Context, token types.Token) {
	denomStr := mustMarshalJSON(token.Denom)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDenom,
			sdk.NewAttribute(types.AttributeKeyDenomHash, token.Denom.Hash().String()),
			sdk.NewAttribute(types.AttributeKeyDenom, denomStr),
		),
	)
}

// mustMarshalJSON json marshals the given type and panics on failure.
func mustMarshalJSON(v any) string {
	bz, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(bz)
}
