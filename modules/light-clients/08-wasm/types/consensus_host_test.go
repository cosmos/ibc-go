package types_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

func (suite *TypesTestSuite) TestGetSelfConsensusState() {
	var (
		consensusHost  clienttypes.ConsensusHost
		consensusState exported.ConsensusState
		height         clienttypes.Height
	)

	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
			expError: nil,
		},
		{
			name: "failure: delegate error",
			malleate: func() {
				consensusHost.(*mock.ConsensusHost).GetSelfConsensusStateFn = func(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
					return nil, mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
	}

	for i, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			height = clienttypes.ZeroHeight()

			wrappedClientConsensusStateBz := clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientConsensusState)
			consensusState = types.NewConsensusState(wrappedClientConsensusStateBz)

			consensusHost = &mock.ConsensusHost{
				GetSelfConsensusStateFn: func(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
					return consensusState, nil
				},
			}

			tc.malleate()

			var err error
			consensusHost, err = types.NewWasmConsensusHost(suite.chainA.Codec, consensusHost)
			suite.Require().NoError(err)

			suite.chainA.App.GetIBCKeeper().SetConsensusHost(consensusHost)

			cs, err := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetSelfConsensusState(suite.chainA.GetContext(), height)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err, "Case %d should have passed: %s", i, tc.name)
				suite.Require().NotNil(cs, "Case %d should have passed: %s", i, tc.name)
				suite.Require().NotNil(cs.(*types.ConsensusState).Data, "Case %d should have passed: %s", i, tc.name)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "Case %d should have failed: %s", i, tc.name)
				suite.Require().Nil(cs, "Case %d should have failed: %s", i, tc.name)
			}
		})
	}
}
