package simulation_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	controllerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	hostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/simulation"
)

func TestProposalMsgs(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)

	ctx := sdk.NewContext(nil, cmtproto.Header{}, true, nil)
	accounts := simtypes.RandomAccounts(r, 3)

	tests := []struct {
		controller   *controllerkeeper.Keeper
		host         *hostkeeper.Keeper
		proposalMsgs int
		isHost       []bool
	}{
		{
			controller:   &controllerkeeper.Keeper{},
			host:         &hostkeeper.Keeper{},
			proposalMsgs: 2,
			isHost:       []bool{true, false},
		},
		{
			controller:   nil,
			host:         nil,
			proposalMsgs: 0,
		},
		{
			controller:   &controllerkeeper.Keeper{},
			proposalMsgs: 1,
			isHost:       []bool{false},
		},
		{
			host:         &hostkeeper.Keeper{},
			proposalMsgs: 1,
			isHost:       []bool{true},
		},
	}

	for _, test := range tests {
		// execute ProposalMsgs function
		weightedProposalMsgs := simulation.ProposalMsgs(test.controller, test.host)
		require.Equal(t, test.proposalMsgs, len(weightedProposalMsgs))

		for idx, weightedMsg := range weightedProposalMsgs {
			// tests weighted interface:
			require.Equal(t, simulation.OpWeightMsgUpdateParams, weightedMsg.AppParamsKey())
			require.Equal(t, simulation.DefaultWeightMsgUpdateParams, weightedMsg.DefaultWeight())

			msg := weightedMsg.MsgSimulatorFn()(r, ctx, accounts)
			if test.isHost[idx] {
				msgUpdateHostParams, ok := msg.(*types.MsgUpdateParams)
				require.True(t, ok)

				require.Equal(t, sdk.AccAddress(address.Module("gov")).String(), msgUpdateHostParams.Signer)
				require.Equal(t, msgUpdateHostParams.Params.HostEnabled, false)
			} else {
				msgUpdateControllerParams, ok := msg.(*controllertypes.MsgUpdateParams)
				require.True(t, ok)

				require.Equal(t, sdk.AccAddress(address.Module("gov")).String(), msgUpdateControllerParams.Signer)
				require.Equal(t, msgUpdateControllerParams.Params.ControllerEnabled, false)
			}
		}
	}
}
