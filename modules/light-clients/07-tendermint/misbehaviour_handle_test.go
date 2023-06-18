package tendermint_test

import (
	"fmt"
	"strings"
	"time"

	tmtypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibctestingmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *TendermintTestSuite) TestVerifyMisbehaviour() {
	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	s.Require().NoError(err)

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

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+3, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour, header 1 time stricly less than header 2 time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+3, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Hour), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, true,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals1, found := s.chainB.GetValsAtHeight(int64(trustedHeight1.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				trustedHeight2 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals2, found := s.chainB.GetValsAtHeight(int64(trustedHeight2.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight1, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals1, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight2, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals2, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehaviour at a previous revision", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}

				// increment revision number
				err = path.EndpointB.UpgradeChain()
				s.Require().NoError(err)
			},
			true,
		},
		{
			"valid misbehaviour at a future revision", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				futureRevision := fmt.Sprintf("%s-%d", strings.TrimSuffix(s.chainB.ChainID, fmt.Sprintf("-%d", clienttypes.ParseChainID(s.chainB.ChainID))), height.GetRevisionNumber()+1)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehaviour with trusted heights at a previous revision", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				// increment revision of chainID
				err = path.EndpointB.UpgradeChain()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(s.chainB.Vals.Validators, altValSet.Proposer))
				bothSigners := s.chainB.Signers
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), bothValSet, s.chainB.NextVals, trustedVals, bothSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, bothValSet, s.chainB.NextVals, trustedVals, bothSigners),
				}
			}, true,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, false,
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, altValSet, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, altValSet, s.chainB.Signers),
				}
			}, false,
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight.Increment().(clienttypes.Height), s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
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
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				s.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig).TrustingPeriod)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, false,
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, s.chainB.NextVals, trustedVals, altSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, false,
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, altValSet, s.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, s.chainB.NextVals, trustedVals, altSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, altValSet, s.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			err = clientState.VerifyClientMessage(s.chainA.GetContext(), s.chainA.App.AppCodec(), clientStore, misbehaviour)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// test both fork and time misbehaviour for chainIDs not in the revision format
// this function is separate as it must use a global variable in the testing package
// to initialize chains not in the revision format
func (s *TendermintTestSuite) TestVerifyMisbehaviourNonRevisionChainID() {
	// NOTE: chains set to non revision format
	ibctesting.ChainIDSuffix = ""

	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	s.Require().NoError(err)

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

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+3, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid time misbehaviour, header 1 time stricly less than header 2 time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+3, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Hour), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, true,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals1, found := s.chainB.GetValsAtHeight(int64(trustedHeight1.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				trustedHeight2 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals2, found := s.chainB.GetValsAtHeight(int64(trustedHeight2.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight1, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals1, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight2, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals2, s.chainB.Signers),
				}
			},
			true,
		},
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(s.chainB.Vals.Validators, altValSet.Proposer))
				bothSigners := s.chainB.Signers
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), bothValSet, s.chainB.NextVals, trustedVals, bothSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, bothValSet, s.chainB.NextVals, trustedVals, bothSigners),
				}
			}, true,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, false,
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, altValSet, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, altValSet, s.chainB.Signers),
				}
			}, false,
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight.Increment().(clienttypes.Height), s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
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
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				s.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig).TrustingPeriod)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, false,
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, s.chainB.NextVals, trustedVals, altSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, false,
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, altValSet, s.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, s.chainB.NextVals, trustedVals, altSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, altValSet, s.chainB.NextVals, trustedVals, altSigners),
				}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			err = clientState.VerifyClientMessage(s.chainA.GetContext(), s.chainA.App.AppCodec(), clientStore, misbehaviour)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}

	// NOTE: reset chain creation to revision format
	ibctesting.ChainIDSuffix = "-1"
}
