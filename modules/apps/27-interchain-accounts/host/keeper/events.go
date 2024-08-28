package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// EmitAcknowledgementEvent emits an event signalling a successful or failed acknowledgement and including the error
// details if any.
func EmitAcknowledgementEvent(ctx context.Context, packet channeltypes.Packet, ack exported.Acknowledgement, err error) {
	attributes := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, icatypes.ModuleName),
		sdk.NewAttribute(icatypes.AttributeKeyHostChannelID, packet.GetDestChannel()),
		sdk.NewAttribute(icatypes.AttributeKeyAckSuccess, strconv.FormatBool(ack.Success())),
	}

	if err != nil {
		attributes = append(attributes, sdk.NewAttribute(icatypes.AttributeKeyAckError, err.Error()))
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: https://github.com/cosmos/ibc-go/issues/5917
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			icatypes.EventTypePacket,
			attributes...,
		),
	)
}

// EmitHostDisabledEvent emits an event signalling that the host submodule is disabled.
func EmitHostDisabledEvent(ctx context.Context, packet channeltypes.Packet) {
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: https://github.com/cosmos/ibc-go/issues/5917
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			icatypes.EventTypePacket,
			sdk.NewAttribute(sdk.AttributeKeyModule, icatypes.ModuleName),
			sdk.NewAttribute(icatypes.AttributeKeyHostChannelID, packet.GetDestChannel()),
			sdk.NewAttribute(icatypes.AttributeKeyAckError, types.ErrHostSubModuleDisabled.Error()),
			sdk.NewAttribute(icatypes.AttributeKeyAckSuccess, "false"),
		),
	)
}
