package keeper

import (
	"context"
	"strconv"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/event"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// EmitAcknowledgementEvent emits an event signalling a successful or failed acknowledgement and including the error
// details if any.
func EmitAcknowledgementEvent(ctx context.Context, env appmodule.Environment, packet channeltypes.Packet, ack exported.Acknowledgement, err error) {
	attributes := []event.Attribute{
		{Key: sdk.AttributeKeyModule, Value: icatypes.ModuleName},
		{Key: icatypes.AttributeKeyHostChannelID, Value: packet.GetDestChannel()},
		{Key: icatypes.AttributeKeyAckSuccess, Value: strconv.FormatBool(ack.Success())},
	}

	if err != nil {
		attributes = append(attributes, event.Attribute{Key: icatypes.AttributeKeyAckError, Value: err.Error()})
	}

	env.EventService.EventManager(ctx).EmitKV(
		icatypes.EventTypePacket,
		attributes...,
	)
}

// EmitHostDisabledEvent emits an event signalling that the host submodule is disabled.
func EmitHostDisabledEvent(ctx context.Context, env appmodule.Environment, packet channeltypes.Packet) {
	env.EventService.EventManager(ctx).EmitKV(
		icatypes.EventTypePacket,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: icatypes.ModuleName},
		event.Attribute{Key: icatypes.AttributeKeyHostChannelID, Value: packet.GetDestChannel()},
		event.Attribute{Key: icatypes.AttributeKeyAckError, Value: types.ErrHostSubModuleDisabled.Error()},
		event.Attribute{Key: icatypes.AttributeKeyAckSuccess, Value: "false"},
	)
}
