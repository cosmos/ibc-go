package keeper

import (
	"context"
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

var _ types.QueryServer = (*Keeper)(nil)

// AccountAddress defines the handler for the Query/AccountAddress RPC method.
func (k *Keeper) AccountAddress(ctx context.Context, req *types.QueryAccountAddressRequest) (*types.QueryAccountAddressResponse, error) {
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

// AccountIdentifier defines the handler for the Query/AccountIdentifier RPC method.
func (k *Keeper) AccountIdentifier(ctx context.Context, req *types.QueryAccountIdentifierRequest) (*types.QueryAccountIdentifierResponse, error) {
	addr, err := sdk.AccAddressFromBech32(req.AccountAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	ics27Acc, err := k.AccountsByAddress.Get(ctx, addr)
	if err != nil {
		return nil, err
	}

	return &types.QueryAccountIdentifierResponse{
		AccountId: ics27Acc.AccountId,
	}, nil
}
