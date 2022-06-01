package keeper

import (
	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
)

// GetClassTrace retreives the full identifiers trace and base classId from the store.
func (k Keeper) GetClassTrace(ctx sdk.Context, denomTraceHash tmbytes.HexBytes) (types.ClassTrace, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ClassTraceKey)
	bz := store.Get(denomTraceHash)
	if bz == nil {
		return types.ClassTrace{}, false
	}

	denomTrace := k.MustUnmarshalClassTrace(bz)
	return denomTrace, true
}

// GetAllClassTraces returns the trace information for all the class.
func (k Keeper) GetAllClassTraces(ctx sdk.Context) types.Traces {
	traces := types.Traces{}
	k.IterateClassTraces(ctx, func(classTrace types.ClassTrace) bool {
		traces = append(traces, classTrace)
		return false
	})

	return traces.Sort()
}

// IterateClassTraces iterates over the class traces in the store
// and performs a callback function.
func (k Keeper) IterateClassTraces(ctx sdk.Context, cb func(denomTrace types.ClassTrace) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.ClassTraceKey)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		classTrace := k.MustUnmarshalClassTrace(iterator.Value())
		if cb(classTrace) {
			break
		}
	}
}

// ClassPathFromHash returns the full class path prefix from an ibc classId with a hash
// component.
func (k Keeper) ClassPathFromHash(ctx sdk.Context, classID string) (string, error) {
	// trim the class prefix, by default "ibc/"
	hexHash := classID[len(types.ClassPrefix+"/"):]

	hash, err := types.ParseHexHash(hexHash)
	if err != nil {
		return "", sdkerrors.Wrap(types.ErrInvalidClassID, err.Error())
	}

	classTrace, found := k.GetClassTrace(ctx, hash)
	if !found {
		return "", sdkerrors.Wrap(types.ErrTraceNotFound, hexHash)
	}
	return classTrace.GetFullClassPath(), nil
}

// HasClassTrace checks if a the key with the given denomination trace hash exists on the store.
func (k Keeper) HasClassTrace(ctx sdk.Context, denomTraceHash tmbytes.HexBytes) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ClassTraceKey)
	return store.Has(denomTraceHash)
}

// SetClassTrace sets a new {trace hash -> class trace} pair to the store.
func (k Keeper) SetClassTrace(ctx sdk.Context, denomTrace types.ClassTrace) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ClassTraceKey)
	bz := k.MustMarshalClassTrace(denomTrace)
	store.Set(denomTrace.Hash(), bz)
}

// MustUnmarshalClassTrace attempts to decode and return an ClassTrace object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalClassTrace(bz []byte) types.ClassTrace {
	var classTrace types.ClassTrace
	k.cdc.MustUnmarshal(bz, &classTrace)
	return classTrace
}

// MustMarshalClassTrace attempts to decode and return an ClassTrace object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalClassTrace(classTrace types.ClassTrace) []byte {
	return k.cdc.MustMarshal(&classTrace)
}

// UnmarshalClassTrace attempts to decode and return an ClassTrace object from
// raw encoded bytes.
func (k Keeper) UnmarshalClassTrace(bz []byte) (types.ClassTrace, error) {
	var classTrace types.ClassTrace
	if err := k.cdc.Unmarshal(bz, &classTrace); err != nil {
		return types.ClassTrace{}, err
	}
	return classTrace, nil
}
