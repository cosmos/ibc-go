package keeper

import (
	"context"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

// getOrCreateICS27Account retrieves an existing ICS27 account or creates a new one if it doesn't exist.
func (k Keeper) getOrCreateICS27Account(ctx context.Context, accountID *types.AccountIdentifier) (*types.ICS27Account, error) {
	existingIcs27Account, err := k.Accounts.Get(ctx, collections.Join3(accountID.ClientId, accountID.Sender, accountID.Salt))
	if err == nil {
		return &existingIcs27Account, nil
	} else if !errorsmod.IsOf(err, collections.ErrNotFound) {
		return nil, err
	}

	// Create a new account
	newAddr, err := types.BuildAddressPredictable(accountID)
	if err != nil {
		return nil, err
	}

	existingAcc := k.accountKeeper.GetAccount(ctx, newAddr)
	if existingAcc != nil {
		// TODO: ensure this cannot be abused
		return nil, errorsmod.Wrapf(types.ErrAccountAlreadyExists, "existing account for newly generated ICS27 account address %s", newAddr)
	}

	newAcc := k.accountKeeper.NewAccountWithAddress(ctx, newAddr)
	k.accountKeeper.SetAccount(ctx, newAcc)

	ics27Account := types.NewICS27Account(newAcc.GetAddress().String(), accountID)
	if err := k.Accounts.Set(ctx, collections.Join3(accountID.ClientId, accountID.Sender, accountID.Salt), ics27Account); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to set account %s in store", ics27Account)
	}

	k.Logger(ctx).Info("Created new ICS27 account", "account", ics27Account)
	return &ics27Account, nil
}

// getOrComputeICS27Adderss retrieves an existing ICS27 account address or computes it if it doesn't exist. This doesn't modify the store.
func (k Keeper) getOrComputeICS27Address(ctx context.Context, accountID *types.AccountIdentifier) (string, error) {
	existingIcs27Account, err := k.Accounts.Get(ctx, collections.Join3(accountID.ClientId, accountID.Sender, accountID.Salt))
	if err == nil {
		return existingIcs27Account.Address, nil
	} else if !errorsmod.IsOf(err, collections.ErrNotFound) {
		return "", err
	}

	// Compute a new address
	newAddr, err := types.BuildAddressPredictable(accountID)
	if err != nil {
		return "", err
	}

	return newAddr.String(), nil
}
