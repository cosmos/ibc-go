package keeper

import (
	"strings"

	"github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
)

// Keeper defines the IBC connection keeper
type Keeper struct {
	Router *types.Router
}

// NewKeeper creates a new IBC connection Keeper instance
func NewKeeper() *Keeper {
	return &Keeper{}
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
