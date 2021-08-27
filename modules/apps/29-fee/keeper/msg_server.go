package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

var _ types.MsgServer = Keeper{}

// EscrowPacketFee
func (k Keeper) EscrowPacketFee(goCtx context.Context, msg *types.MsgEscrowPacketFee) (*types.MsgEscrowPacketFeeResponse, error)

// RegisterCounterPartyAddress
func (k Keeper) RegisterCounterPartyAddress(goCtx context.Context, msg *types.MsgRegisterCounterpartyAddress) (*types.MsgRegisterCounterPartyAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	a, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	c, err := sdk.AccAddressFromBech32(msg.CounterpartyAddress)
	if err != nil {
		return nil, err
	}

	if err := k.SendRegisterCounterPartyAddress(
		ctx, a, c,
	); err != nil {
		return nil, err
	}

	k.Logger(ctx).Info("Relayer register counterparty address", "relayer", msg.Address, "counterparty", msg.CounterpartyAddress)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRegisterCounterpartyAddress,
			// TODO: do we need these?
			// sdk.NewAttribute(sdk.AttributeKeySender, msg.Address),
			// sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})

	return &types.MsgRegisterCounterPartyAddressResponse{}, nil
}
