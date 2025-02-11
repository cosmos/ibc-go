package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clientv1keeper "github.com/cosmos/ibc-go/v9/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/types"
)

type Keeper struct {
	cdc            codec.BinaryCodec
	ClientV1Keeper *clientv1keeper.Keeper
}

// NewKeeper creates a new client v2 keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	clientV1Keeper *clientv1keeper.Keeper,
) *Keeper {
	return &Keeper{
		cdc:            cdc,
		ClientV1Keeper: clientV1Keeper,
	}
}

// SetClientCounterparty sets counterpartyInfo for a given clientID
func (k *Keeper) SetClientCounterparty(ctx sdk.Context, clientID string, counterparty types.CounterpartyInfo) {
	store := k.ClientV1Keeper.ClientStore(ctx, clientID)
	store.Set(types.CounterpartyKey(), k.cdc.MustMarshal(&counterparty))
}

// GetClientCounterparty gets counterpartyInfo for a given clientID
func (k *Keeper) GetClientCounterparty(ctx sdk.Context, clientID string) (types.CounterpartyInfo, bool) {
	store := k.ClientV1Keeper.ClientStore(ctx, clientID)
	bz := store.Get(types.CounterpartyKey())
	if len(bz) == 0 {
		return types.CounterpartyInfo{}, false
	}

	var counterparty types.CounterpartyInfo
	k.cdc.MustUnmarshal(bz, &counterparty)
	return counterparty, true
}
