package keeper

import (
	"context"
	"encoding/json"
	"strconv"

	"cosmossdk.io/core/event"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// EmitTransferEvent emits an ibc transfer event on successful transfers.
func (k Keeper) EmitTransferEvent(ctx context.Context, sender, receiver string, tokens types.Tokens, memo string, forwardingHops []types.Hop) error {
	tokensStr := mustMarshalJSON(tokens)
	forwardingHopsStr := mustMarshalJSON(forwardingHops)

	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeTransfer,
		event.NewAttribute(types.AttributeKeySender, sender),
		event.NewAttribute(types.AttributeKeyReceiver, receiver),
		event.NewAttribute(types.AttributeKeyTokens, tokensStr),
		event.NewAttribute(types.AttributeKeyMemo, memo),
		event.NewAttribute(types.AttributeKeyForwardingHops, forwardingHopsStr),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	)
}

// EmitOnRecvPacketEvent emits a fungible token packet event in the OnRecvPacket callback
func (k Keeper) EmitOnRecvPacketEvent(ctx context.Context, packetData types.FungibleTokenPacketDataV2, ack channeltypes.Acknowledgement, ackErr error) error {
	tokensStr := mustMarshalJSON(packetData.Tokens)
	forwardingHopStr := mustMarshalJSON(packetData.Forwarding.Hops)

	eventAttributes := []event.Attribute{
		event.NewAttribute(types.AttributeKeySender, packetData.Sender),
		event.NewAttribute(types.AttributeKeyReceiver, packetData.Receiver),
		event.NewAttribute(types.AttributeKeyTokens, tokensStr),
		event.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
		event.NewAttribute(types.AttributeKeyForwardingHops, forwardingHopStr),
		event.NewAttribute(types.AttributeKeyAckSuccess, strconv.FormatBool(ack.Success())),
	}

	if ackErr != nil {
		eventAttributes = append(eventAttributes, event.NewAttribute(types.AttributeKeyAckError, ackErr.Error()))
	}

	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypePacket,
		eventAttributes...,
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	)
}

// EmitOnAcknowledgementPacketEvent emits a fungible token packet event in the OnAcknowledgementPacket callback
func (k Keeper) EmitOnAcknowledgementPacketEvent(ctx context.Context, packetData types.FungibleTokenPacketDataV2, ack channeltypes.Acknowledgement) error {
	tokensStr := mustMarshalJSON(packetData.Tokens)
	forwardingHopsStr := mustMarshalJSON(packetData.Forwarding.Hops)

	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypePacket,
		event.NewAttribute(sdk.AttributeKeySender, packetData.Sender),
		event.NewAttribute(types.AttributeKeyReceiver, packetData.Receiver),
		event.NewAttribute(types.AttributeKeyTokens, tokensStr),
		event.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
		event.NewAttribute(types.AttributeKeyForwardingHops, forwardingHopsStr),
		event.NewAttribute(types.AttributeKeyAck, ack.String()),
	); err != nil {
		return err
	}

	switch resp := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		if err := k.EventService.EventManager(ctx).EmitKV(
			types.EventTypePacket,
			event.NewAttribute(types.AttributeKeyAckSuccess, string(resp.Result)),
		); err != nil {
			return err
		}
	case *channeltypes.Acknowledgement_Error:
		if err := k.EventService.EventManager(ctx).EmitKV(
			types.EventTypePacket,
			event.NewAttribute(types.AttributeKeyAckError, resp.Error),
		); err != nil {
			return err
		}
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	)
}

// EmitOnTimeoutEvent emits a fungible token packet event in the OnTimeoutPacket callback
func (k Keeper) EmitOnTimeoutEvent(ctx context.Context, packetData types.FungibleTokenPacketDataV2) error {
	tokensStr := mustMarshalJSON(packetData.Tokens)
	forwardingHopsStr := mustMarshalJSON(packetData.Forwarding.Hops)

	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeTimeout,
		event.NewAttribute(types.AttributeKeyReceiver, packetData.Sender),
		event.NewAttribute(types.AttributeKeyRefundTokens, tokensStr),
		event.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
		event.NewAttribute(types.AttributeKeyForwardingHops, forwardingHopsStr),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	)
}

// EmitDenomEvent emits a denomination event in the OnRecv callback.
func (k Keeper) EmitDenomEvent(ctx context.Context, token types.Token) error {
	return k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeDenom,
		event.NewAttribute(types.AttributeKeyDenomHash, token.Denom.Hash().String()),
		event.NewAttribute(types.AttributeKeyDenom, mustMarshalJSON(token.Denom)),
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
