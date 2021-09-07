package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

func TestKeyRelayerAddress(t *testing.T) {
	var (
		relayerAddress = "relayer_address"
	)

	key := types.KeyRelayerAddress(relayerAddress)
	require.Equal(t, string(key), "relayerAddress/relayer_address")
}
