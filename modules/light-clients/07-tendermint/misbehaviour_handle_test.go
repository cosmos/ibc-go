package tendermint_test

import (
	"errors"
	"fmt"
	"strings"
	"time"

	cmttypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *TendermintTestSuite) TestVerifyMisbehaviour() {
	// Setup different validators and signers for testing different types of updates
	altPrivVal := cmttypes.NewMockPV()
	altPubKey, err := altPrivVal.GetPubKey()
	s.Require().NoError(err)

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
		expErr   error
	}{
		{
			"valid fork misbehaviour", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+3, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"valid time misbehaviour, header 1 time strictly less than header 2 time", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+3, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Hour), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, nil,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals1, ok := s.chainB.TrustedValidators[trustedHeight1.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				trustedHeight2, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals2, ok := s.chainB.TrustedValidators[trustedHeight2.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight1, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals1, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight2, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals2, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"valid misbehaviour at a previous revision", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}

				// increment revision number
				err = path.EndpointB.UpgradeChain()
				s.Require().NoError(err)
			},
			nil,
		},
		{
			"valid misbehaviour at a future revision", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				futureRevision := fmt.Sprintf("%s-%d", strings.TrimSuffix(s.chainB.ChainID, fmt.Sprintf("-%d", clienttypes.ParseChainID(s.chainB.ChainID))), height.GetRevisionNumber()+1)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"valid misbehaviour with trusted heights at a previous revision", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				// increment revision of chainID
				err = path.EndpointB.UpgradeChain()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(s.chainB.Vals.Validators, altValSet.Proposer))
				bothSigners := s.chainB.Signers
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), bothValSet, s.chainB.NextVals, trustedVals, bothSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, bothValSet, s.chainB.NextVals, trustedVals, bothSigners),
				}
			}, nil,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, errors.New("invalid light client misbehaviour"),
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, altValSet, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, altValSet, s.chainB.Signers),
				}
			}, errors.New("invalid validator set"),
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight.Increment().(clienttypes.Height), s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, errors.New("consensus state not found"),
		},
		{
			"invalid tendermint misbehaviour", func() {
				misbehaviour = &solomachine.Misbehaviour{}
			}, errors.New("invalid client type"),
		},
		{
			"trusting period expired", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				s.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig).TrustingPeriod)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, errors.New("time since latest trusted state has passed the trusting period"),
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), altValSet, s.chainB.NextVals, trustedVals, altSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, errors.New("validator set in header has too much change from trusted validator set"),
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, altValSet, s.chainB.NextVals, trustedVals, altSigners),
				}
			}, errors.New("validator set in header has too much change from trusted validator set"),
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), altValSet, s.chainB.NextVals, trustedVals, altSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, altValSet, s.chainB.NextVals, trustedVals, altSigners),
				}
			}, errors.New("validator set in header has too much change from trusted validator set"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(s.chainA.GetContext(), path.EndpointA.ClientID, misbehaviour)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expErr.Error())
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
	altPrivVal := cmttypes.NewMockPV()
	altPubKey, err := altPrivVal.GetPubKey()
	s.Require().NoError(err)

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
		expErr   error
	}{
		{
			"valid fork misbehaviour", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"valid time misbehaviour", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+3, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"valid time misbehaviour, header 1 time strictly less than header 2 time", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+3, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Hour), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"valid misbehavior at height greater than last consensusState", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, nil,
		},
		{
			"valid misbehaviour with different trusted heights", func() {
				trustedHeight1, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals1, ok := s.chainB.TrustedValidators[trustedHeight1.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				trustedHeight2, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals2, ok := s.chainB.TrustedValidators[trustedHeight2.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight1, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals1, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight2, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals2, s.chainB.Signers),
				}
			},
			nil,
		},
		{
			"consensus state's valset hash different from misbehaviour should still pass", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(s.chainB.Vals.Validators, altValSet.Proposer))
				bothSigners := s.chainB.Signers
				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), bothValSet, s.chainB.NextVals, trustedVals, bothSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, bothValSet, s.chainB.NextVals, trustedVals, bothSigners),
				}
			}, nil,
		},
		{
			"invalid misbehaviour: misbehaviour from different chain", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, errors.New("validator set in header has too much change from trusted validator set"),
		},
		{
			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, altValSet, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, altValSet, s.chainB.Signers),
				}
			}, errors.New("invalid validator set"),
		},
		{
			"trusted consensus state does not exist", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight.Increment().(clienttypes.Height), s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, errors.New("consensus state not found"),
		},
		{
			"invalid tendermint misbehaviour", func() {
				misbehaviour = &solomachine.Misbehaviour{}
			}, errors.New("nvalid client type"),
		},
		{
			"trusting period expired", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				s.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig).TrustingPeriod)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, errors.New("time since latest trusted state has passed the trusting period"),
		},
		{
			"header 1 valset has too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), altValSet, s.chainB.NextVals, trustedVals, altSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, errors.New("validator set in header has too much change from trusted validator set"),
		},
		{
			"header 2 valset has too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, altValSet, s.chainB.NextVals, trustedVals, altSigners),
				}
			}, errors.New("validator set in header has too much change from trusted validator set"),
		},
		{
			"both header 1 and header 2 valsets have too much change", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviour = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), altValSet, s.chainB.NextVals, trustedVals, altSigners),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, altValSet, s.chainB.NextVals, trustedVals, altSigners),
				}
			}, errors.New("validator set in header has too much change from trusted validator set"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(s.chainA.GetContext(), path.EndpointA.ClientID, misbehaviour)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}

	// NOTE: reset chain creation to revision format
	ibctesting.ChainIDSuffix = "-1"
}
