package tendermint_test

import (
	"fmt"
	"strings"
	"time"

	cmttypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *TendermintTestSuite) TestVerifyMisbehaviour() {
	// Setup different validators and signers for testing different types of updates
	altPrivVal := cmttypes.NewMockPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	// create modified heights to use for test-cases
	altVal := cmttypes.NewValidator(altPubKey, 100)

	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{altVal})
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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+3, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour, header 1 time strictly less than header 2 time", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+3, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Hour), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, true,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals1, err := suite.chainB.GetTrustedValidators(int64(trustedHeight1.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				trustedHeight2, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals2, err := suite.chainB.GetTrustedValidators(int64(trustedHeight2.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight1, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals1, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight2, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals2, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehaviour at a previous revision", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}

				// increment revision number
				err = path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"valid misbehaviour at a future revision", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				futureRevision := fmt.Sprintf("%s-%d", strings.TrimSuffix(suite.chainB.ChainID, fmt.Sprintf("-%d", clienttypes.ParseChainID(suite.chainB.ChainID))), height.GetRevisionNumber()+1)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehaviour with trusted heights at a previous revision", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				// increment revision of chainID
				err = path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altValSet.Proposer))
				bothSigners := suite.chainB.Signers
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
				}
			}, true,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight.Increment().(clienttypes.Height), suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"invalid tendermint misbehaviour", func() {
				misbehaviour = &solomachine.Misbehaviour{}
			}, false,
		},
		{
			"trusting period expired", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				suite.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig).TrustingPeriod)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
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

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(suite.chainA.GetContext(), path.EndpointA.ClientID, misbehaviour)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// test both fork and time misbehaviour for chainIDs not in the revision format
// this function is separate as it must use a global variable in the testing package
// to initialize chains not in the revision format
func (suite *TendermintTestSuite) TestVerifyMisbehaviourNonRevisionChainID() {
	// NOTE: chains set to non revision format
	ibctesting.ChainIDSuffix = ""

	// Setup different validators and signers for testing different types of updates
	altPrivVal := cmttypes.NewMockPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	// create modified heights to use for test-cases
	altVal := cmttypes.NewValidator(altPubKey, 100)

	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{altVal})
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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+3, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour, header 1 time strictly less than header 2 time", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+3, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Hour), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, true,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals1, err := suite.chainB.GetTrustedValidators(int64(trustedHeight1.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				trustedHeight2, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals2, err := suite.chainB.GetTrustedValidators(int64(trustedHeight2.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight1, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals1, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight2, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals2, suite.chainB.Signers),
				}
			},
			true,
		},
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altValSet.Proposer))
				bothSigners := suite.chainB.Signers
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
				}
			}, true,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight.Increment().(clienttypes.Height), suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"invalid tendermint misbehaviour", func() {
				misbehaviour = &solomachine.Misbehaviour{}
			}, false,
		},
		{
			"trusting period expired", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				suite.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig).TrustingPeriod)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
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

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(suite.chainA.GetContext(), path.EndpointA.ClientID, misbehaviour)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}

	// NOTE: reset chain creation to revision format
	ibctesting.ChainIDSuffix = "-1"
}
