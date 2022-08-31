package simulation

import (
	"fmt"
	"math/rand"

	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	gogotypes "github.com/gogo/protobuf/types"

	controllertypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
)

// ParamChanges defines the parameters that can be modified by param change proposals
// on the simulation
func ParamChanges(r *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(controllertypes.SubModuleName, string(controllertypes.KeyControllerEnabled),
			func(r *rand.Rand) string {
				controllerEnabled := RadomEnabled(r)
				return fmt.Sprintf("%s", types.ModuleCdc.MustMarshalJSON(&gogotypes.BoolValue{Value: controllerEnabled})) //nolint:gosimple
			},
		),
		simulation.NewSimParamChange(hosttypes.SubModuleName, string(hosttypes.KeyHostEnabled),
			func(r *rand.Rand) string {
				receiveEnabled := RadomEnabled(r)
				return fmt.Sprintf("%s", types.ModuleCdc.MustMarshalJSON(&gogotypes.BoolValue{Value: receiveEnabled})) //nolint:gosimple
			},
		),
	}
}
