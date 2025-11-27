package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
)

// TODO: Write unit tests #8321

// InitGenesis
func (k *Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	// Initialize store refund path for forwarded packets in genesis state that have not yet been acked.
	store := k.storeService.OpenKVStore(ctx)
	for key, value := range state.InFlightPackets {
		bz := k.cdc.MustMarshal(&value)
		if err := store.Set([]byte(key), bz); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis
func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	store := k.storeService.OpenKVStore(ctx)

	inFlightPackets := make(map[string]types.InFlightPacket)

	itr, err := store.Iterator(nil, nil)
	if err != nil {
		panic(err)
	}
	for ; itr.Valid(); itr.Next() {
		var inFlightPacket types.InFlightPacket
		k.cdc.MustUnmarshal(itr.Value(), &inFlightPacket)
		inFlightPackets[string(itr.Key())] = inFlightPacket
	}
	return &types.GenesisState{InFlightPackets: inFlightPackets}
}
