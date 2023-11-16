package types

import (
	storetypes "github.com/cosmos/cosmos-sdk/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// ClientKeeper defines the expected client keeper
type ClientKeeper interface {
	ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	SetClientState(ctx sdk.Context, clientID string, clientState exported.ClientState)
}
