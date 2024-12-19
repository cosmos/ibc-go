package simulation_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesaddress "github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/simulation"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
)

func TestProposalMsgs(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)

	ctx := sdk.NewContext(nil, true, nil)
	accounts := simtypes.RandomAccounts(r, 3)

	// execute ProposalMsgs function
	weightedProposalMsgs := simulation.ProposalMsgs()
	require.Equal(t, len(weightedProposalMsgs), 1)

	w0 := weightedProposalMsgs[0]

	codec := codecaddress.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	msg, err := w0.MsgSimulatorFn()(ctx, r, accounts, codec)
	require.NoError(t, err)
	msgUpdateParams, ok := msg.(*types.MsgUpdateParams)
	require.True(t, ok)

	require.Equal(t, sdk.AccAddress(typesaddress.Module("gov")).String(), msgUpdateParams.Signer)
	require.EqualValues(t, msgUpdateParams.Params.SendEnabled, false)
}
