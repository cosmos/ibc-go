package simulation_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/simulation"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// TestRandomizedGenState tests the normal scenario of applying RandomizedGenState.
// Abnormal scenarios are not tested here.
func TestRandomizedGenState(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	s := rand.NewSource(1)
	r := rand.New(s)

	simState := &module.SimulationState{
		AppParams:    make(simtypes.AppParams),
		Cdc:          cdc,
		Rand:         r,
		NumBonded:    3,
		Accounts:     simtypes.RandomAccounts(r, 3),
		InitialStake: sdkmath.NewInt(1000),
		GenState:     make(map[string]json.RawMessage),
	}

	simulation.RandomizedGenState(simState)

	var rateLimitingGenesis types.GenesisState
	simState.Cdc.MustUnmarshalJSON(simState.GenState[types.ModuleName], &rateLimitingGenesis)

	require.NotNil(t, rateLimitingGenesis.HourEpoch)
	require.Equal(t, uint64(0), rateLimitingGenesis.HourEpoch.EpochNumber)
	require.NotEmpty(t, rateLimitingGenesis.HourEpoch.Duration)
	require.Len(t, rateLimitingGenesis.RateLimits, 0)
	require.Len(t, rateLimitingGenesis.WhitelistedAddressPairs, 0)
	require.Len(t, rateLimitingGenesis.BlacklistedDenoms, 0)
	require.Len(t, rateLimitingGenesis.PendingSendPacketSequenceNumbers, 0)
}
