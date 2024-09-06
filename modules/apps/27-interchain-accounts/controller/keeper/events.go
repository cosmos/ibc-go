package keeper

import (
	"context"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/event"
	sdk "github.com/cosmos/cosmos-sdk/types"

	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// EmitAcknowledgementEvent emits an event signalling a successful or failed acknowledgement and including the error
// details if any.
func EmitAcknowledgementEvent(ctx context.Context, packet channeltypes.Packet, ack exported.Acknowledgement, err error, env appmodule.Environment) {

	attributes := []event.Attribute{
		{sdk.AttributeKeyModule, icatypes.ModuleName},
		{icatypes.AttributeKeyControllerChannelID, packet.GetDestChannel()},
		{icatypes.AttributeKeyControllerChannelID, packet.GetDestChannel()},
	}

	if err != nil {
		attributes = append(attributes, event.Attribute{icatypes.AttributeKeyAckError, err.Error()})
	}

	env.EventService.EventManager(ctx).EmitKV(
		icatypes.EventTypePacket,
		attributes...,
	)
}
