package tendermint_test

import (
	"time"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmttypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *TendermintTestSuite) TestVerifyHeader() {
	var (
		path   *ibctesting.Path
		header *ibctm.Header
	)

	// Setup different validators and signers for testing different types of updates
	altPrivVal := cmttypes.NewMockPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	revisionHeight := int64(height.RevisionHeight)

	// create modified heights to use for test-cases
	altVal := cmttypes.NewValidator(altPubKey, 100)
	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{altVal})
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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				// passing the ProposedHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// commit some blocks so that the created Header now has a previous height as the BlockHeight
				suite.coordinator.CommitNBlocks(suite.chainB, 5)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			expPass: true,
		},
		{
			name: "successful verify header: header with future height and different validator set",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+5, trustedHeight, suite.chainB.ProposedHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners)
			},
			expPass: true,
		},
		{
			name: "successful verify header: header with next height and different validator set",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners)
			},
			expPass: true,
		},
		{
			name: "unsuccessful updates, passed in incorrect trusted validators for given consensus state",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				// Create bothValSet with both suite validator and altVal
				bothValSet := cmttypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time, bothValSet, bothValSet, bothValSet, bothSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header with next height: update header mismatches nextValSetHash",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				// this will err as altValSet.Hash() != consState.NextValidatorsHash
				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update with future height: too much change in validator set",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)
				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				header = suite.chainB.CreateTMClientHeader(chainIDRevision1, 3, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header height < consensus height",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				heightMinus1 := clienttypes.NewHeight(trustedHeight.RevisionNumber, trustedHeight.RevisionHeight-1)

				// Make new header at height less than latest client state
				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(heightMinus1.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
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
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time.Add(-time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header with incorrect header chain-id",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				header = suite.chainB.CreateTMClientHeader(chainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: trusting period has passed since last client timestamp",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight))
				suite.Require().NoError(err)

				header = suite.chainA.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+1, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				suite.chainB.ExpireClient(ibctesting.TrustingPeriod)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update for a previous revision",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				// passing the ProposedHeader.Height as the block height as it will become an update to previous revision once we upgrade the client
				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// increment the revision of the chain
				err = path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)
			},
			expPass: false,
		},
		{
			name: "successful update with identical header to a previous update",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				// passing the ProposedHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// update client so the header constructed becomes a duplicate
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			expPass: true,
		},

		{
			name: "unsuccessful update to a future revision",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID+"-1", suite.chainB.ProposedHeader.Height+5, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},

		{
			name: "unsuccessful update: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				// increment the revision of the chain
				err = path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)

				header = suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			// ensure counterparty state is committed
			suite.coordinator.CommitBlock(suite.chainB)
			trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			suite.Require().True(ok)
			header, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			suite.Require().NoError(err)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(suite.chainA.GetContext(), path.EndpointA.ClientID, header)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TendermintTestSuite) TestUpdateState() {
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
				suite.Require().True(ok)
				suite.Require().True(path.EndpointA.GetClientLatestHeight().(clienttypes.Height).LT(tmHeader.GetHeight()))
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)

				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().True(clientState.LatestHeight.EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				suite.Require().True(clientState.LatestHeight.EQ(consensusHeights[0]))
			}, true,
		},
		{
			"success with height earlier than latest height", func() {
				// commit a block so the pre-created ClientMessage
				// isn't used to update the client to a newer height
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)
				suite.Require().True(path.EndpointA.GetClientLatestHeight().(clienttypes.Height).GT(tmHeader.GetHeight()))

				prevClientState = path.EndpointA.GetClientState()
			},
			func() {
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().Equal(clientState, prevClientState) // fill in height, no change to client state
				suite.Require().True(clientState.LatestHeight.GT(consensusHeights[0]))
			}, true,
		},
		{
			"success with duplicate header", func() {
				// update client in advance
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// use the same header which just updated the client
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)
				clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				suite.Require().NoError(err)

				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)
				suite.Require().Equal(path.EndpointA.GetClientLatestHeight().(clienttypes.Height), tmHeader.GetHeight())

				prevClientState = path.EndpointA.GetClientState()
				prevConsensusState = path.EndpointA.GetConsensusState(tmHeader.GetHeight())
			},
			func() {
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().Equal(clientState, prevClientState)
				suite.Require().True(clientState.LatestHeight.EQ(consensusHeights[0]))

				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)
				suite.Require().Equal(path.EndpointA.GetConsensusState(tmHeader.GetHeight()), prevConsensusState)
			}, true,
		},
		{
			"success with pruned consensus state", func() {
				// this height will be expired and pruned
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				var ok bool
				pruneHeight, ok = path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				// Increment the time by a week
				suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

				// create the consensus state that can be used as trusted height for next update
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// Increment the time by another week, then update the client.
				// This will cause the first two consensus states to become expired.
				suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// ensure counterparty state is committed
				suite.coordinator.CommitBlock(suite.chainB)
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)
				clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				suite.Require().NoError(err)
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)

				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().True(clientState.LatestHeight.EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				suite.Require().True(clientState.LatestHeight.EQ(consensusHeights[0]))

				// ensure consensus state was pruned
				_, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				suite.Require().False(found)
			}, true,
		},
		{
			"success with pruned consensus state using duplicate header", func() {
				// this height will be expired and pruned
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				var ok bool
				pruneHeight, ok = path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				// assert that a consensus state exists at the prune height
				consensusState, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				suite.Require().True(found)
				suite.Require().NotNil(consensusState)

				// Increment the time by a week
				suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

				// create the consensus state that can be used as trusted height for next update
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// Increment the time by another week, then update the client.
				// This will cause the first two consensus states to become expired.
				suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// use the same header which just updated the client
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)
				clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				suite.Require().NoError(err)
			},
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)

				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().True(clientState.LatestHeight.EQ(tmHeader.GetHeight())) // new update, updated client state should have changed
				suite.Require().True(clientState.LatestHeight.EQ(consensusHeights[0]))

				// ensure consensus state was pruned
				_, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				suite.Require().False(found)
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
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			pruneHeight = clienttypes.ZeroHeight()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			// ensure counterparty state is committed
			suite.coordinator.CommitBlock(suite.chainB)
			trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			suite.Require().True(ok)
			clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			suite.Require().NoError(err)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
			suite.Require().True(found)

			tc.malleate()

			clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			if tc.expPass {
				consensusHeights = lightClientModule.UpdateState(suite.chainA.GetContext(), path.EndpointA.ClientID, clientMessage)

				header, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)

				expConsensusState := &ibctm.ConsensusState{
					Timestamp:          header.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()),
					NextValidatorsHash: header.Header.NextValidatorsHash,
				}

				bz := clientStore.Get(host.ConsensusStateKey(header.GetHeight()))
				updatedConsensusState := clienttypes.MustUnmarshalConsensusState(suite.chainA.App.AppCodec(), bz)

				suite.Require().Equal(expConsensusState, updatedConsensusState)

			} else {
				consensusHeights = lightClientModule.UpdateState(suite.chainA.GetContext(), path.EndpointA.ClientID, clientMessage)
				suite.Require().Empty(consensusHeights)

				consensusState, found := suite.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.GetClientConsensusState(suite.chainA.GetContext(), path.EndpointA.ClientID, clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight())))
				suite.Require().False(found)
				suite.Require().Nil(consensusState)
			}

			// perform custom checks
			tc.expResult()
		})
	}
}

func (suite *TendermintTestSuite) TestUpdateStateCheckTx() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupClients()

	createClientMessage := func() exported.ClientMessage {
		trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
		suite.Require().True(ok)
		header, err := path.EndpointB.Chain.IBCClientHeader(path.EndpointB.Chain.LatestCommittedHeader, trustedHeight)
		suite.Require().NoError(err)
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
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

	lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
	suite.Require().True(found)

	ctx = path.EndpointA.Chain.GetContext().WithIsCheckTx(true)
	lightClientModule.UpdateState(ctx, path.EndpointA.ClientID, createClientMessage())

	// Increment the time by another week, then update the client.
	// This will cause the first two consensus states to become expired.
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
	ctx = path.EndpointA.Chain.GetContext().WithIsCheckTx(true)
	lightClientModule.UpdateState(ctx, path.EndpointA.ClientID, createClientMessage())

	assertPrune := func(pruned bool) {
		// check consensus states and associated metadata
		consState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
		suite.Require().Equal(!pruned, ok)

		processTime, ok := ibctm.GetProcessedTime(clientStore, pruneHeight)
		suite.Require().Equal(!pruned, ok)

		processHeight, ok := ibctm.GetProcessedHeight(clientStore, pruneHeight)
		suite.Require().Equal(!pruned, ok)

		consKey := ibctm.GetIterationKey(clientStore, pruneHeight)

		if pruned {
			suite.Require().Nil(consState, "expired consensus state not pruned")
			suite.Require().Empty(processTime, "processed time metadata not pruned")
			suite.Require().Nil(processHeight, "processed height metadata not pruned")
			suite.Require().Nil(consKey, "iteration key not pruned")
		} else {
			suite.Require().NotNil(consState, "expired consensus state pruned")
			suite.Require().NotEqual(uint64(0), processTime, "processed time metadata pruned")
			suite.Require().NotNil(processHeight, "processed height metadata pruned")
			suite.Require().NotNil(consKey, "iteration key pruned")
		}
	}

	assertPrune(false)

	// simulation mode must prune to calculate gas correctly
	ctx = ctx.WithExecMode(sdk.ExecModeSimulate)
	lightClientModule.UpdateState(ctx, path.EndpointA.ClientID, createClientMessage())

	assertPrune(true)
}

func (suite *TendermintTestSuite) TestPruneConsensusState() {
	// create path and setup clients
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
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
	suite.Require().NoError(err)
	expiredHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
	suite.Require().True(ok)

	// expected values that must still remain in store after pruning
	expectedConsState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, expiredHeight)
	suite.Require().True(ok)
	ctx = path.EndpointA.Chain.GetContext()
	clientStore = path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)
	expectedProcessTime, ok := ibctm.GetProcessedTime(clientStore, expiredHeight)
	suite.Require().True(ok)
	expectedProcessHeight, ok := ibctm.GetProcessedHeight(clientStore, expiredHeight)
	suite.Require().True(ok)
	expectedConsKey := ibctm.GetIterationKey(clientStore, expiredHeight)
	suite.Require().NotNil(expectedConsKey)

	// Increment the time by a week
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

	// create the consensus state that can be used as trusted height for next update
	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	// Increment the time by another week, then update the client.
	// This will cause the first two consensus states to become expired.
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	ctx = path.EndpointA.Chain.GetContext()
	clientStore = path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

	// check that the first expired consensus state got deleted along with all associated metadata
	consState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
	suite.Require().Nil(consState, "expired consensus state not pruned")
	suite.Require().False(ok)
	// check processed time metadata is pruned
	processTime, ok := ibctm.GetProcessedTime(clientStore, pruneHeight)
	suite.Require().Equal(uint64(0), processTime, "processed time metadata not pruned")
	suite.Require().False(ok)
	processHeight, ok := ibctm.GetProcessedHeight(clientStore, pruneHeight)
	suite.Require().Nil(processHeight, "processed height metadata not pruned")
	suite.Require().False(ok)

	// check iteration key metadata is pruned
	consKey := ibctm.GetIterationKey(clientStore, pruneHeight)
	suite.Require().Nil(consKey, "iteration key not pruned")

	// check that second expired consensus state doesn't get deleted
	// this ensures that there is a cap on gas cost of UpdateClient
	consState, ok = path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, expiredHeight)
	suite.Require().Equal(expectedConsState, consState, "consensus state incorrectly pruned")
	suite.Require().True(ok)
	// check processed time metadata is not pruned
	processTime, ok = ibctm.GetProcessedTime(clientStore, expiredHeight)
	suite.Require().Equal(expectedProcessTime, processTime, "processed time metadata incorrectly pruned")
	suite.Require().True(ok)

	// check processed height metadata is not pruned
	processHeight, ok = ibctm.GetProcessedHeight(clientStore, expiredHeight)
	suite.Require().Equal(expectedProcessHeight, processHeight, "processed height metadata incorrectly pruned")
	suite.Require().True(ok)

	// check iteration key metadata is not pruned
	consKey = ibctm.GetIterationKey(clientStore, expiredHeight)
	suite.Require().Equal(expectedConsKey, consKey, "iteration key incorrectly pruned")
}

func (suite *TendermintTestSuite) TestCheckForMisbehaviour() {
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
				suite.Require().True(ok)

				consensusState := &ibctm.ConsensusState{
					Timestamp:          header.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()),
					NextValidatorsHash: header.Header.NextValidatorsHash,
				}

				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), path.EndpointA.ClientID, tmHeader.GetHeight(), consensusState)
			},
			false,
		},
		{
			"invalid fork misbehaviour: identical headers", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				misbehaviourHeader := suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
				clientMessage = &ibctm.Misbehaviour{
					Header1: misbehaviourHeader,
					Header2: misbehaviourHeader,
				}
			}, false,
		},
		{
			"invalid time misbehaviour: monotonically increasing time", func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				clientMessage = &ibctm.Misbehaviour{
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+3, trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, false,
		},
		{
			"consensus state already exists, app hash mismatch",
			func() {
				header, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)

				consensusState := &ibctm.ConsensusState{
					Timestamp:          header.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot([]byte{}), // empty bytes
					NextValidatorsHash: header.Header.NextValidatorsHash,
				}

				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), path.EndpointA.ClientID, tmHeader.GetHeight(), consensusState)
			},
			true,
		},
		{
			"previous consensus state exists and header time is before previous consensus state time",
			func() {
				header, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)

				// offset header timestamp before previous consensus state timestamp
				header.Header.Time = header.GetTime().Add(-time.Hour)
			},
			true,
		},
		{
			"next consensus state exists and header time is after next consensus state time",
			func() {
				header, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)

				// commit block and update client, adding a new consensus state
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// increase timestamp of current header
				header.Header.Time = header.Header.Time.Add(time.Hour)
			},
			true,
		},
		{
			"valid fork misbehaviour returns true",
			func() {
				trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)
				header1, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				suite.Require().NoError(err)

				// commit block and update client
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				trustedHeight, ok = path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				suite.Require().True(ok)
				header2, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
				suite.Require().NoError(err)

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
				suite.Require().True(ok)

				trustedVals, err := suite.chainB.GetTrustedValidators(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().NoError(err)

				clientMessage = &ibctm.Misbehaviour{
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height+3, trustedHeight, suite.chainB.ProposedHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.ProposedHeader.Height, trustedHeight, suite.chainB.ProposedHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			// ensure counterparty state is committed
			suite.coordinator.CommitBlock(suite.chainB)
			trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			suite.Require().True(ok)
			clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			suite.Require().NoError(err)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
			suite.Require().True(found)

			tc.malleate()

			foundMisbehaviour := lightClientModule.CheckForMisbehaviour(
				suite.chainA.GetContext(),
				path.EndpointA.ClientID,
				clientMessage,
			)

			if tc.expPass {
				suite.Require().True(foundMisbehaviour)
			} else {
				suite.Require().False(foundMisbehaviour)
			}
		})
	}
}

func (suite *TendermintTestSuite) TestUpdateStateOnMisbehaviour() {
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

		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
			suite.Require().True(found)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			lightClientModule.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), path.EndpointA.ClientID, nil)

			if tc.expPass {
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				suite.Require().Equal(frozenHeight, newClientState.(*ibctm.ClientState).FrozenHeight)
			}
		})
	}
}
