package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/core/event"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// emitCreateClientEvent emits a create client event
func (k *Keeper) emitCreateClientEvent(ctx context.Context, clientID, clientType string, initialHeight exported.Height) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeCreateClient,
		event.NewAttribute(types.AttributeKeyClientID, clientID),
		event.NewAttribute(types.AttributeKeyClientType, clientType),
		event.NewAttribute(types.AttributeKeyConsensusHeight, initialHeight.String()),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitUpdateClientEvent emits an update client event
func (k *Keeper) emitUpdateClientEvent(ctx context.Context, clientID string, clientType string, consensusHeights []exported.Height, _ codec.BinaryCodec, _ exported.ClientMessage) error {
	var consensusHeightAttr string
	if len(consensusHeights) != 0 {
		consensusHeightAttr = consensusHeights[0].String()
	}

	consensusHeightsAttr := make([]string, len(consensusHeights))
	for i, height := range consensusHeights {
		consensusHeightsAttr[i] = height.String()
	}

	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeUpdateClient,
		event.NewAttribute(types.AttributeKeyClientID, clientID),
		event.NewAttribute(types.AttributeKeyClientType, clientType),
		// Deprecated: AttributeKeyConsensusHeight is deprecated and will be removed in a future release.
		// Please use AttributeKeyConsensusHeights instead.
		event.NewAttribute(types.AttributeKeyConsensusHeight, consensusHeightAttr),
		event.NewAttribute(types.AttributeKeyConsensusHeights, strings.Join(consensusHeightsAttr, ",")),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitUpgradeClientEvent emits an upgrade client event
func (k *Keeper) emitUpgradeClientEvent(ctx context.Context, clientID, clientType string, latestHeight exported.Height) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeUpgradeClient,
		event.NewAttribute(types.AttributeKeyClientID, clientID),
		event.NewAttribute(types.AttributeKeyClientType, clientType),
		event.NewAttribute(types.AttributeKeyConsensusHeight, latestHeight.String()),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitSubmitMisbehaviourEvent emits a client misbehaviour event
func (k *Keeper) emitSubmitMisbehaviourEvent(ctx context.Context, clientID string, clientType string) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeSubmitMisbehaviour,
		event.NewAttribute(types.AttributeKeyClientID, clientID),
		event.NewAttribute(types.AttributeKeyClientType, clientType),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitRecoverClientEvent emits a recover client event
func (k *Keeper) emitRecoverClientEvent(ctx context.Context, clientID, clientType string) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeRecoverClient,
		event.NewAttribute(types.AttributeKeySubjectClientID, clientID),
		event.NewAttribute(types.AttributeKeyClientType, clientType),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitScheduleIBCSoftwareUpgradeEvent emits a schedule IBC software upgrade event
func (k *Keeper) emitScheduleIBCSoftwareUpgradeEvent(ctx context.Context, title string, height int64) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeScheduleIBCSoftwareUpgrade,
		event.NewAttribute(types.AttributeKeyUpgradePlanTitle, title),
		event.NewAttribute(types.AttributeKeyUpgradePlanHeight, fmt.Sprintf("%d", height)),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitUpgradeChainEvent emits an upgrade chain event.
func (k *Keeper) EmitUpgradeChainEvent(ctx context.Context, height int64) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeUpgradeChain,
		event.NewAttribute(types.AttributeKeyUpgradePlanHeight, strconv.FormatInt(height, 10)),
		event.NewAttribute(types.AttributeKeyUpgradeStore, upgradetypes.StoreKey), // which store to query proof of consensus state from
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}
