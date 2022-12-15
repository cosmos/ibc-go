package simulation

import (
	"fmt"
	"math/rand"

	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	gogotypes "github.com/gogo/protobuf/types"

	controllerkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/keeper"
	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	hostkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/keeper"
	hosttypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
)

// ParamChanges defines the parameters that can be modified by param change proposals
// on the simulation
func ParamChanges(r *rand.Rand, controllerKeeper *controllerkeeper.Keeper, hostKeeper *hostkeeper.Keeper) []simtypes.ParamChange {
	var paramChanges []simtypes.ParamChange
	if controllerKeeper != nil {
		paramChanges = append(paramChanges, simulation.NewSimParamChange(controllertypes.SubModuleName, string(controllertypes.KeyControllerEnabled),
			func(r *rand.Rand) string {
				controllerEnabled := RandomEnabled(r)
				return fmt.Sprintf("%s", types.ModuleCdc.MustMarshalJSON(&gogotypes.BoolValue{Value: controllerEnabled})) //nolint:gosimple
			},
		))
	}

	if hostKeeper != nil {
		paramChanges = append(paramChanges, simulation.NewSimParamChange(hosttypes.SubModuleName, string(hosttypes.KeyHostEnabled),
			func(r *rand.Rand) string {
				receiveEnabled := RandomEnabled(r)
				return fmt.Sprintf("%s", types.ModuleCdc.MustMarshalJSON(&gogotypes.BoolValue{Value: receiveEnabled})) //nolint:gosimple
			},
		))
	}

	return paramChanges
}
