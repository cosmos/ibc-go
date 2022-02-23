package ibctesting_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	"github.com/cosmos/ibc-go/v3/testing/mock"
)

func TestCreateSortedSignerArray(t *testing.T) {
	privVal1 := mock.NewPV()
	pubKey1, err := privVal1.GetPubKey()
	require.NoError(t, err)

	privVal2 := mock.NewPV()
	pubKey2, err := privVal2.GetPubKey()
	require.NoError(t, err)

	validator1 := tmtypes.NewValidator(pubKey1, 1)
	validator2 := tmtypes.NewValidator(pubKey2, 2)

	expected := []tmtypes.PrivValidator{privVal2, privVal1}

	actual := ibctesting.CreateSortedSignerArray(privVal1, privVal2, validator1, validator2)
	require.Equal(t, expected, actual)

	// swap order
	actual = ibctesting.CreateSortedSignerArray(privVal2, privVal1, validator2, validator1)
	require.Equal(t, expected, actual)

	// smaller address
	validator1.Address = []byte{1}
	validator2.Address = []byte{2}
	validator2.VotingPower = 1

	expected = []tmtypes.PrivValidator{privVal1, privVal2}

	actual = ibctesting.CreateSortedSignerArray(privVal1, privVal2, validator1, validator2)
	require.Equal(t, expected, actual)

	// swap order
	actual = ibctesting.CreateSortedSignerArray(privVal2, privVal1, validator2, validator1)
	require.Equal(t, expected, actual)
}

func TestChangeValSet(t *testing.T) {
	coord := ibctesting.NewCoordinator(t, 2)
	chainA := coord.GetChain(ibctesting.GetChainID(1))
	chainB := coord.GetChain(ibctesting.GetChainID(2))

	path := ibctesting.NewPath(chainA, chainB)
	coord.Setup(path)

	amount, ok := sdk.NewIntFromString("10000000000000000000")
	require.True(t, ok)

	val := chainA.App.GetStakingKeeper().GetValidators(chainA.GetContext(), 1)

	chainA.App.GetStakingKeeper().Delegate(chainA.GetContext(), chainA.SenderAccounts[1].SenderAccount.GetAddress(),
		amount, types.Unbonded, val[0], true)

	coord.CommitBlock(chainA)

	path.EndpointB.UpdateClient()

	path.EndpointB.UpdateClient()

}
