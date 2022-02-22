package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
)

func TestKeyCounterpartyRelayer(t *testing.T) {
	var (
		relayerAddress = "relayer_address"
		channelID      = "channel-0"
	)

	key := types.KeyCounterpartyRelayer(relayerAddress, channelID)
	require.Equal(t, string(key), fmt.Sprintf("%s/%s/%s", types.CounterpartyRelayerAddressKeyPrefix, relayerAddress, channelID))
}
