package keeper

import (
	"context"
	"strconv"

	"cosmossdk.io/core/event"

	sdk "github.com/cosmos/cosmos-sdk/types"

	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// EmitAcknowledgementEvent emits an event signalling a successful or failed acknowledgement and including the error
// details if any.
func (k *Keeper) EmitAcknowledgementEvent(ctx context.Context, packet channeltypes.Packet, ack exported.Acknowledgement, err error) error {
	attributes := []event.Attribute{
		event.NewAttribute(sdk.AttributeKeyModule, icatypes.ModuleName),
		event.NewAttribute(icatypes.AttributeKeyControllerChannelID, packet.GetDestChannel()),
		event.NewAttribute(icatypes.AttributeKeyAckSuccess, strconv.FormatBool(ack.Success())),
	}

	if err != nil {
		attributes = append(attributes, event.NewAttribute(icatypes.AttributeKeyAckError, err.Error()))
	}

	return k.EventService.EventManager(ctx).EmitKV(icatypes.EventTypePacket, attributes...)
}
