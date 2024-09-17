package keeper

import (
	"fmt"
	"strconv"
	"strings"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// emitCreateClientEvent emits a create client event
func emitCreateClientEvent(ctx sdk.Context, clientID, clientType string, initialHeight exported.Height) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientType),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, initialHeight.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitUpdateClientEvent emits an update client event
func emitUpdateClientEvent(ctx sdk.Context, clientID string, clientType string, consensusHeights []exported.Height, _ codec.BinaryCodec, _ exported.ClientMessage) {
	var consensusHeightAttr string
	if len(consensusHeights) != 0 {
		consensusHeightAttr = consensusHeights[0].String()
	}

	consensusHeightsAttr := make([]string, len(consensusHeights))
	for i, height := range consensusHeights {
		consensusHeightsAttr[i] = height.String()
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientType),
			// Deprecated: AttributeKeyConsensusHeight is deprecated and will be removed in a future release.
			// Please use AttributeKeyConsensusHeights instead.
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, consensusHeightAttr),
			sdk.NewAttribute(types.AttributeKeyConsensusHeights, strings.Join(consensusHeightsAttr, ",")),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitUpgradeClientEvent emits an upgrade client event
func emitUpgradeClientEvent(ctx sdk.Context, clientID, clientType string, latestHeight exported.Height) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpgradeClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientType),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, latestHeight.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitSubmitMisbehaviourEvent emits a client misbehaviour event
func emitSubmitMisbehaviourEvent(ctx sdk.Context, clientID string, clientType string) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSubmitMisbehaviour,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientType),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitRecoverClientEvent emits a recover client event
func emitRecoverClientEvent(ctx sdk.Context, clientID, clientType string) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecoverClient,
			sdk.NewAttribute(types.AttributeKeySubjectClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientType),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitScheduleIBCSoftwareUpgradeEvent emits a schedule IBC software upgrade event
func emitScheduleIBCSoftwareUpgradeEvent(ctx sdk.Context, title string, height int64) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeScheduleIBCSoftwareUpgrade,
			sdk.NewAttribute(types.AttributeKeyUpgradePlanTitle, title),
			sdk.NewAttribute(types.AttributeKeyUpgradePlanHeight, fmt.Sprintf("%d", height)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// EmitUpgradeChainEvent emits an upgrade chain event.
func EmitUpgradeChainEvent(ctx sdk.Context, height int64) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpgradeChain,
			sdk.NewAttribute(types.AttributeKeyUpgradePlanHeight, strconv.FormatInt(height, 10)),
			sdk.NewAttribute(types.AttributeKeyUpgradeStore, upgradetypes.StoreKey), // which store to query proof of consensus state from
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}
