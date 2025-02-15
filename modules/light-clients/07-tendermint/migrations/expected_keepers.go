package migrations

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// ClientKeeper expected account IBC client keeper
type ClientKeeper interface {
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	IterateClientStates(ctx sdk.Context, prefix []byte, cb func(string, exported.ClientState) bool)
	ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore
	Logger(ctx sdk.Context) log.Logger
}
