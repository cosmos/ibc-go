package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// EmitCreateClientEvent emits a create client event
func EmitCreateClientEvent(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientState.ClientType()),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, clientState.GetLatestHeight().String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// EmitUpdateClientEvent emits an update client event
func EmitUpdateClientEvent(ctx sdk.Context, clientID string, clientType string, consensusHeights []exported.Height, clientMsgStr string) {
	var consensusHeightStr []string
	for _, height := range consensusHeights {
		consensusHeightStr = append(consensusHeightStr, height.String())
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientType),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, consensusHeights[0].String()),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, strings.Join(consensusHeightStr, ",")),
			sdk.NewAttribute(types.AttributeKeyHeader, clientMsgStr),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// EmitUpdateClientEvent emits an upgrade client event
func EmitUpgradeClientEvent(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpgradeClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientState.ClientType()),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, clientState.GetLatestHeight().String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// EmitUpdateClientProposalEvent emits an update client proposal event
func EmitUpdateClientProposalEvent(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateClientProposal,
			sdk.NewAttribute(types.AttributeKeySubjectClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientState.ClientType()),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, clientState.GetLatestHeight().String()),
		),
	)
}

// EmitSubmitMisbehaviourEvent emits a client misbehaviour event
func EmitSubmitMisbehaviourEvent(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSubmitMisbehaviour,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientState.ClientType()),
		),
	)
}
