package tendermint_test

import (
	"time"

	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibctestingmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *TendermintTestSuite) TestVerifyHeader() {
	var (
		path   *ibctesting.Path
		header *ibctm.Header
	)

	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	s.Require().NoError(err)

	revisionHeight := int64(height.RevisionHeight)

	// create modified heights to use for test-cases
	altVal := tmtypes.NewValidator(altPubKey, 100)
	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})
	altSigners := getAltSigners(altVal, altPrivVal)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name:     "success",
			malleate: func() {},
			expPass:  true,
		},
		{
			name: "successful verify header for header with a previous height",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)

				// commit some blocks so that the created Header now has a previous height as the BlockHeight
				s.coordinator.CommitNBlocks(s.chainB, 5)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)
			},
			expPass: true,
		},
		{
			name: "successful verify header: header with future height and different validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(s.chainB.Vals.Validators, altVal))
				bothSigners := s.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+5, trustedHeight, s.chainB.CurrentHeader.Time, bothValSet, s.chainB.NextVals, trustedVals, bothSigners)
			},
			expPass: true,
		},
		{
			name: "successful verify header: header with next height and different validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(s.chainB.Vals.Validators, altVal))
				bothSigners := s.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, bothValSet, s.chainB.NextVals, trustedVals, bothSigners)
			},
			expPass: true,
		},
		{
			name: "unsuccessful updates, passed in incorrect trusted validators for given consensus state",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(s.chainB.Vals.Validators, altVal))
				bothSigners := s.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time, bothValSet, bothValSet, bothValSet, bothSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header with next height: update header mismatches nextValSetHash",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				// this will err as altValSet.Hash() != consState.NextValidatorsHash
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update with future height: too much change in validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				header = s.chainB.CreateTMClientHeader(chainIDRevision1, 3, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header height < consensus height",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				heightMinus1 := clienttypes.NewHeight(trustedHeight.RevisionNumber, trustedHeight.RevisionHeight-1)

				// Make new header at height less than latest client state
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(heightMinus1.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header basic validation failed",
			malleate: func() {
				// cause header to fail validatebasic by changing commit height to mismatch header height
				header.SignedHeader.Commit.Height = revisionHeight - 1
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header timestamp is not past last client timestamp",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				s.Require().True(found)

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time.Add(-time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header with incorrect header chain-id",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				s.Require().True(found)

				header = s.chainB.CreateTMClientHeader(chainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: trusting period has passed since last client timestamp",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				s.Require().True(found)

				header = s.chainA.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+1, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)

				s.chainB.ExpireClient(ibctesting.TrustingPeriod)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update for a previous revision",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become an update to previous revision once we upgrade the client
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)

				// increment the revision of the chain
				err = path.EndpointB.UpgradeChain()
				s.Require().NoError(err)
			},
			expPass: false,
		},
		{
			name: "successful update with identical header to a previous update",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)

				// update client so the header constructed becomes a duplicate
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)
			},
			expPass: true,
		},

		{
			name: "unsuccessful update to a future revision",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID+"-1", s.chainB.CurrentHeader.Height+5, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expPass: false,
		},

		{
			name: "unsuccessful update: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				// increment the revision of the chain
				err = path.EndpointB.UpgradeChain()
				s.Require().NoError(err)

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.SetupTest()
		path = ibctesting.NewPath(s.chainA, s.chainB)

		err := path.EndpointA.CreateClient()
		s.Require().NoError(err)

		// ensure counterparty state is committed
		s.coordinator.CommitBlock(s.chainB)
		header, err = path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
		s.Require().NoError(err)

		tc.malleate()

		clientState := path.EndpointA.GetClientState()

		clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

		err = clientState.VerifyClientMessage(s.chainA.GetContext(), s.chainA.App.AppCodec(), clientStore, header)

		if tc.expPass {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().Error(err)
		}
	}
}

func (s *TendermintTestSuite) TestUpdateState() {
	var (
		path               *ibctesting.Path
		clientMessage      exported.ClientMessage
		clientStore        sdk.KVStore
		consensusHeights   []exported.Height
		pruneHeight        clienttypes.Height
		prevClientState    exported.ClientState
		prevConsensusState exported.ConsensusState
	)

	testCases := []struct {
		name      string
		malleate  func()
		expResult func()
		expPass   bool
	}{
		{
			"success with height later than latest height", func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)
				s.Require().True(path.EndpointA.GetClientState().GetLatestHeight().LT(tmHeader.GetHeight()))
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				clientState := path.EndpointA.GetClientState()
				s.Require().True(clientState.GetLatestHeight().EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				s.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))
			}, true,
		},
		{
			"success with height earlier than latest height", func() {
				// commit a block so the pre-created ClientMessage
				// isn't used to update the client to a newer height
				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)
				s.Require().True(path.EndpointA.GetClientState().GetLatestHeight().GT(tmHeader.GetHeight()))

				prevClientState = path.EndpointA.GetClientState()
			},
			func() {
				clientState := path.EndpointA.GetClientState()
				s.Require().Equal(clientState, prevClientState) // fill in height, no change to client state
				s.Require().True(clientState.GetLatestHeight().GT(consensusHeights[0]))
			}, true,
		},
		{
			"success with duplicate header", func() {
				// update client in advance
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				// use the same header which just updated the client
				clientMessage, err = path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				s.Require().NoError(err)

				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)
				s.Require().Equal(path.EndpointA.GetClientState().GetLatestHeight(), tmHeader.GetHeight())

				prevClientState = path.EndpointA.GetClientState()
				prevConsensusState = path.EndpointA.GetConsensusState(tmHeader.GetHeight())
			},
			func() {
				clientState := path.EndpointA.GetClientState()
				s.Require().Equal(clientState, prevClientState)
				s.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))

				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)
				s.Require().Equal(path.EndpointA.GetConsensusState(tmHeader.GetHeight()), prevConsensusState)
			}, true,
		},
		{
			"success with pruned consensus state", func() {
				// this height will be expired and pruned
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)
				pruneHeight = path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Increment the time by a week
				s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

				// create the consensus state that can be used as trusted height for next update
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				// Increment the time by another week, then update the client.
				// This will cause the first two consensus states to become expired.
				s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				// ensure counterparty state is committed
				s.coordinator.CommitBlock(s.chainB)
				clientMessage, err = path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				s.Require().NoError(err)
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				clientState := path.EndpointA.GetClientState()
				s.Require().True(clientState.GetLatestHeight().EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				s.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))

				// ensure consensus state was pruned
				_, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				s.Require().False(found)
			}, true,
		},
		{
			"success with pruned consensus state using duplicate header", func() {
				// this height will be expired and pruned
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)
				pruneHeight = path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// assert that a consensus state exists at the prune height
				consensusState, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				s.Require().True(found)
				s.Require().NotNil(consensusState)

				// Increment the time by a week
				s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

				// create the consensus state that can be used as trusted height for next update
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				// Increment the time by another week, then update the client.
				// This will cause the first two consensus states to become expired.
				s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				// use the same header which just updated the client
				clientMessage, err = path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				s.Require().NoError(err)
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				clientState := path.EndpointA.GetClientState()
				s.Require().True(clientState.GetLatestHeight().EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				s.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))

				// ensure consensus state was pruned
				_, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				s.Require().False(found)
			}, true,
		},
		{
			"invalid ClientMessage type", func() {
				clientMessage = &ibctm.Misbehaviour{}
			},
			func() {},
			false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			pruneHeight = clienttypes.ZeroHeight()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			// ensure counterparty state is committed
			s.coordinator.CommitBlock(s.chainB)
			clientMessage, err = path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore = s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			if tc.expPass {
				consensusHeights = clientState.UpdateState(s.chainA.GetContext(), s.chainA.App.AppCodec(), clientStore, clientMessage)

				header := clientMessage.(*ibctm.Header)
				expConsensusState := &ibctm.ConsensusState{
					Timestamp:          header.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()),
					NextValidatorsHash: header.Header.NextValidatorsHash,
				}

				bz := clientStore.Get(host.ConsensusStateKey(header.GetHeight()))
				updatedConsensusState := clienttypes.MustUnmarshalConsensusState(s.chainA.App.AppCodec(), bz)

				s.Require().Equal(expConsensusState, updatedConsensusState)

			} else {
				s.Require().Panics(func() {
					clientState.UpdateState(s.chainA.GetContext(), s.chainA.App.AppCodec(), clientStore, clientMessage)
				})
			}

			// perform custom checks
			tc.expResult()
		})
	}
}

func (s *TendermintTestSuite) TestPruneConsensusState() {
	// create path and setup clients
	path := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(path)

	// get the first height as it will be pruned first.
	var pruneHeight exported.Height
	getFirstHeightCb := func(height exported.Height) bool {
		pruneHeight = height
		return true
	}
	ctx := path.EndpointA.Chain.GetContext()
	clientStore := path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)
	ibctm.IterateConsensusStateAscending(clientStore, getFirstHeightCb)

	// this height will be expired but not pruned
	err := path.EndpointA.UpdateClient()
	s.Require().NoError(err)
	expiredHeight := path.EndpointA.GetClientState().GetLatestHeight()

	// expected values that must still remain in store after pruning
	expectedConsState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, expiredHeight)
	s.Require().True(ok)
	ctx = path.EndpointA.Chain.GetContext()
	clientStore = path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)
	expectedProcessTime, ok := ibctm.GetProcessedTime(clientStore, expiredHeight)
	s.Require().True(ok)
	expectedProcessHeight, ok := ibctm.GetProcessedHeight(clientStore, expiredHeight)
	s.Require().True(ok)
	expectedConsKey := ibctm.GetIterationKey(clientStore, expiredHeight)
	s.Require().NotNil(expectedConsKey)

	// Increment the time by a week
	s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

	// create the consensus state that can be used as trusted height for next update
	err = path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	// Increment the time by another week, then update the client.
	// This will cause the first two consensus states to become expired.
	s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
	err = path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	ctx = path.EndpointA.Chain.GetContext()
	clientStore = path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

	// check that the first expired consensus state got deleted along with all associated metadata
	consState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
	s.Require().Nil(consState, "expired consensus state not pruned")
	s.Require().False(ok)
	// check processed time metadata is pruned
	processTime, ok := ibctm.GetProcessedTime(clientStore, pruneHeight)
	s.Require().Equal(uint64(0), processTime, "processed time metadata not pruned")
	s.Require().False(ok)
	processHeight, ok := ibctm.GetProcessedHeight(clientStore, pruneHeight)
	s.Require().Nil(processHeight, "processed height metadata not pruned")
	s.Require().False(ok)

	// check iteration key metadata is pruned
	consKey := ibctm.GetIterationKey(clientStore, pruneHeight)
	s.Require().Nil(consKey, "iteration key not pruned")

	// check that second expired consensus state doesn't get deleted
	// this ensures that there is a cap on gas cost of UpdateClient
	consState, ok = path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, expiredHeight)
	s.Require().Equal(expectedConsState, consState, "consensus state incorrectly pruned")
	s.Require().True(ok)
	// check processed time metadata is not pruned
	processTime, ok = ibctm.GetProcessedTime(clientStore, expiredHeight)
	s.Require().Equal(expectedProcessTime, processTime, "processed time metadata incorrectly pruned")
	s.Require().True(ok)

	// check processed height metadata is not pruned
	processHeight, ok = ibctm.GetProcessedHeight(clientStore, expiredHeight)
	s.Require().Equal(expectedProcessHeight, processHeight, "processed height metadata incorrectly pruned")
	s.Require().True(ok)

	// check iteration key metadata is not pruned
	consKey = ibctm.GetIterationKey(clientStore, expiredHeight)
	s.Require().Equal(expectedConsKey, consKey, "iteration key incorrectly pruned")
}

func (s *TendermintTestSuite) TestCheckForMisbehaviour() {
	var (
		path          *ibctesting.Path
		clientMessage exported.ClientMessage
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid update no misbehaviour",
			func() {},
			false,
		},
		{
			"consensus state already exists, already updated",
			func() {
				header, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				consensusState := &ibctm.ConsensusState{
					Timestamp:          header.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()),
					NextValidatorsHash: header.Header.NextValidatorsHash,
				}

				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID, tmHeader.GetHeight(), consensusState)
			},
			false,
		},
		{
			"invalid fork misbehaviour: identical headers", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviourHeader := s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
				clientMessage = &ibctm.Misbehaviour{
					Header1: misbehaviourHeader,
					Header2: misbehaviourHeader,
				}
			}, false,
		},
		{
			"invalid time misbehaviour: monotonically increasing time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				clientMessage = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+3, trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, false,
		},
		{
			"consensus state already exists, app hash mismatch",
			func() {
				header, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				consensusState := &ibctm.ConsensusState{
					Timestamp:          header.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot([]byte{}), // empty bytes
					NextValidatorsHash: header.Header.NextValidatorsHash,
				}

				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID, tmHeader.GetHeight(), consensusState)
			},
			true,
		},
		{
			"previous consensus state exists and header time is before previous consensus state time",
			func() {
				header, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				// offset header timestamp before previous consensus state timestamp
				header.Header.Time = header.GetTime().Add(-time.Hour)
			},
			true,
		},
		{
			"next consensus state exists and header time is after next consensus state time",
			func() {
				header, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				// commit block and update client, adding a new consensus state
				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				// increase timestamp of current header
				header.Header.Time = header.Header.Time.Add(time.Hour)
			},
			true,
		},
		{
			"valid fork misbehaviour returns true",
			func() {
				header1, err := path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				s.Require().NoError(err)

				// commit block and update client
				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				header2, err := path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				s.Require().NoError(err)

				// assign the same height, each header will have a different commit hash
				header1.Header.Height = header2.Header.Height

				clientMessage = &ibctm.Misbehaviour{
					Header1:  header1,
					Header2:  header2,
					ClientId: path.EndpointA.ClientID,
				}
			},
			true,
		},
		{
			"valid time misbehaviour: not monotonically increasing time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := s.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				s.Require().True(found)

				clientMessage = &ibctm.Misbehaviour{
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height+3, trustedHeight, s.chainB.CurrentHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.CurrentHeader.Height, trustedHeight, s.chainB.CurrentHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			// reset suite to create fresh application state
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			// ensure counterparty state is committed
			s.coordinator.CommitBlock(s.chainB)
			clientMessage, err = path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			foundMisbehaviour := clientState.CheckForMisbehaviour(
				s.chainA.GetContext(),
				s.chainA.App.AppCodec(),
				clientStore, // pass in clientID prefixed clientStore
				clientMessage,
			)

			if tc.expPass {
				s.Require().True(foundMisbehaviour)
			} else {
				s.Require().False(foundMisbehaviour)
			}
		})
	}
}

func (s *TendermintTestSuite) TestUpdateStateOnMisbehaviour() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			// reset suite to create fresh application state
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			clientState.UpdateStateOnMisbehaviour(s.chainA.GetContext(), s.chainA.App.AppCodec(), clientStore, nil)

			if tc.expPass {
				clientStateBz := clientStore.Get(host.ClientStateKey())
				s.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(s.chainA.Codec, clientStateBz)
				s.Require().Equal(frozenHeight, newClientState.(*ibctm.ClientState).FrozenHeight)
			}
		})
	}
}
