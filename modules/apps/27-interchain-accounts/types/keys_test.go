package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
)

func TestKeyActiveChannel(t *testing.T) {
	key := types.KeyActiveChannel("owner")
	require.Equal(t, string(key), "activeChannel/owner")
}

func TestGetIdentifier(t *testing.T) {
	identifier := types.GetIdentifier(types.PortID, "channel-0")
	require.Equal(t, identifier, fmt.Sprintf("%s/channel-0/", types.PortID))
}
