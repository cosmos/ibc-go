package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
	host "github.com/cosmos/ibc-go/v4/modules/core/24-host"
	"github.com/tendermint/tendermint/libs/log"
)

// Keeper define 31-ibc-query keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec
}

// NewKeeper creates a new 31-ibc-query Keeper instance
func NewKeeper(cdc codec.BinaryCodec, key sdk.StoreKey) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: key,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+host.ModuleName+"-"+types.ModuleName)
}

// TODO
// func handleIbcQuery
// 1. set unique query Id
// 2. set query in private store

// TODO
// func handleIbcQueryResult
// 1. relayer should be call this func with query result
// 2. save query in private store

// GetAllCrossChainQueries returns a list of all CrossChainQueries that are stored in state
func (k Keeper) GetAllCrossChainQueries(ctx sdk.Context) []*types.CrossChainQuery {
	var crossChainQueries []*types.CrossChainQuery
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.QueryKey)

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// unmarshal
		query := k.MustUnmarshalQuery(iterator.Value())
		crossChainQueries = append(crossChainQueries, &query)
	}
	return crossChainQueries
}

// GetCrossChainQuery retrieve the CrossChainQuery stored in state given the query id
func (k Keeper) GetCrossChainQuery(ctx sdk.Context, queryId string) (types.CrossChainQuery, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryKey)
	key := []byte(queryId)
	bz := store.Get(key)
	if bz == nil {
		return types.CrossChainQuery{}, false
	}

	return k.MustUnmarshalQuery(bz), true
}

// SetCrossChainQuery stores the CrossChainQuery in state keyed by the query id
func (k Keeper) SetCrossChainQuery(ctx sdk.Context, query *types.CrossChainQuery) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryKey)
	bz := k.MustMarshalQuery(query)
	store.Set([]byte(query.Id), bz)
}

// DeleteCrossChainQuery deletes CrossChainQuery associated with the query id
func (k Keeper) DeleteCrossChainQuery(ctx sdk.Context, queryId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryKey)
	store.Delete([]byte(queryId))
}

// GetAllCrossChainQueryResults returns a list of all CrossChainQueryResults that are stored in state
func (k Keeper) GetAllCrossChainQueryResults(ctx sdk.Context) []*types.CrossChainQueryResult {
	var crossChainQueryResults []*types.CrossChainQueryResult
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.QueryResultKey)

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// unmarshal
		result := k.MustUnmarshalQueryResult(iterator.Value())
		crossChainQueryResults = append(crossChainQueryResults, &result)
	}
	return crossChainQueryResults
}

// GetCrossChainQueryResult retrieve the CrossChainQueryResult stored in state given the query id
func (k Keeper) GetCrossChainQueryResult(ctx sdk.Context, queryId string) (types.CrossChainQueryResult, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryResultKey)
	key := []byte(queryId)
	bz := store.Get(key)
	if bz == nil {
		return types.CrossChainQueryResult{}, false
	}

	return k.MustUnmarshalQueryResult(bz), true
}

// SetCrossChainQueryResult stores the CrossChainQueryResult in state keyed by the query id
func (k Keeper) SetCrossChainQueryResult(ctx sdk.Context, result *types.CrossChainQueryResult) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryResultKey)
	bz := k.MustMarshalQueryResult(result)
	store.Set([]byte(result.Id), bz)
}

// DeleteCrossChainQueryResult deletes CrossChainQueryResult associated with the query id
func (k Keeper) DeleteCrossChainQueryResult(ctx sdk.Context, queryId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryResultKey)
	store.Delete([]byte(queryId))
}

// MustMarshalQuery attempts to encode a CrossChainQuery object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalQuery(query *types.CrossChainQuery) []byte {
	return k.cdc.MustMarshal(query)
}

// MustUnmarshalQuery attempts to decode and return a CrossChainQuery object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalQuery(bz []byte) types.CrossChainQuery {
	var query types.CrossChainQuery
	k.cdc.MustUnmarshal(bz, &query)
	return query
}

// MustMarshalQuery attempts to encode a CrossChainQuery object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalQueryResult(result *types.CrossChainQueryResult) []byte {
	return k.cdc.MustMarshal(result)
}

// MustUnmarshalQuery attempts to decode and return a CrossChainQuery object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalQueryResult(bz []byte) types.CrossChainQueryResult {
	var result types.CrossChainQueryResult
	k.cdc.MustUnmarshal(bz, &result)
	return result
}
