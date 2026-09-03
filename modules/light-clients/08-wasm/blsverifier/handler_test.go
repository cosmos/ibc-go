package blsverifier_test

import (
	"encoding/json"
	"testing"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v11/blsverifier"
)

func TestCustomQuerier_GasConsumptionAndBounds(t *testing.T) {
	gasMeter := storetypes.NewGasMeter(1_000_000)
	ctx := sdk.Context{}.WithGasMeter(gasMeter)

	querier := blsverifier.CustomQuerier()

	t.Run("aggregate query consumes proportional gas", func(t *testing.T) {
		publicKeys := make([][]byte, 5)
		for i := 0; i < 5; i++ {
			publicKeys[i] = []byte("mock_public_key_bytes_len_48____________________")
		}

		reqPayload, err := json.Marshal(map[string]interface{}{
			"aggregate": map[string]interface{}{
				"public_keys": publicKeys,
			},
		})
		require.NoError(t, err)

		gasBefore := ctx.GasMeter().GasConsumed()
		// Note: execution may return an error on mock key decoding, but gas must still be consumed before decoding
		_, _ = querier(ctx, reqPayload)
		gasAfter := ctx.GasMeter().GasConsumed()

		expectedGas := uint64(len(publicKeys)) * blsverifier.GasCostPerBLSAggregateKey
		require.Equal(t, gasBefore+expectedGas, gasAfter)
	})

	t.Run("exceeding max public keys returns error without panic", func(t *testing.T) {
		publicKeys := make([][]byte, blsverifier.MaxBLSPublicKeys+1)
		reqPayload, err := json.Marshal(map[string]interface{}{
			"aggregate": map[string]interface{}{
				"public_keys": publicKeys,
			},
		})
		require.NoError(t, err)

		_, err = querier(ctx, reqPayload)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds maximum allowed")
	})
}
