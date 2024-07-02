package v9

import (
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ClientKeeper interface {
	ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore
}
