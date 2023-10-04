package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	Get(ctx sdk.Context, key []byte, ptr interface{})
	Set(ctx sdk.Context, key []byte, param interface{})
}
