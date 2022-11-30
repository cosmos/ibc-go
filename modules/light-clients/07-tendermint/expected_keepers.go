package tendermint

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// ClientKeeper expected account IBC client keeper
type ClientKeeper interface {
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	IterateClientStates(ctx sdk.Context, prefix []byte, cb func(string, exported.ClientState) bool)
	ClientStore(ctx sdk.Context, clientID string) sdk.KVStore
}
