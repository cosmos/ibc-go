package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
)

func TestKeyActiveChannel(t *testing.T) {
	key := types.KeyActiveChannel("owner")
	require.Equal(t, string(key), "activeChannel/owner")
}
