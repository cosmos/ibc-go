package types

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
	clienttypesv2 "github.com/cosmos/ibc-go/v11/modules/core/02-client/v2/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
)

// TokenFactoryKeeper defines the expected interface for the token factory module
type TokenFactoryKeeper interface {
	// MintTo mints new tokens of `denom` into `address`.
	// MUST fail if `denom` is not recognized or not authorized for minting.
	MintTo(ctx context.Context, denom string, amount math.Int, to sdk.AccAddress) error
	// BurnFrom burns tokens of `denom` from `address`.
	// MUST fail if `address` does not have enough balance or burn is not permitted.
	BurnFrom(ctx context.Context, denom string, amount math.Int, from sdk.AccAddress) error
	// HasDenom checks if a denom exists in the token factory
	HasDenom(ctx context.Context, denom string) bool
}

// GMPKeeper defines the expected interface for the GMP module
type GMPKeeper interface {
	// GetAccount retrieves the ICS27 account for a given address (reverse lookup)
	GetAccount(ctx context.Context, address sdk.AccAddress) (*gmptypes.ICS27Account, error)
}

// AccountKeeper defines the expected interface for the account keeper
type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// MessageRouter defines the expected message router interface
type MessageRouter interface {
	Handler(msg sdk.Msg) func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error)
}

// IBCClientKeeper defines the expected interface for the IBC client keeper
type IBCClientKeeper interface {
	GetClientState(ctx sdk.Context, clientID string) (ibcexported.ClientState, bool)
}

// IBCClientV2Keeper defines the expected interface for the IBCv2 client keeper
type IBCClientV2Keeper interface {
	GetClientCounterparty(ctx sdk.Context, clientID string) (clienttypesv2.CounterpartyInfo, bool)
}
