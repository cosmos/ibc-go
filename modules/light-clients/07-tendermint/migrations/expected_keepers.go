package migrations

import (
	"context"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// ClientKeeper expected account IBC client keeper
type ClientKeeper interface {
	GetClientState(ctx context.Context, clientID string) (exported.ClientState, bool)
	IterateClientStates(ctx context.Context, prefix []byte, cb func(string, exported.ClientState) bool)
	ClientStore(ctx context.Context, clientID string) storetypes.KVStore
}
