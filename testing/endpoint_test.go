package ibctesting_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	ibctmtypes "github.com/cosmos/ibc-go/v3/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func TestUpgradeChain(t *testing.T) {
	coord := ibctesting.NewCoordinator(t, 2)
	chainA := coord.GetChain(ibctesting.GetChainID(1))
	chainB := coord.GetChain(ibctesting.GetChainID(2))

	path := ibctesting.NewPath(chainA, chainB)
	err := path.EndpointA.CreateClient()
	require.NoError(t, err)

	err = path.EndpointB.UpgradeChain(path.EndpointA.GetClientState().(*ibctmtypes.ClientState))
	require.NoError(t, err)
}
