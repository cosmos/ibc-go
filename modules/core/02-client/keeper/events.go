package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/event"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// emitCreateClientEvent emits a create client event
func emitCreateClientEvent(ctx context.Context, env appmodule.Environment, clientID, clientType string, initialHeight exported.Height) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeCreateClient,
		event.Attribute{Key: types.AttributeKeyClientID, Value: clientID},
		event.Attribute{Key: types.AttributeKeyClientType, Value: clientType},
		event.Attribute{Key: types.AttributeKeyConsensusHeight, Value: initialHeight.String()},
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.AttributeValueCategory},
	)
}

// emitUpdateClientEvent emits an update client event
func emitUpdateClientEvent(ctx context.Context, env appmodule.Environment, clientID string, clientType string, consensusHeights []exported.Height, _ codec.BinaryCodec, _ exported.ClientMessage) {
	var consensusHeightAttr string
	if len(consensusHeights) != 0 {
		consensusHeightAttr = consensusHeights[0].String()
	}

	consensusHeightsAttr := make([]string, len(consensusHeights))
	for i, height := range consensusHeights {
		consensusHeightsAttr[i] = height.String()
	}

	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeUpdateClient,
		event.Attribute{Key: types.AttributeKeyClientID, Value: clientID},
		event.Attribute{Key: types.AttributeKeyClientType, Value: clientType},
		// Deprecated: AttributeKeyConsensusHeight is deprecated and will be removed in a future release.
		// Please use AttributeKeyConsensusHeights instead.
		event.Attribute{Key: types.AttributeKeyConsensusHeight, Value: consensusHeightAttr},
		event.Attribute{Key: types.AttributeKeyConsensusHeights, Value: strings.Join(consensusHeightsAttr, ",")},
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.AttributeValueCategory},
	)
}

// emitUpgradeClientEvent emits an upgrade client event
func emitUpgradeClientEvent(ctx context.Context, env appmodule.Environment, clientID, clientType string, latestHeight exported.Height) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeUpgradeClient,
		event.Attribute{Key: types.AttributeKeyClientID, Value: clientID},
		event.Attribute{Key: types.AttributeKeyClientType, Value: clientType},
		event.Attribute{Key: types.AttributeKeyConsensusHeight, Value: latestHeight.String()},
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.AttributeValueCategory},
	)
}

// emitSubmitMisbehaviourEvent emits a client misbehaviour event
func emitSubmitMisbehaviourEvent(ctx context.Context, env appmodule.Environment, clientID string, clientType string) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeSubmitMisbehaviour,
		event.Attribute{Key: types.AttributeKeyClientID, Value: clientID},
		event.Attribute{Key: types.AttributeKeyClientType, Value: clientType},
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.AttributeValueCategory},
	)
}

// emitRecoverClientEvent emits a recover client event
func emitRecoverClientEvent(ctx context.Context, env appmodule.Environment, clientID, clientType string) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeRecoverClient,
		event.Attribute{Key: types.AttributeKeySubjectClientID, Value: clientID},
		event.Attribute{Key: types.AttributeKeyClientType, Value: clientType},
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.AttributeValueCategory},
	)
}

// emitScheduleIBCSoftwareUpgradeEvent emits a schedule IBC software upgrade event
func emitScheduleIBCSoftwareUpgradeEvent(ctx context.Context, env appmodule.Environment, title string, height int64) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeScheduleIBCSoftwareUpgrade,
		event.Attribute{Key: types.AttributeKeyUpgradePlanTitle, Value: title},
		event.Attribute{Key: types.AttributeKeyUpgradePlanHeight, Value: fmt.Sprintf("%d", height)},
	)
	env.EventService.EventManager(ctx).EmitKV(

		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.AttributeValueCategory},
	)
}

// EmitUpgradeChainEvent emits an upgrade chain event.
func EmitUpgradeChainEvent(ctx context.Context, env appmodule.Environment, height int64) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeUpgradeChain,
		event.Attribute{Key: types.AttributeKeyUpgradePlanHeight, Value: strconv.FormatInt(height, 10)},
		event.Attribute{Key: types.AttributeKeyUpgradeStore, Value: upgradetypes.StoreKey}, // which store to query proof of consensus state from
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.AttributeValueCategory},
	)
}
