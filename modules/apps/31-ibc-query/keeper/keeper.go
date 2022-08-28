package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
	host "github.com/cosmos/ibc-go/v4/modules/core/24-host"
	"github.com/tendermint/tendermint/libs/log"
)

// Keeper define 31-ibc-query keeper
type Keeper struct {
	storeKey       sdk.StoreKey
	cdc            codec.BinaryCodec
	scopedKeeper   capabilitykeeper.ScopedKeeper
}

// NewKeeper creates a new 31-ibc-query Keeper instance
func NewKeeper(cdc codec.BinaryCodec, key sdk.StoreKey,) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: key,

	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+host.ModuleName+"-"+types.ModuleName)
}

func (k Keeper) GenerateQueryIdentifier(ctx sdk.Context) string {
	nextQuerySeq := k.GetNextQuerySequence(ctx)
	queryID := types.FormatQueryIdentifier(nextQuerySeq)

	nextQuerySeq++
	k.SetNextQuerySequence(ctx, nextQuerySeq)
	return queryID
}

func (k Keeper) GetNextQuerySequence(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.KeyNextQuerySequence))
	if bz == nil {
		panic("next connection sequence is nil")
	}

	return sdk.BigEndianToUint64(bz)
}

func (k Keeper) SetNextQuerySequence(ctx sdk.Context, sequence uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := sdk.Uint64ToBigEndian(sequence)
	store.Set([]byte(types.KeyNextQuerySequence), bz)
}

// SetSubmitCrossChainQuery stores the MsgSubmitCrossChainQuery in state keyed by the query id
func (k Keeper) SetSubmitCrossChainQuery(ctx sdk.Context, query types.MsgSubmitCrossChainQuery) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryKey)
    bz := k.MustMarshalQuery(&query)
	store.Set(host.QueryKey(query.Id), bz)
}

// GetSubmitCrossChainQuery retrieve the MsgSubmitCrossChainQuery stored in state given the query id
func (k Keeper) GetSubmitCrossChainQuery(ctx sdk.Context, queryId string) (types.MsgSubmitCrossChainQuery, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryKey)
	key := host.QueryKey(queryId)
	bz := store.Get(key)
	if bz == nil {
		return types.MsgSubmitCrossChainQuery{}, false
	}

	return k.MustUnmarshalQuery(bz), true
}

// GetAllSubmitCrossChainQueries returns a list of all MsgSubmitCrossChainQueries that are stored in state
func (k Keeper) GetAllSubmitCrossChainQueries(ctx sdk.Context) []*types.MsgSubmitCrossChainQuery {
	var crossChainQueries []*types.MsgSubmitCrossChainQuery
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

// DeleteCrossChainQuery deletes MsgSubmitCrossChainQuery associated with the query id
func (k Keeper) DeleteSubmitCrossChainQuery(ctx sdk.Context, queryId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryKey)
	store.Delete(host.QueryKey(queryId))
}




// TODO
// func handleIbcQueryResult
// 1. relayer should be call this func with query result
// 2. save query in private store


// SetCrossChainQueryResult stores the CrossChainQueryResult in state keyed by the query id
func (k Keeper) SetSubmitCrossChainQueryResult(ctx sdk.Context, result types.MsgSubmitCrossChainQueryResult) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryResultKey)
	bz := k.MustMarshalQueryResult(&result)
	store.Set(host.QueryResultKey(result.Id), bz)
}

// GetCrossChainQueryResult retrieve the CrossChainQueryResult stored in state given the query id
func (k Keeper) GetCrossChainQueryResult(ctx sdk.Context, queryId string) (types.MsgSubmitCrossChainQueryResult, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryResultKey)
	key := host.QueryResultKey(queryId)
	bz := store.Get(key)
	if bz == nil {
		return types.MsgSubmitCrossChainQueryResult{}, false
	}

	return k.MustUnmarshalQueryResult(bz), true
}

// GetAllCrossChainQueryResults returns a list of all CrossChainQueryResults that are stored in state
func (k Keeper) GetAllSubmitCrossChainQueryResults(ctx sdk.Context) []*types.MsgSubmitCrossChainQueryResult {
	var crossChainQueryResults []*types.MsgSubmitCrossChainQueryResult
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

// DeleteCrossChainQueryResult deletes CrossChainQueryResult associated with the query id
func (k Keeper) DeleteSubmitCrossChainQueryResult(ctx sdk.Context, queryId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryResultKey)
	store.Delete(host.QueryResultKey(queryId))
}




// MustMarshalQuery attempts to encode a CrossChainQuery object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalQuery(query *types.MsgSubmitCrossChainQuery) []byte {
	return k.cdc.MustMarshal(query)
}

// MustUnmarshalQuery attempts to decode and return a CrossChainQuery object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalQuery(bz []byte) types.MsgSubmitCrossChainQuery {
	var query types.MsgSubmitCrossChainQuery
	k.cdc.MustUnmarshal(bz, &query)
	return query
}

// MustMarshalQuery attempts to encode a CrossChainQuery object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalQueryResult(result *types.MsgSubmitCrossChainQueryResult) []byte {
	return k.cdc.MustMarshal(result)
}

// MustUnmarshalQuery attempts to decode and return a CrossChainQuery object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalQueryResult(bz []byte) types.MsgSubmitCrossChainQueryResult {
	var result types.MsgSubmitCrossChainQueryResult
	k.cdc.MustUnmarshal(bz, &result)
	return result
}



