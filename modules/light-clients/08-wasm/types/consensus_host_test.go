package types_test

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v8/testing/mock"
)

func (suite *TypesTestSuite) TestGetSelfConsensusState() {
	var height clienttypes.Height

	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name: "zero height",
			malleate: func() {
				height = clienttypes.ZeroHeight()
			},
			expError: clienttypes.ErrInvalidHeight,
		},
		{
			name: "height > latest height",
			malleate: func() {
				height = clienttypes.NewHeight(1, uint64(suite.chainA.GetContext().BlockHeight())+1)
			},
			expError: stakingtypes.ErrNoHistoricalInfo,
		},
		{
			name: "pruned historical info",
			malleate: func() {
				height = clienttypes.NewHeight(1, uint64(suite.chainA.GetContext().BlockHeight())-1)

				err := suite.chainA.GetSimApp().StakingKeeper.DeleteHistoricalInfo(suite.chainA.GetContext(), int64(height.GetRevisionHeight()))
				suite.Require().NoError(err)
			},
			expError: stakingtypes.ErrNoHistoricalInfo,
		},
		{
			name: "custom consensus host: failure",
			malleate: func() {
				consensusHost := &mock.ConsensusHost{
					GetSelfConsensusStateFn: func(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
						return nil, mock.MockApplicationCallbackError
					},
				}
				suite.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetSelfConsensusHost(consensusHost)
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "custom consensus host: success",
			malleate: func() {
				consensusHost := &mock.ConsensusHost{
					GetSelfConsensusStateFn: func(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
						return &solomachine.ConsensusState{}, nil
					},
				}
				suite.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetSelfConsensusHost(consensusHost)
			},
			expError: nil,
		},
		{
			name: "latest height - 1",
			malleate: func() {
				height = clienttypes.NewHeight(1, uint64(suite.chainA.GetContext().BlockHeight())-1)
			},
			expError: nil,
		},
		{
			name: "latest height",
			malleate: func() {
				// historical info is set on BeginBlock in x/staking, which is now encapsulated within the FinalizeBlock abci method,
				// thus, we do not have historical info for current height due to how the ibctesting library operates.
				// ibctesting calls app.Commit() as a final step on NextBlock and we invoke test code before FinalizeBlock is called at the current height once again.
				err := suite.chainA.GetSimApp().StakingKeeper.TrackHistoricalInfo(suite.chainA.GetContext())
				suite.Require().NoError(err)

				height = clienttypes.GetSelfHeight(suite.chainA.GetContext())
			},
			expError: nil,
		},
	}

	for i, tc := range cases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			height = clienttypes.ZeroHeight()

			tc.malleate()

			cs, err := suite.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.GetSelfConsensusState(suite.chainA.GetContext(), height)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err, "Case %d should have passed: %s", i, tc.name)
				suite.Require().NotNil(cs, "Case %d should have passed: %s", i, tc.name)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "Case %d should have failed: %s", i, tc.name)
				suite.Require().Nil(cs, "Case %d should have failed: %s", i, tc.name)
			}
		})
	}
}

func (suite *TypesTestSuite) TestValidateSelfClient() {
	var clientState exported.ClientState
	var consensusHost *mock.ConsensusHost
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
				consensusHost.ValidateSelfClientFn = func(ctx sdk.Context, clientState exported.ClientState) error {
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
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
