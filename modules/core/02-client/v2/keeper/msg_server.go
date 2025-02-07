package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/types"
)

var _ types.MsgServer = &Keeper{}

func (k Keeper) RegisterCounterparty(ctx context.Context, counterparty *types.MsgRegisterCounterparty) (*types.MsgRegisterCounterpartyResponse, error) {
	//TODO implement me
	panic("implement me")
}
