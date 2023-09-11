package simulation_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/simulation"
)

func TestProposalMsgs(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)

	ctx := sdk.NewContext(nil, cmtproto.Header{}, true, nil)
	accounts := simtypes.RandomAccounts(r, 3)

	// execute ProposalMsgs function
	weightedProposalMsgs := simulation.ProposalMsgs()
	require.Equal(t, 2, len(weightedProposalMsgs))
	w0 := weightedProposalMsgs[0]

	// tests w0 interface:
	require.Equal(t, simulation.OpWeightMsgUpdateParams, w0.AppParamsKey())
	require.Equal(t, simulation.DefaultWeightMsgUpdateParams, w0.DefaultWeight())

	msg := w0.MsgSimulatorFn()(r, ctx, accounts)
	msgUpdateHostParams, ok := msg.(*types.MsgUpdateParams)
	require.True(t, ok)

	require.Equal(t, sdk.AccAddress(address.Module("gov")).String(), msgUpdateHostParams.Signer)
	require.Equal(t, msgUpdateHostParams.Params.HostEnabled, false)

	w1 := weightedProposalMsgs[1]

	// tests w1 interface:
	require.Equal(t, simulation.OpWeightMsgUpdateParams, w1.AppParamsKey())
	require.Equal(t, simulation.DefaultWeightMsgUpdateParams, w1.DefaultWeight())

	msg1 := w1.MsgSimulatorFn()(r, ctx, accounts)
	msgUpdateControllerParams, ok := msg1.(*controllertypes.MsgUpdateParams)
	require.True(t, ok)

	require.Equal(t, sdk.AccAddress(address.Module("gov")).String(), msgUpdateControllerParams.Signer)
	require.Equal(t, msgUpdateControllerParams.Params.ControllerEnabled, false)
}
