package simulation_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/simulation"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

func TestDecodeStore(t *testing.T) {
	channelID := "channel-0"
	denom := "atom"

	testCases := []struct {
		name        string
		kvA         kv.Pair
		kvB         kv.Pair
		expectedLog string
	}{
		{
			"Port",
			kv.Pair{
				Key:   types.KeyPort("port_id"),
				Value: []byte("port_id"),
			},
			kv.Pair{
				Key:   types.KeyPort("port_id"),
				Value: []byte("port_id"),
			},
			fmt.Sprintf("Port A: %s\nPort B: %s", "port_id", "port_id"),
		},
		{
			"Rate Limit",
			kv.Pair{
				Key:   types.KeyRateLimitItem(denom, channelID),
				Value: createFlowBytes(t),
			},
			kv.Pair{
				Key:   types.KeyRateLimitItem(denom, channelID),
				Value: createFlowBytes(t),
			},
			fmt.Sprintf("Flow A: %v\nFlow B: %v", createFlow(), createFlow()),
		},
		{
			"Unknown",
			kv.Pair{
				Key:   []byte{0x99},
				Value: []byte{0x99},
			},
			kv.Pair{
				Key:   []byte{0x99},
				Value: []byte{0x99},
			},
			fmt.Sprintf("invalid %s key prefix 99", types.ModuleName),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			decoder := simulation.NewDecodeStore()
			if tc.name == "Unknown" {
				require.Panics(t, func() {
					decoder(tc.kvA, tc.kvB)
				})
			} else {
				log := decoder(tc.kvA, tc.kvB)
				require.Equal(t, tc.expectedLog, log)
			}
		})
	}
}

func createFlow() *types.Flow {
	return &types.Flow{
		Inflow:       sdkmath.NewInt(100),
		Outflow:      sdkmath.NewInt(50),
		ChannelValue: sdkmath.NewInt(1000),
	}
}

func createFlowBytes(t *testing.T) []byte {
	flow := createFlow()
	bz, err := types.ModuleCdc.Marshal(flow)
	require.NoError(t, err)
	return bz
}
