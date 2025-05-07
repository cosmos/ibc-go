package keeper

import (
	"context"
	"encoding/hex"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// AccountAddress defines the handler for the Query/AccountAddress RPC method.
func (k Keeper) AccountAddress(ctx context.Context, req *types.QueryAccountAddressRequest) (*types.QueryAccountAddressResponse, error) {
	salt, err := hex.DecodeString(req.Salt)
	if err != nil {
		return nil, err
	}

	accountID := types.NewAccountIdentifier(req.ClientId, req.Sender, salt)
	address, err := k.getOrComputeICS27Address(ctx, &accountID)
	if err != nil {
		return nil, err
	}

	return &types.QueryAccountAddressResponse{
		AccountAddress: address,
	}, nil
}
