package keeper

import (
	"context"

	"cosmossdk.io/core/event"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
)

// emitConnectionOpenInitEvent emits a connection open init event
func (k *Keeper) emitConnectionOpenInitEvent(ctx context.Context, connectionID string, clientID string, counterparty types.Counterparty) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeConnectionOpenInit,
		event.NewAttribute(types.AttributeKeyConnectionID, connectionID),
		event.NewAttribute(types.AttributeKeyClientID, clientID),
		event.NewAttribute(types.AttributeKeyCounterpartyClientID, counterparty.ClientId),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitConnectionOpenTryEvent emits a connection open try event
func (k *Keeper) emitConnectionOpenTryEvent(ctx context.Context, connectionID string, clientID string, counterparty types.Counterparty) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeConnectionOpenTry,
		event.NewAttribute(types.AttributeKeyConnectionID, connectionID),
		event.NewAttribute(types.AttributeKeyClientID, clientID),
		event.NewAttribute(types.AttributeKeyCounterpartyClientID, counterparty.ClientId),
		event.NewAttribute(types.AttributeKeyCounterpartyConnectionID, counterparty.ConnectionId),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitConnectionOpenAckEvent emits a connection open acknowledge event
func (k *Keeper) emitConnectionOpenAckEvent(ctx context.Context, connectionID string, connectionEnd types.ConnectionEnd) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeConnectionOpenAck,
		event.NewAttribute(types.AttributeKeyConnectionID, connectionID),
		event.NewAttribute(types.AttributeKeyClientID, connectionEnd.ClientId),
		event.NewAttribute(types.AttributeKeyCounterpartyClientID, connectionEnd.Counterparty.ClientId),
		event.NewAttribute(types.AttributeKeyCounterpartyConnectionID, connectionEnd.Counterparty.ConnectionId),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitConnectionOpenConfirmEvent emits a connection open confirm event
func (k *Keeper) emitConnectionOpenConfirmEvent(ctx context.Context, connectionID string, connectionEnd types.ConnectionEnd) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeConnectionOpenConfirm,
		event.NewAttribute(types.AttributeKeyConnectionID, connectionID),
		event.NewAttribute(types.AttributeKeyClientID, connectionEnd.ClientId),
		event.NewAttribute(types.AttributeKeyCounterpartyClientID, connectionEnd.Counterparty.ClientId),
		event.NewAttribute(types.AttributeKeyCounterpartyConnectionID, connectionEnd.Counterparty.ConnectionId),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}
