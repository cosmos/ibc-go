package keeper

import (
	"strings"

	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// Keeper defines the IBC connection keeper
type Keeper struct {
	Router   *types.Router
	RouterV2 *api.Router
}

// NewKeeper creates a new IBC connection Keeper instance
func NewKeeper() *Keeper {
	return &Keeper{}
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"/"+types.SubModuleName)
}

// Route returns a IBCModule for a given module, and a boolean indicating
// whether or not the route is present.
func (k *Keeper) Route(module string) (types.IBCModule, bool) {
	routes, ok := k.Router.Route(module)

	if ok {
		return routes, true
	}

	for _, prefix := range k.Router.Keys() {
		if strings.Contains(module, prefix) {
			return k.Router.Route(prefix)
		}
	}

	return nil, false
}
