package types_test

import (
	"fmt"
	"time"

	tmtypes "github.com/tendermint/tendermint/types"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	types "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
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
	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})
	altSigners := []tmtypes.PrivValidator{altPrivVal}

	testCases := []struct {
		name      string
		setup     func(*TendermintTestSuite)
		expPass   bool
		expFrozen bool
	}{
		{
			name: "successful update with next height and same validator set",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass:   true,
			expFrozen: false,
		},
		{
			name: "successful update with future height and different validator set",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				bothValSet, bothSigners := getBothSigners(suite, altVal, altPrivVal)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus5.RevisionHeight), height, suite.headerTime, bothValSet, suite.valSet, bothSigners)
				currentTime = suite.now
			},
			expPass:   true,
			expFrozen: false,
		},
		{
			name: "successful update with next height and different validator set",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				bothValSet, bothSigners := getBothSigners(suite, altVal, altPrivVal)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), bothValSet.Hash())
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, bothValSet, bothValSet, bothSigners)
				currentTime = suite.now
			},
			expPass:   true,
			expFrozen: false,
		},
		{
			name: "successful update for a previous height",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				consStateHeight = heightMinus3
				bothValSet, bothSigners := getBothSigners(suite, altVal, altPrivVal)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightMinus1.RevisionHeight), heightMinus3, suite.headerTime, bothValSet, suite.valSet, bothSigners)
				currentTime = suite.now
			},
			expPass:   true,
			expFrozen: false,
		},
		{
			name: "successful update for a previous revision",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainIDRevision1, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				consStateHeight = heightMinus3
				bothValSet, bothSigners := getBothSigners(suite, altVal, altPrivVal)
				newHeader = suite.chainA.CreateTMClientHeader(chainIDRevision0, int64(height.RevisionHeight), heightMinus3, suite.headerTime, bothValSet, suite.valSet, bothSigners)
				currentTime = suite.now
			},
			expPass:   true,
			expFrozen: false,
		},
		{
			name: "successful update with identical header to a previous update",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, heightPlus1, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
				ctx := suite.chainA.GetContext().WithBlockTime(currentTime)
				// Store the header's consensus state in client store before UpdateClient call
				suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(ctx, clientID, heightPlus1, newHeader.ConsensusState())
			},
			expPass:   true,
			expFrozen: false,
		},
		{
			name: "misbehaviour detection: header conflicts with existing consensus state",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, heightPlus1, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
				ctx := suite.chainA.GetContext().WithBlockTime(currentTime)
				// Change the consensus state of header and store in client store to create a conflict
				conflictConsState := newHeader.ConsensusState()
				conflictConsState.Root = commitmenttypes.NewMerkleRoot([]byte("conflicting apphash"))
				suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(ctx, clientID, heightPlus1, conflictConsState)
			},
			expPass:   true,
			expFrozen: true,
		},
		{
			name: "unsuccessful update with incorrect header chain-id",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader("ethermint", int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful update to a future revision",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainIDRevision0, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainIDRevision1, 1, height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful update: header height revision and trusted height revision mismatch",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainIDRevision1, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.NewHeight(1, 1), commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainIDRevision1, 3, height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful update with next height: update header mismatches nextValSetHash",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				bothValSet, bothSigners := getBothSigners(suite, altVal, altPrivVal)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, bothValSet, suite.valSet, bothSigners)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful update with next height: update header mismatches different nextValSetHash",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				bothValSet, _ := getBothSigners(suite, altVal, altPrivVal)
				signers := getSuiteSigners(suite)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), bothValSet.Hash())
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, bothValSet, signers)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful update with future height: too much change in validator set",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus5.RevisionHeight), height, suite.headerTime, altValSet, suite.valSet, altSigners)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful updates, passed in incorrect trusted validators for given consensus state",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				bothValSet, bothSigners := getBothSigners(suite, altVal, altPrivVal)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus5.RevisionHeight), height, suite.headerTime, bothValSet, bothValSet, bothSigners)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful update: trusting period has passed since last client timestamp",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				// make current time pass trusting period from last timestamp on clientstate
				currentTime = suite.now.Add(trustingPeriod)
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful update: header timestamp is past current timestamp",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.now.Add(time.Minute), suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "unsuccessful update: header timestamp is not past last client timestamp",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.clientTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "header basic validation failed",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightPlus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				// cause new header to fail validatebasic by changing commit height to mismatch header height
				newHeader.SignedHeader.Commit.Height = revisionHeight - 1
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
		{
			name: "header height < consensus height",
			setup: func(suite *TendermintTestSuite) {
				clientState = types.NewClientState(chainID, types.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.NewHeight(height.RevisionNumber, heightPlus5.RevisionHeight), commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
				consensusState = types.NewConsensusState(suite.clientTime, commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()), suite.valsHash)
				signers := getSuiteSigners(suite)
				// Make new header at height less than latest client state
				newHeader = suite.chainA.CreateTMClientHeader(chainID, int64(heightMinus1.RevisionHeight), height, suite.headerTime, suite.valSet, suite.valSet, signers)
				currentTime = suite.now
			},
			expPass:   false,
			expFrozen: false,
		},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case: %s", tc.name), func() {
			// reset suite to create fresh application state
			suite.SetupTest()

			consStateHeight = height // must be explicitly changed
			// setup test
			tc.setup(suite)

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

				if tc.expFrozen {
					suite.Require().True(newClientState.IsFrozen(), "client did not freeze after conflicting header was submitted to UpdateClient")
					suite.Require().Equal(newClientState.GetFrozenHeight(), newHeader.GetHeight(), "client frozen at wrong height")
				}

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
		})
	}
}
