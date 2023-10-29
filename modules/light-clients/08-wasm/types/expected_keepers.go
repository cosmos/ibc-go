package types

import (
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ClientKeeper defines the expected client keeper
type ClientKeeper interface {
	ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore
}
