package types_test

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v8/testing/mock"
)

func (suite *TypesTestSuite) TestGetSelfConsensusState() {

}

func (suite *TypesTestSuite) TestValidateSelfClient() {
	var clientState exported.ClientState
	var consensusHost clienttypes.ConsensusHost
	var cdc codec.BinaryCodec

	testCases := []struct {
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
			name: "failure: invalid data",
			malleate: func() {
				clientState = types.NewClientState(nil, wasmtesting.Code, clienttypes.ZeroHeight())
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "failure: invalid delegate",
			malleate: func() {
				consensusHost = nil
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "failure: invalid codec",
			malleate: func() {
				cdc = nil
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "failure: invalid clientstate type",
			malleate: func() {
				clientState = &ibctm.ClientState{}
			},
			expError: clienttypes.ErrInvalidClient,
		},
		{
			name: "failure: delegate error propagates",
			malleate: func() {
				consensusHost.(*mock.ConsensusHost).ValidateSelfClientFn = func(ctx sdk.Context, clientState exported.ClientState) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			clientState = types.NewClientState(wasmtesting.CreateMockClientStateBz(suite.chainA.Codec, suite.checksum), wasmtesting.Code, clienttypes.ZeroHeight())
			consensusHost = &mock.ConsensusHost{}
			cdc = suite.chainA.Codec

			tc.malleate()

			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetSelfConsensusHost(
				types.NewWasmConsensusHost(cdc, consensusHost),
			)

			err := suite.chainA.App.GetIBCKeeper().ClientKeeper.ValidateSelfClient(suite.chainA.GetContext(), clientState)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err, "expected valid client for case: %s", tc.name)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "expected %s got %s", tc.expError, err)
			}
		})
	}
}
