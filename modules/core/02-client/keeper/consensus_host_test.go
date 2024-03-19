package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/cosmos/ibc-go/v8/testing/mock"
)

func (suite *KeeperTestSuite) TestGetSelfConsensusState() {
	var height types.Height

	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name: "zero height",
			malleate: func() {
				height = types.ZeroHeight()
			},
			expError: stakingtypes.ErrNoHistoricalInfo,
		},
		{
			name: "height > latest height",
			malleate: func() {
				height = types.NewHeight(0, uint64(suite.ctx.BlockHeight())+1)
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
				suite.keeper.SetSelfConsensusHost(consensusHost)
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
				suite.keeper.SetSelfConsensusHost(consensusHost)
			},
			expError: nil,
		},
		{
			name: "latest height - 1",
			malleate: func() {
				height = types.NewHeight(0, uint64(suite.ctx.BlockHeight())-1)
			},
			expError: nil,
		},
		{
			name: "latest height",
			malleate: func() {
				height = types.GetSelfHeight(suite.ctx)
			},
			expError: nil,
		},
	}

	for i, tc := range cases {
		suite.SetupTest()
		suite.ctx = suite.ctx.WithBlockHeight(10)
		tc := tc

		height = types.ZeroHeight()

		tc.malleate()

		cs, err := suite.keeper.GetSelfConsensusState(suite.ctx, height)

		expPass := tc.expError == nil
		if expPass {
			suite.Require().NoError(err, "Case %d should have passed: %s", i, tc.name)
			suite.Require().NotNil(cs, "Case %d should have passed: %s", i, tc.name)
		} else {
			suite.Require().ErrorIs(err, tc.expError, "Case %d should have failed: %s", i, tc.name)
			suite.Require().Nil(cs, "Case %d should have failed: %s", i, tc.name)
		}
	}
}

func (suite *KeeperTestSuite) TestValidateSelfClient() {
	testClientHeight := types.GetSelfHeight(suite.chainA.GetContext())
	testClientHeight.RevisionHeight--

	var clientState exported.ClientState

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name: "success",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			},
			expError: nil,
		},
		{
			name: "success with nil UpgradePath",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), nil)
			},
			expError: nil,
		},
		{
			name: "success with custom self validator: solomachine",
			malleate: func() {
				clientState = solomachine.NewClientState(1, &solomachine.ConsensusState{})

				smConsensusHost := &mock.ConsensusHost{
					ValidateSelfClientFn: func(ctx sdk.Context, clientState exported.ClientState) error {
						smClientState, ok := clientState.(*solomachine.ClientState)
						suite.Require().True(ok)
						suite.Require().Equal(uint64(1), smClientState.Sequence)

						return nil
					},
				}

				// add mock validation logic
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetSelfConsensusHost(smConsensusHost)
			},
			expError: nil,
		},
		{
			name: "frozen client",
			malleate: func() {
				clientState = &ibctm.ClientState{ChainId: suite.chainA.ChainID, TrustLevel: ibctm.DefaultTrustLevel, TrustingPeriod: trustingPeriod, UnbondingPeriod: ubdPeriod, MaxClockDrift: maxClockDrift, FrozenHeight: testClientHeight, LatestHeight: testClientHeight, ProofSpecs: commitmenttypes.GetSDKSpecs(), UpgradePath: ibctesting.UpgradePath}
			},
			expError: types.ErrClientFrozen,
		},
		{
			name: "incorrect chainID",
			malleate: func() {
				clientState = ibctm.NewClientState("gaiatestnet", ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			},
			expError: types.ErrInvalidClient,
		},
		{
			name: "invalid client height",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.GetSelfHeight(suite.chainA.GetContext()).Increment().(types.Height), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			},
			expError: types.ErrInvalidClient,
		},
		{
			name: "invalid client type",
			malleate: func() {
				clientState = solomachine.NewClientState(0, &solomachine.ConsensusState{PublicKey: suite.solomachine.ConsensusState().PublicKey, Diversifier: suite.solomachine.Diversifier, Timestamp: suite.solomachine.Time})
			},
			expError: types.ErrInvalidClient,
		},
		{
			name: "invalid client revision",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeightRevision1, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			},
			expError: types.ErrInvalidClient,
		},
		{
			name: "invalid proof specs",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, nil, ibctesting.UpgradePath)
			},
			expError: types.ErrInvalidClient,
		},
		{
			name: "invalid trust level",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.Fraction{Numerator: 0, Denominator: 1}, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			},
			expError: types.ErrInvalidClient,
		},
		{
			name: "invalid unbonding period",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+10, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			},
			expError: types.ErrInvalidClient,
		},
		{
			name: "invalid trusting period",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ubdPeriod+10, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			},
			expError: types.ErrInvalidClient,
		},
		{
			name: "invalid upgrade path",
			malleate: func() {
				clientState = ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), []string{"bad", "upgrade", "path"})
			},
			expError: types.ErrInvalidClient,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			err := suite.chainA.App.GetIBCKeeper().ClientKeeper.ValidateSelfClient(suite.chainA.GetContext(), clientState)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err, "expected valid client for case: %s", tc.name)
			} else {
				suite.Require().Error(err, "expected invalid client for case: %s", tc.name)
			}
		})
	}
}
