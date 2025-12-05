package keeper

import (
	"context"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

// InitGenesis initializes the module state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	for _, account := range data.Ics27Accounts {
		if _, err := sdk.AccAddressFromBech32(account.AccountAddress); err != nil {
			return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
		if _, err := sdk.AccAddressFromBech32(account.AccountId.Sender); err != nil {
			return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}

		if err := k.Accounts.Set(ctx, collections.Join3(account.AccountId.ClientId, account.AccountId.Sender, account.AccountId.Salt), types.ICS27Account{
			Address:   account.AccountAddress,
			AccountId: &account.AccountId,
		}); err != nil {
			return err
		}
	}

	return nil
}

// ExportGenesis exports the module state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var accounts []types.RegisteredICS27Account
	if err := k.Accounts.Walk(ctx, nil, func(key collections.Triple[string, string, []byte], value types.ICS27Account) (bool, error) {
		accounts = append(accounts, types.RegisteredICS27Account{
			AccountAddress: value.Address,
			AccountId:      *value.AccountId,
		})

		return false, nil
	}); err != nil {
		return nil, err
	}

	return &types.GenesisState{
		Ics27Accounts: accounts,
	}, nil
}
