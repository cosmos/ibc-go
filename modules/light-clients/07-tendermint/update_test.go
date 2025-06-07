package tendermint_test

import (
	"errors"
	"time"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmttypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *TendermintTestSuite) TestVerifyHeader() {
	var (
		path   *ibctesting.Path
		header *ibctm.Header
	)

	// Setup different validators and signers for testing different types of updates
	altPrivVal := cmttypes.NewMockPV()
	altPubKey, err := altPrivVal.GetPubKey()
	s.Require().NoError(err)

	revisionHeight := int64(height.RevisionHeight)

	// create modified heights to use for test-cases
	altVal := cmttypes.NewValidator(altPubKey, 100)
	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{altVal})
	altSigners := getAltSigners(altVal, altPrivVal)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			name:     "success",
			malleate: func() {},
			expErr:   nil,
		},
		{
			name: "successful verify header for header with a previous height",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				// passing the ProposedHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)

				// commit some blocks so that the created Header now has a previous height as the BlockHeight
				s.coordinator.CommitNBlocks(s.chainB, 5)

				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)
			},
			expErr: nil,
		},
		{
			name: "successful verify header: header with future height and different validator set",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(s.chainB.Vals.Validators, altVal))
				bothSigners := s.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+5, trustedHeight, s.chainB.ProposedHeader.Time, bothValSet, s.chainB.NextVals, trustedVals, bothSigners)
			},
			expErr: nil,
		},
		{
			name: "successful verify header: header with next height and different validator set",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(s.chainB.Vals.Validators, altVal))
				bothSigners := s.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, bothValSet, s.chainB.NextVals, trustedVals, bothSigners)
			},
			expErr: nil,
		},
		{
			name: "unsuccessful updates, passed in incorrect trusted validators for given consensus state",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(s.chainB.Vals.Validators, altVal))
				bothSigners := s.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time, bothValSet, bothValSet, bothValSet, bothSigners)
			},
			expErr: errors.New("invalid validator set"),
		},
		{
			name: "unsuccessful verify header with next height: update header mismatches nextValSetHash",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				// this will err as altValSet.Hash() != consState.NextValidatorsHash
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expErr: errors.New("failed to verify header"),
		},
		{
			name: "unsuccessful update with future height: too much change in validator set",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expErr: errors.New("failed to verify header: cant trust new val set"),
		},
		{
			name: "unsuccessful verify header: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)
				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				header = s.chainB.CreateTMClientHeader(chainIDRevision1, 3, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expErr: errors.New("invalid client header"),
		},
		{
			name: "unsuccessful verify header: header height < consensus height",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				heightMinus1 := clienttypes.NewHeight(trustedHeight.RevisionNumber, trustedHeight.RevisionHeight-1)

				// Make new header at height less than latest client state
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(heightMinus1.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expErr: errors.New("invalid client header"),
		},
		{
			name: "unsuccessful verify header: header basic validation failed",
			malleate: func() {
				// cause header to fail validatebasic by changing commit height to mismatch header height
				header.Commit.Height = revisionHeight - 1
			},
			expErr: errors.New("header and commit height mismatch"),
		},
		{
			name: "unsuccessful verify header: header timestamp is not past last client timestamp",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time.Add(-time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expErr: errors.New("failed to verify header"),
		},
		{
			name: "unsuccessful verify header: header with incorrect header chain-id",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				header = s.chainB.CreateTMClientHeader(chainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expErr: errors.New("header height revision 0 does not match trusted header revision 1"),
		},
		{
			name: "unsuccessful update: trusting period has passed since last client timestamp",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight]
				s.Require().True(ok)

				header = s.chainA.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+1, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)

				s.chainB.ExpireClient(ibctesting.TrustingPeriod)
			},
			expErr: errors.New("failed to verify header"),
		},
		{
			name: "unsuccessful update for a previous revision",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				// passing the ProposedHeader.Height as the block height as it will become an update to previous revision once we upgrade the client
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)

				// increment the revision of the chain
				err = path.EndpointB.UpgradeChain()
				s.Require().NoError(err)
			},
			expErr: errors.New("failed to verify header"),
		},
		{
			name: "successful update with identical header to a previous update",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				// passing the ProposedHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)

				// update client so the header constructed becomes a duplicate
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)
			},
			expErr: nil,
		},

		{
			name: "unsuccessful update to a future revision",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID+"-1", s.chainB.ProposedHeader.Height+5, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expErr: errors.New("failed to verify header"),
		},

		{
			name: "unsuccessful update: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				// increment the revision of the chain
				err = path.EndpointB.UpgradeChain()
				s.Require().NoError(err)

				header = s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
			},
			expErr: errors.New("header height revision 2 does not match trusted header revision 1"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			// ensure counterparty state is committed
			s.coordinator.CommitBlock(s.chainB)
			trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			s.Require().True(ok)
			header, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			s.Require().NoError(err)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(s.chainA.GetContext(), path.EndpointA.ClientID, header)

			if tc.expErr == nil {
				s.Require().NoError(err, tc.name)
			} else {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}

func (s *TendermintTestSuite) TestUpdateState() {
	var (
		path               *ibctesting.Path
		clientMessage      exported.ClientMessage
		clientStore        storetypes.KVStore
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
				s.Require().True(path.EndpointA.GetClientLatestHeight().(clienttypes.Height).LT(tmHeader.GetHeight()))
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				s.Require().True(clientState.LatestHeight.EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				s.Require().True(clientState.LatestHeight.EQ(consensusHeights[0]))
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
				s.Require().True(path.EndpointA.GetClientLatestHeight().(clienttypes.Height).GT(tmHeader.GetHeight()))

				prevClientState = path.EndpointA.GetClientState()
			},
			func() {
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				s.Require().Equal(clientState, prevClientState) // fill in height, no change to client state
				s.Require().True(clientState.LatestHeight.GT(consensusHeights[0]))
			}, true,
		},
		{
			"success with duplicate header", func() {
				// update client in advance
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				// use the same header which just updated the client
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)
				clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				s.Require().NoError(err)

				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)
				s.Require().Equal(path.EndpointA.GetClientLatestHeight().(clienttypes.Height), tmHeader.GetHeight())

				prevClientState = path.EndpointA.GetClientState()
				prevConsensusState = path.EndpointA.GetConsensusState(tmHeader.GetHeight())
			},
			func() {
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				s.Require().Equal(clientState, prevClientState)
				s.Require().True(clientState.LatestHeight.EQ(consensusHeights[0]))

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
				var ok bool
				pruneHeight, ok = path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)
				clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				s.Require().NoError(err)
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				s.Require().True(clientState.LatestHeight.EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				s.Require().True(clientState.LatestHeight.EQ(consensusHeights[0]))

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
				var ok bool
				pruneHeight, ok = path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)
				clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				s.Require().NoError(err)
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				s.Require().True(clientState.LatestHeight.EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				s.Require().True(clientState.LatestHeight.EQ(consensusHeights[0]))

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
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			pruneHeight = clienttypes.ZeroHeight()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			// ensure counterparty state is committed
			s.coordinator.CommitBlock(s.chainB)
			trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			s.Require().True(ok)
			clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			s.Require().NoError(err)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			clientStore = s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			if tc.expPass {
				consensusHeights = lightClientModule.UpdateState(s.chainA.GetContext(), path.EndpointA.ClientID, clientMessage)

				header, ok := clientMessage.(*ibctm.Header)
				s.Require().True(ok)

				expConsensusState := &ibctm.ConsensusState{
					Timestamp:          header.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()),
					NextValidatorsHash: header.Header.NextValidatorsHash,
				}

				bz := clientStore.Get(host.ConsensusStateKey(header.GetHeight()))
				updatedConsensusState := clienttypes.MustUnmarshalConsensusState(s.chainA.App.AppCodec(), bz)

				s.Require().Equal(expConsensusState, updatedConsensusState)
			} else {
				consensusHeights = lightClientModule.UpdateState(s.chainA.GetContext(), path.EndpointA.ClientID, clientMessage)
				s.Require().Empty(consensusHeights)

				consensusState, found := s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.GetClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID, clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight())))
				s.Require().False(found)
				s.Require().Nil(consensusState)
			}

			// perform custom checks
			tc.expResult()
		})
	}
}

func (s *TendermintTestSuite) TestUpdateStateCheckTx() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	createClientMessage := func() exported.ClientMessage {
		trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
		s.Require().True(ok)
		header, err := path.EndpointB.Chain.IBCClientHeader(path.EndpointB.Chain.LatestCommittedHeader, trustedHeight)
		s.Require().NoError(err)
		return header
	}

	// get the first height as it will be pruned first.
	var pruneHeight exported.Height
	getFirstHeightCb := func(height exported.Height) bool {
		pruneHeight = height
		return true
	}
	ctx := path.EndpointA.Chain.GetContext()
	clientStore := path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)
	ibctm.IterateConsensusStateAscending(clientStore, getFirstHeightCb)

	// Increment the time by a week
	s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

	lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
	s.Require().NoError(err)

	ctx = path.EndpointA.Chain.GetContext().WithIsCheckTx(true)
	lightClientModule.UpdateState(ctx, path.EndpointA.ClientID, createClientMessage())

	// Increment the time by another week, then update the client.
	// This will cause the first two consensus states to become expired.
	s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
	ctx = path.EndpointA.Chain.GetContext().WithIsCheckTx(true)
	lightClientModule.UpdateState(ctx, path.EndpointA.ClientID, createClientMessage())

	assertPrune := func(pruned bool) {
		// check consensus states and associated metadata
		consState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
		s.Require().Equal(!pruned, ok)

		processTime, ok := ibctm.GetProcessedTime(clientStore, pruneHeight)
		s.Require().Equal(!pruned, ok)

		processHeight, ok := ibctm.GetProcessedHeight(clientStore, pruneHeight)
		s.Require().Equal(!pruned, ok)

		consKey := ibctm.GetIterationKey(clientStore, pruneHeight)

		if pruned {
			s.Require().Nil(consState, "expired consensus state not pruned")
			s.Require().Empty(processTime, "processed time metadata not pruned")
			s.Require().Nil(processHeight, "processed height metadata not pruned")
			s.Require().Nil(consKey, "iteration key not pruned")
		} else {
			s.Require().NotNil(consState, "expired consensus state pruned")
			s.Require().NotEqual(uint64(0), processTime, "processed time metadata pruned")
			s.Require().NotNil(processHeight, "processed height metadata pruned")
			s.Require().NotNil(consKey, "iteration key pruned")
		}
	}

	assertPrune(false)

	// simulation mode must prune to calculate gas correctly
	ctx = ctx.WithExecMode(sdk.ExecModeSimulate)
	lightClientModule.UpdateState(ctx, path.EndpointA.ClientID, createClientMessage())

	assertPrune(true)
}

func (s *TendermintTestSuite) TestPruneConsensusState() {
	// create path and setup clients
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

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
	expiredHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
	s.Require().True(ok)

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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				misbehaviourHeader := s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers)
				clientMessage = &ibctm.Misbehaviour{
					Header1: misbehaviourHeader,
					Header2: misbehaviourHeader,
				}
			}, false,
		},
		{
			"invalid time misbehaviour: monotonically increasing time", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				clientMessage = &ibctm.Misbehaviour{
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+3, trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)
				header1, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				s.Require().NoError(err)

				// commit block and update client
				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				trustedHeight, ok = path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)
				header2, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)

				trustedVals, ok := s.chainB.TrustedValidators[trustedHeight.RevisionHeight+1]
				s.Require().True(ok)

				clientMessage = &ibctm.Misbehaviour{
					Header2: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height+3, trustedHeight, s.chainB.ProposedHeader.Time.Add(time.Minute), s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
					Header1: s.chainB.CreateTMClientHeader(s.chainB.ChainID, s.chainB.ProposedHeader.Height, trustedHeight, s.chainB.ProposedHeader.Time, s.chainB.Vals, s.chainB.NextVals, trustedVals, s.chainB.Signers),
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// reset suite to create fresh application state
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			// ensure counterparty state is committed
			s.coordinator.CommitBlock(s.chainB)
			trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			s.Require().True(ok)
			clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			s.Require().NoError(err)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			foundMisbehaviour := lightClientModule.CheckForMisbehaviour(
				s.chainA.GetContext(),
				path.EndpointA.ClientID,
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
		s.Run(tc.name, func() {
			// reset suite to create fresh application state
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			err := path.EndpointA.CreateClient()
			s.Require().NoError(err)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			lightClientModule.UpdateStateOnMisbehaviour(s.chainA.GetContext(), path.EndpointA.ClientID, nil)

			if tc.expPass {
				clientStateBz := clientStore.Get(host.ClientStateKey())
				s.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(s.chainA.Codec, clientStateBz)
				s.Require().Equal(frozenHeight, newClientState.(*ibctm.ClientState).FrozenHeight)
			}
		})
	}
}
