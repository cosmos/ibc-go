package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// AccountAddress defines the handler for the Query/AccountAddress RPC method.
func (k Keeper) AccountAddress(ctx context.Context, req *types.QueryAccountAddressRequest) (*types.QueryAccountAddressResponse, error) {
	// TODO: Add logic
	panic("not implemented")
}
