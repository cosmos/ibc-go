package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clientv1keeper "github.com/cosmos/ibc-go/v10/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
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

// GetConfig returns the ibc-client v2 configuration for the given clientID.
func (k *Keeper) GetConfig(ctx sdk.Context, clientID string) types.Config {
	store := k.ClientV1Keeper.ClientStore(ctx, clientID)
	bz := store.Get(types.ConfigKey())
	if len(bz) == 0 {
		return types.NewConfig()
	}

	var config types.Config
	k.cdc.MustUnmarshal(bz, &config)
	return config
}

// SetConfig sets ibc-client v2 configuration for the given clientID.
func (k *Keeper) SetConfig(ctx sdk.Context, clientID string, config types.Config) {
	store := k.ClientV1Keeper.ClientStore(ctx, clientID)
	bz := k.cdc.MustMarshal(&config)
	store.Set(types.ConfigKey(), bz)
}
