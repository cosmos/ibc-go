package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines a subset of methods implemented by the cosmos-sdk account keeper
type AccountKeeper interface {
	// Return a new account with the next account number and the specified address. Does not save the new account to the store.
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	// Retrieve an account from the store.
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	// Set an account in the store.
	SetAccount(ctx context.Context, acc sdk.AccountI)
}
