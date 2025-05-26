package simulation

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// Simulation parameter constants
const (
	enabledKey           = "enabled"
	defaultMaxOutflowKey = "default_max_outflow"
	defaultMaxInflowKey  = "default_max_inflow"
	defaultPeriodKey     = "default_period"
)

// RandomEnabled randomized enabled param with 75% prob of being true.
func RandomEnabled(r *rand.Rand) bool {
	return r.Int63n(101) <= 75
}

// RandomMaxValue returns a random max value between min and max
func RandomMaxValue(r *rand.Rand, mn, mx int64) string {
	return sdkmath.NewInt(r.Int63n(mx-mn) + mn).String()
}

// RandomPeriod returns a random period in seconds (between 1 hour and 1 week)
func RandomPeriod(r *rand.Rand) uint64 {
	// Random period between 1 hour (3600 seconds) and 1 week (604800 seconds)
	return uint64(r.Int63n(604800-3600) + 3600)
}

// RandomizedGenState generates a random GenesisState for rate-limiting
func RandomizedGenState(simState *module.SimulationState) {
	var enabled bool
	simState.AppParams.GetOrGenerate(
		enabledKey, &enabled, simState.Rand,
		func(r *rand.Rand) { enabled = RandomEnabled(r) },
	)

	var defaultMaxOutflow string
	simState.AppParams.GetOrGenerate(
		defaultMaxOutflowKey, &defaultMaxOutflow, simState.Rand,
		func(r *rand.Rand) { defaultMaxOutflow = RandomMaxValue(r, 100000, 10000000) },
	)

	var defaultMaxInflow string
	simState.AppParams.GetOrGenerate(
		defaultMaxInflowKey, &defaultMaxInflow, simState.Rand,
		func(r *rand.Rand) { defaultMaxInflow = RandomMaxValue(r, 100000, 10000000) },
	)

	var defaultPeriod uint64
	simState.AppParams.GetOrGenerate(
		defaultPeriodKey, &defaultPeriod, simState.Rand,
		func(r *rand.Rand) { defaultPeriod = RandomPeriod(r) },
	)

	// We use the randomly generated values only for logging, as the core params are not directly in GenesisState
	rateLimitingGenesis := types.GenesisState{
		RateLimits:                       []types.RateLimit{},
		WhitelistedAddressPairs:          []types.WhitelistedAddressPair{},
		BlacklistedDenoms:                []string{},
		PendingSendPacketSequenceNumbers: []string{},
		HourEpoch: types.HourEpoch{
			EpochNumber:      0,
			Duration:         time.Hour,
			EpochStartTime:   time.Time{},
			EpochStartHeight: 0,
		},
	}

	params := types.Params{
		Enabled:           enabled,
		DefaultMaxOutflow: defaultMaxOutflow,
		DefaultMaxInflow:  defaultMaxInflow,
		DefaultPeriod:     defaultPeriod,
	}

	bz, err := json.MarshalIndent(&params, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated %s parameters:\n%s\n", types.ModuleName, bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&rateLimitingGenesis)
}
