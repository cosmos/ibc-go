package types_test

import (
	"time"

	tmtypes "github.com/tendermint/tendermint/types"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	types "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
	ibctestingmock "github.com/cosmos/ibc-go/testing/mock"
)

func (suite *TendermintTestSuite) TestCheckHeaderAndUpdateState() {
	var (
		clientState     *types.ClientState
		consensusState  *types.ConsensusState
		consStateHeight clienttypes.Height
		newHeader       *types.Header
		currentTime     time.Time
	)

	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	revisionHeight := int64(height.RevisionHeight)

	// create modified heights to use for test-cases
	heightPlus1 := clienttypes.NewHeight(height.RevisionNumber, height.RevisionHeight+1)
	heightMinus1 := clienttypes.NewHeight(height.RevisionNumber, height.RevisionHeight-1)
	heightMinus3 := clienttypes.NewHeight(height.RevisionNumber, height.RevisionHeight-3)
	heightPlus5 := clienttypes.NewHeight(height.RevisionNumber, height.RevisionHeight+5)

	altVal := tmtypes.NewValidator(altPubKey, revisionHeight)

	// Create bothValSet with both suite validator and altVal. Would be valid update
	bothValSet := tmtypes.NewValidatorSet(append(suite.valSet.Validators, altVal))
	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})

	signers := []tmtypes.PrivValidator{suite.privVal}

	// Create signer array and ensure it is in same order as bothValSet
	_, suiteVal := suite.valSet.GetByIndex(0)
	bothSigners := ibctesting.CreateSortedSignerArray(altPrivVal, suite.privVal, altVal, suiteVal)

	altSigners := []tmtypes.PrivValidator{altPrivVal}

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			name: "successful update with next height and same validator set",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass: true,
		},
		{
			name: "successful update with future height and different validator set",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus5.RevisionHeight), height, suite.headerTime, bothValSet, suite.valSet, bothSigners)
				currentTime = suite.now
			},
			expPass: true,
		},
		{
			name: "successful update with next height and different validator set",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), bothValSet.Hash())
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, bothValSet, bothValSet, bothSigners)
				currentTime = suite.now
			},
			expPass: true,
		},
		{
			name: "successful update for a previous height",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				consStateHeight = heightMinus3
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightMinus1.RevisionHeight), heightMinus3, suite.headerTime, bothValSet, suite.valSet, bothSigners)
				currentTime = suite.now
			},
			expPass: true,
		},
		{
			name: "successful update for a previous revision",
			setup: func() {
				clientState = types.NewClientState(chainIDRevision1, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainIDRevision0, int64(height.RevisionHeight), heightMinus3, suite.headerTime, bothValSet, suite.valSet, bothSigners)
				currentTime = suite.now
			},
			expPass: true,
		},
		{
			name: "unsuccessful update with incorrect header chain-id",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader("ethermint", int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "unsuccessful update to a future revision",
			setup: func() {
				clientState = types.NewClientState(chainIDRevision0, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainIDRevision1, 1, height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: header height revision and trusted height revision mismatch",
			setup: func() {
				clientState = types.NewClientState(chainIDRevision1, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.NewHeight(1, 1), commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainIDRevision1, 3, height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "unsuccessful update with next height: update header mismatches nextValSetHash",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, bothValSet, suite.valSet, bothSigners)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "unsuccessful update with next height: update header mismatches different nextValSetHash",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), bothValSet.Hash())
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, bothValSet, signers)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "unsuccessful update with future height: too much change in validator set",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus5.RevisionHeight), height, suite.headerTime, altValSet, suite.valSet, altSigners)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "unsuccessful updates, passed in incorrect trusted validators for given consensus state",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus5.RevisionHeight), height, suite.headerTime, bothValSet, bothValSet, bothSigners)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: trusting period has passed since last client timestamp",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				// make current time pass trusting period from last timestamp on clientstate
				currentTime = suite.now.Add(trustingPeriod)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: header timestamp is past current timestamp",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.now.Add(time.Minute), suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: header timestamp is not past last client timestamp",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.clientTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "header basic validation failed",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				// cause new header to fail validatebasic by changing commit height to mismatch header height
				newHeader.SignedHeader.Commit.Height = revisionHeight - 1
				currentTime = suite.now
			},
			expPass: false,
		},
		{
			name: "header height < consensus height",
			setup: func() {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.NewHeight(height.RevisionNumber, heightPlus5.RevisionHeight), commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				// Make new header at height less than latest client state
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightMinus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass: false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		consStateHeight = height // must be explicitly changed
		// setup test
		tc.setup()

		// Set current timestamp in context
		ctx := suite.chainA.GetContext().WithBlockTime(currentTime)

		// Set trusted consensus state in client store
		suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(ctx, clientID, consStateHeight, consensusState)

		height := newHeader.GetHeight()
		expectedConsensus := &types.ConsensusState{
			Timestamp:          newHeader.GetTime(),
			Root:               commitmenttypes.NewMerkleRoot(newHeader.Header.GetAppHash()),
			NextValidatorsHash: newHeader.Header.NextValidatorsHash,
		}

		newClientState, consensusState, err := clientState.CheckHeaderAndUpdateState(
			ctx,
			suite.cdc,
			suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID), // pass in clientID prefixed clientStore
			newHeader,
		)

		if tc.expPass {
			suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)

			// Determine if clientState should be updated or not
			// TODO: check the entire Height struct once GetLatestHeight returns clienttypes.Height
			if height.GT(clientState.LatestHeight) {
				// Header Height is greater than clientState latest Height, clientState should be updated with header.GetHeight()
				suite.Require().Equal(height, newClientState.GetLatestHeight(), "clientstate height did not update")
			} else {
				// Update will add past consensus state, clientState should not be updated at all
				suite.Require().Equal(clientState.LatestHeight, newClientState.GetLatestHeight(), "client state height updated for past header")
			}

			suite.Require().Equal(expectedConsensus, consensusState, "valid test case %d failed: %s", i, tc.name)
		} else {
			suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			suite.Require().Nil(newClientState, "invalid test case %d passed: %s", i, tc.name)
			suite.Require().Nil(consensusState, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func (suite *TendermintTestSuite) TestPruneConsensusState() {
	ctx := suite.chainA.GetContext().WithBlockTime(suite.now)

	// create 2 consensus states with the clientTime along with the metadata that would be stored by UpdateClient
	consensusState := types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(ctx, clientID, height, consensusState)
	types.SetProcessedTime(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(ctx, clientID), height, uint64(ctx.BlockTime().UnixNano()))
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(ctx, clientID), height)
	nextHeight := height.Increment()
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(ctx, clientID, nextHeight, consensusState)
	types.SetProcessedTime(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(ctx, clientID), nextHeight, uint64(ctx.BlockTime().UnixNano()))
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(ctx, clientID), nextHeight)

	// Set consensus state to a timestamp a week from now so that it is valid for the next consensus state which will be two weeks from now.
	trustedCtx := suite.chainA.GetContext().WithBlockTime(suite.now.Add(7 * 24 * time.Hour))
	trustedHeight := nextHeight.Increment().(clienttypes.Height)
	trustedConsensusState := types.NewConsensusState(suite.clientTime.Add(7*24*time.Hour), commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(trustedCtx, clientID, trustedHeight, trustedConsensusState)
	types.SetProcessedTime(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(trustedCtx, clientID), trustedHeight, uint64(trustedCtx.BlockTime().UnixNano()))
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(trustedCtx, clientID), trustedHeight)

	// Set current time to be two weeks from suite.now. This will cause the first two consensus state to become expired
	// Create the header that will be submitted to `CheckHeaderAndUpdateState`
	currentHeight := trustedHeight.Increment().(clienttypes.Height)
	clientState := types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, trustedHeight, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
	currentCtx := suite.chainA.GetContext().WithBlockTime(suite.now.Add(2 * 7 * 24 * time.Hour))
	newHeader := suite.chainA.CreateTMClientHeader(chainID, int64(currentHeight.RevisionHeight), trustedHeight, currentCtx.BlockTime(), suite.valSet, suite.valSet, []tmtypes.PrivValidator{suite.privVal})

	// CheckHeaderAndUpdateState must prune the oldest expired consensus state which is at height: `height`
	_, _, err := clientState.CheckHeaderAndUpdateState(
		currentCtx,
		suite.cdc,
		suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(currentCtx, clientID),
		newHeader,
	)
	suite.Require().NoError(err)

	// check that the first expired consensus state got deleted along with all associated metadata
	consState, err := types.GetConsensusState(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(currentCtx, clientID), suite.cdc, height)
	suite.Require().Nil(consState, "expired consensus state not pruned")
	suite.Require().Error(err, "getting deleted consensus state did not return error")
	// check processed time metadata is pruned
	processTime, ok := types.GetProcessedTime(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(currentCtx, clientID), height)
	suite.Require().Equal(uint64(0), processTime, "processed time metadata not pruned")
	suite.Require().False(ok)
	// check iteration key metadata is pruned
	consKey := types.GetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(currentCtx, clientID), height)
	suite.Require().Nil(consKey, "iteration key not pruned")

	// check that second expired consensus state doesn't get deleted
	// this ensures that there is a cap on gas cost of UpdateClient
	consState, err = types.GetConsensusState(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(currentCtx, clientID), suite.cdc, nextHeight)
	suite.Require().Equal(consensusState, consState, "consensus state is unexpectedly pruned")
	suite.Require().NoError(err)
	// check processed time metadata is not pruned
	processTime, ok = types.GetProcessedTime(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(currentCtx, clientID), nextHeight)
	suite.Require().Equal(uint64(ctx.BlockTime().UnixNano()), processTime, "processed time metadata is incorrect")
	suite.Require().True(ok)
	// check iteration key metadata is not pruned
	consKey = types.GetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(currentCtx, clientID), nextHeight)
	suite.Require().Equal(host.ConsensusStateKey(nextHeight), consKey, "iteration key does not store consensus state key correctly")
}
