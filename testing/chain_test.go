package ibctesting_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctesting "github.com/cosmos/ibc-go/v4/testing"
)

func TestChangeValSet(t *testing.T) {
	coord := ibctesting.NewCoordinator(t, 2)
	chainA := coord.GetChain(ibctesting.GetChainID(1))
	chainB := coord.GetChain(ibctesting.GetChainID(2))

	path := ibctesting.NewPath(chainA, chainB)
	coord.Setup(path)

	amount, ok := sdk.NewIntFromString("10000000000000000000")
	require.True(t, ok)
	amount2, ok := sdk.NewIntFromString("30000000000000000000")
	require.True(t, ok)

	val := chainA.App.GetStakingKeeper().GetValidators(chainA.GetContext(), 4)

	chainA.App.GetStakingKeeper().Delegate(chainA.GetContext(), chainA.SenderAccounts[1].SenderAccount.GetAddress(),
		amount, types.Unbonded, val[1], true)
	chainA.App.GetStakingKeeper().Delegate(chainA.GetContext(), chainA.SenderAccounts[3].SenderAccount.GetAddress(),
		amount2, types.Unbonded, val[3], true)

	coord.CommitBlock(chainA)

	// verify that update clients works even after validator update goes into effect
	path.EndpointB.UpdateClient()
	path.EndpointB.UpdateClient()
}
