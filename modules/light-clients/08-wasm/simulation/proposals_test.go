package simulation_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/simulation"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func TestProposalMsgs(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)

	ctx := sdk.NewContext(nil, cmtproto.Header{}, true, nil)
	accounts := simtypes.RandomAccounts(r, 3)

	// execute ProposalMsgs function
	weightedProposalMsgs := simulation.ProposalMsgs()
	require.Len(t, weightedProposalMsgs, 1)
	w0 := weightedProposalMsgs[0]

	require.Equal(t, simulation.OpWeightMsgStoreCode, w0.AppParamsKey())
	require.Equal(t, simulation.DefaultWeightMsgStoreCode, w0.DefaultWeight())

	msg := w0.MsgSimulatorFn()(r, ctx, accounts)
	msgStoreCode, ok := msg.(*types.MsgStoreCode)
	require.True(t, ok)

	require.Equal(t, sdk.AccAddress(address.Module("gov")).String(), msgStoreCode.Signer)
	require.Equal(t, []byte{0x01}, msgStoreCode.WasmByteCode)
}
