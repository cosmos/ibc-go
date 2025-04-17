package simulation_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/simulation"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

func TestProposalMsgs(t *testing.T) {
	// Initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)
	ctx := sdk.Context{}
	accounts := simtypes.RandomAccounts(r, 3)

	// Generate the weighted operations
	weightedProposalMsgs := simulation.ProposalMsgs()

	// Require some operations were registered
	require.NotEmpty(t, weightedProposalMsgs)

	// Specifically test the MsgUpdateParams proposal
	require.Equal(t, simulation.OpWeightMsgUpdateParams, weightedProposalMsgs[0].AppParamsKey())
	require.Equal(t, simulation.DefaultWeightMsgUpdateParams, weightedProposalMsgs[0].DefaultWeight())

	// Test the content generator function
	msg := weightedProposalMsgs[0].MsgSimulatorFn()(r, ctx, accounts)
	require.NotNil(t, msg)

	msgUpdateParams, ok := msg.(*types.MsgUpdateParams)
	require.True(t, ok)

	// Test the message is valid
	err := msgUpdateParams.ValidateBasic()
	require.NoError(t, err)

	// Test that the signer is the governance module account
	expectedAddress := address.Module("gov")
	signers := msgUpdateParams.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, sdk.AccAddress(expectedAddress).String(), msgUpdateParams.Signer)

	// Validate params
	require.NoError(t, msgUpdateParams.Params.Validate())
}

func TestRandomizedParams(t *testing.T) {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	// Test RandomEnabled
	for i := 0; i < 100; i++ {
		enabled := simulation.RandomEnabled(r)
		require.NotPanics(t, func() { _ = enabled })
	}

	// Test RandomMaxValue
	for i := 0; i < 100; i++ {
		maxVal := simulation.RandomMaxValue(r, 100000, 10000000)
		require.NotEmpty(t, maxVal)
	}

	// Test RandomPeriod
	for i := 0; i < 100; i++ {
		period := simulation.RandomPeriod(r)
		require.GreaterOrEqual(t, period, uint64(3600)) // 1 hour minimum
		require.LessOrEqual(t, period, uint64(604800))  // 1 week maximum
	}
}
