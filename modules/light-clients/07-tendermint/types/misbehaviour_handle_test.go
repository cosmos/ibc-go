package types_test

import (
	"time"

	tmtypes "github.com/tendermint/tendermint/types"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	smtypes "github.com/cosmos/ibc-go/v3/modules/light-clients/06-solomachine/types"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	ibctestingmock "github.com/cosmos/ibc-go/v3/testing/mock"
)

func (suite *TendermintTestSuite) TestVerifyMisbehaviour() {
	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	// create modified heights to use for test-cases
	altVal := tmtypes.NewValidator(altPubKey, 100)

	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})
	altSigners := getAltSigners(altVal, altPrivVal)

	var (
		path         *ibctesting.Path
		misbehaviour exported.ClientMessage
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid fork misbehaviour", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour, header 1 time stricly less than header 2 time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Hour), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, true,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals1, found := suite.chainB.GetValsAtHeight(int64(trustedHeight1.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				trustedHeight2 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals2, found := suite.chainB.GetValsAtHeight(int64(trustedHeight2.RevisionHeight) + 1)
				suite.Require().True(found)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight1, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals1, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight2, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals2, suite.chainB.Signers),
				}

			},
			true,
		},
		/*
			{

				"valid misbehaviour at a previous revision",
				types.NewClientState(chainIDRevision1, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.NewHeight(1, 1), commitmenttypes.GetSDKSpecs(), upgradePath, false, false),
				types.NewConsensusState(suite.now, commitmenttypes.NewMerkleRoot(tmhash.Sum([]byte("app_hash"))), bothValsHash),
				heightMinus1,
				types.NewConsensusState(suite.now, commitmenttypes.NewMerkleRoot(tmhash.Sum([]byte("app_hash"))), suite.valsHash),
				heightMinus3,
				&types.Misbehaviour{
					Header1:  suite.chainA.CreateTMClientHeader(chainIDRevision0, int64(height.RevisionHeight+1), heightMinus1, suite.now, bothValSet, bothValSet, bothValSet, bothSigners),
					Header2:  suite.chainA.CreateTMClientHeader(chainIDRevision0, int64(height.RevisionHeight+1), heightMinus3, suite.now.Add(time.Minute), bothValSet, bothValSet, suite.valSet, bothSigners),
					ClientId: chainID,
				},
				suite.now,
				true,
			},
			{
				"valid misbehaviour at a future revision",
				types.NewClientState(chainIDRevision0, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false),
				types.NewConsensusState(suite.now, commitmenttypes.NewMerkleRoot(tmhash.Sum([]byte("app_hash"))), bothValsHash),
				heightMinus1,
				types.NewConsensusState(suite.now, commitmenttypes.NewMerkleRoot(tmhash.Sum([]byte("app_hash"))), suite.valsHash),
				heightMinus3,
				&types.Misbehaviour{
					Header1:  suite.chainA.CreateTMClientHeader(chainIDRevision0, 3, heightMinus1, suite.now, bothValSet, bothValSet, bothValSet, bothSigners),
					Header2:  suite.chainA.CreateTMClientHeader(chainIDRevision0, 3, heightMinus3, suite.now.Add(time.Minute), bothValSet, bothValSet, suite.valSet, bothSigners),
					ClientId: chainID,
				},
				suite.now,
				true,
			},
			{
				"valid misbehaviour with trusted heights at a previous revision",
				types.NewClientState(chainIDRevision1, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.NewHeight(1, 1), commitmenttypes.GetSDKSpecs(), upgradePath, false, false),
				types.NewConsensusState(suite.now, commitmenttypes.NewMerkleRoot(tmhash.Sum([]byte("app_hash"))), bothValsHash),
				heightMinus1,
				types.NewConsensusState(suite.now, commitmenttypes.NewMerkleRoot(tmhash.Sum([]byte("app_hash"))), suite.valsHash),
				heightMinus3,
				&types.Misbehaviour{
					Header1:  suite.chainA.CreateTMClientHeader(chainIDRevision1, 1, heightMinus1, suite.now, bothValSet, bothValSet, bothValSet, bothSigners),
					Header2:  suite.chainA.CreateTMClientHeader(chainIDRevision1, 1, heightMinus3, suite.now.Add(time.Minute), bothValSet, bothValSet, suite.valSet, bothSigners),
					ClientId: chainID,
				},
				suite.now,
				true,
			},
		*/
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altValSet.Proposer))
				bothSigners := suite.chainB.Signers
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
				}
			}, true,
		},
		{
			"invalid fork misbehaviour: identical headers", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviourHeader := suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
				misbehaviour = &types.Misbehaviour{
					Header1: misbehaviourHeader,
					Header2: misbehaviourHeader,
				}
			}, false,
		},
		{
			"invalid time misbehaviour: monotonically increasing time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}

			}, false,
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight.Increment().(clienttypes.Height), suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"invalid tendermint misbehaviour", func() {
				misbehaviour = &smtypes.Misbehaviour{}
			}, false,
		},
		{
			"trusting period expired", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				suite.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig).TrustingPeriod)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &types.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			clientState := path.EndpointA.GetClientState()

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, misbehaviour)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
