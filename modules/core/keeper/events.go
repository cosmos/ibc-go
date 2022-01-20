package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	connectiontypes "github.com/cosmos/ibc-go/v3/modules/core/03-connection/types"
)

// EmitOpenConnectionInitEvent emits an open connection init event
func EmitOpenConnectionInitEvent(ctx sdk.Context, connectionID string, msg *connectiontypes.MsgConnectionOpenInit) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			connectiontypes.EventTypeConnectionOpenInit,
			sdk.NewAttribute(connectiontypes.AttributeKeyConnectionID, connectionID),
			sdk.NewAttribute(connectiontypes.AttributeKeyClientID, msg.ClientId),
			sdk.NewAttribute(connectiontypes.AttributeKeyCounterpartyClientID, msg.Counterparty.ClientId),
			sdk.NewAttribute(connectiontypes.AttributeKeyCounterpartyConnectionID, msg.Counterparty.ConnectionId),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, connectiontypes.AttributeValueCategory),
		),
	})
}

// EmitOpenConnectionOpenTryEvent emits an open connection try event
func EmitOpenConnectionOpenTryEvent(ctx sdk.Context, connectionID string, msg *connectiontypes.MsgConnectionOpenTry) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			connectiontypes.EventTypeConnectionOpenTry,
			sdk.NewAttribute(connectiontypes.AttributeKeyConnectionID, connectionID),
			sdk.NewAttribute(connectiontypes.AttributeKeyClientID, msg.ClientId),
			sdk.NewAttribute(connectiontypes.AttributeKeyCounterpartyClientID, msg.Counterparty.ClientId),
			sdk.NewAttribute(connectiontypes.AttributeKeyCounterpartyConnectionID, msg.Counterparty.ConnectionId),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, connectiontypes.AttributeValueCategory),
		),
	})
}

// EmitOpenConnectionOpenAckEvent emits an open connection try event
func EmitConnectionOpenAckEvent(ctx sdk.Context, connectionID string, connectionEnd connectiontypes.ConnectionEnd) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			connectiontypes.EventTypeConnectionOpenAck,
			sdk.NewAttribute(connectiontypes.AttributeKeyConnectionID, connectionID),
			sdk.NewAttribute(connectiontypes.AttributeKeyClientID, connectionEnd.ClientId),
			sdk.NewAttribute(connectiontypes.AttributeKeyCounterpartyClientID, connectionEnd.Counterparty.ClientId),
			sdk.NewAttribute(connectiontypes.AttributeKeyCounterpartyConnectionID, connectionEnd.Counterparty.ConnectionId),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, connectiontypes.AttributeValueCategory),
		),
	})
}

// EmitOpenConnectionOpenConfirmEvent emits an open connection try event
func EmitConnectionOpenConfirmEvent(ctx sdk.Context, connectionID string, connectionEnd connectiontypes.ConnectionEnd) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			connectiontypes.EventTypeConnectionOpenConfirm,
			sdk.NewAttribute(connectiontypes.AttributeKeyConnectionID, connectionID),
			sdk.NewAttribute(connectiontypes.AttributeKeyClientID, connectionEnd.ClientId),
			sdk.NewAttribute(connectiontypes.AttributeKeyCounterpartyClientID, connectionEnd.Counterparty.ClientId),
			sdk.NewAttribute(connectiontypes.AttributeKeyCounterpartyConnectionID, connectionEnd.Counterparty.ConnectionId),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, connectiontypes.AttributeValueCategory),
		),
	})
}
