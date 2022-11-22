package types_test

import (
	"math"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/01-dymint/types"
	solomachinetypes "github.com/cosmos/ibc-go/v3/modules/light-clients/06-solomachine/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *DymintTestSuite) TestGetConsensusState() {
	var (
		height                  exported.Height
		path                    *ibctesting.Path
		dymintCounterpartyChain *ibctesting.TestChain
		endpointClientID        string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"consensus state not found", func() {
				// use height with no consensus state set
				height = height.(clienttypes.Height).Increment()
			}, false,
		},
		{
			"not a consensus state interface", func() {
				// marshal an empty client state and set as consensus state
				store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpointClientID)
				clientStateBz := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.MustMarshalClientState(&types.ClientState{})
				store.Set(host.ConsensusStateKey(height), clientStateBz)
			}, false,
		},
		{
			"invalid consensus state (solomachine)", func() {
				// marshal and set solomachine consensus state
				store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpointClientID)
				consensusStateBz := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.MustMarshalConsensusState(&solomachinetypes.ConsensusState{})
				store.Set(host.ConsensusStateKey(height), consensusStateBz)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			suite.coordinator.Setup(path)

			if suite.chainA.TestChainClient.GetSelfClientType() == exported.Dymint {
				dymintCounterpartyChain = suite.chainB
				endpointClientID = path.EndpointB.ClientID
			} else {
				// chainB must be Dymint
				dymintCounterpartyChain = suite.chainA
				endpointClientID = path.EndpointA.ClientID
			}

			clientState := dymintCounterpartyChain.GetClientState(endpointClientID)
			height = clientState.GetLatestHeight()

			tc.malleate() // change vars as necessary

			store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpointClientID)
			consensusState, err := types.GetConsensusState(store, dymintCounterpartyChain.Codec, height)

			if tc.expPass {
				suite.Require().NoError(err)
				expConsensusState, found := dymintCounterpartyChain.GetConsensusState(endpointClientID, height)
				suite.Require().True(found)
				suite.Require().Equal(expConsensusState, consensusState)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(consensusState)
			}
		})
	}
}

func (suite *DymintTestSuite) TestGetProcessedTime() {
	var (
		dymintCounterpartyChain *ibctesting.TestChain
		endpoint                *ibctesting.Endpoint
		expectedTime            time.Time
	)
	// setup
	path := ibctesting.NewPath(suite.chainA, suite.chainB)

	suite.coordinator.UpdateTime()

	if suite.chainB.TestChainClient.GetSelfClientType() == exported.Tendermint {
		// chainA must be Dymint
		dymintCounterpartyChain = suite.chainB
		endpoint = path.EndpointB
		// coordinator increments time before creating client
		expectedTime = dymintCounterpartyChain.TestChainClient.(*ibctesting.TestChainTendermint).CurrentHeader.Time.Add(ibctesting.TimeIncrement)
	} else {
		// chainB must be Dymint
		dymintCounterpartyChain = suite.chainA
		endpoint = path.EndpointA
		if dymintCounterpartyChain.TestChainClient.GetSelfClientType() == exported.Tendermint {
			expectedTime = dymintCounterpartyChain.TestChainClient.(*ibctesting.TestChainTendermint).CurrentHeader.Time.Add(ibctesting.TimeIncrement)
		} else {
			expectedTime = dymintCounterpartyChain.TestChainClient.(*ibctesting.TestChainDymint).CurrentHeader.Time.Add(ibctesting.TimeIncrement)
		}
	}

	// Verify ProcessedTime on CreateClient
	err := endpoint.CreateClient()
	suite.Require().NoError(err)

	clientState := dymintCounterpartyChain.GetClientState(endpoint.ClientID)
	height := clientState.GetLatestHeight()

	store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpoint.ClientID)
	actualTime, ok := types.GetProcessedTime(store, height)
	suite.Require().True(ok, "could not retrieve processed time for stored consensus state")
	suite.Require().Equal(uint64(expectedTime.UnixNano()), actualTime, "retrieved processed time is not expected value")

	suite.coordinator.UpdateTime()
	// coordinator increments time before updating client
	if dymintCounterpartyChain.TestChainClient.GetSelfClientType() == exported.Tendermint {
		expectedTime = dymintCounterpartyChain.TestChainClient.(*ibctesting.TestChainTendermint).CurrentHeader.Time.Add(ibctesting.TimeIncrement)
	} else {
		expectedTime = dymintCounterpartyChain.TestChainClient.(*ibctesting.TestChainDymint).CurrentHeader.Time.Add(ibctesting.TimeIncrement)
	}

	// Verify ProcessedTime on UpdateClient
	err = endpoint.UpdateClient()
	suite.Require().NoError(err)

	clientState = dymintCounterpartyChain.GetClientState(endpoint.ClientID)
	height = clientState.GetLatestHeight()

	store = dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpoint.ClientID)
	actualTime, ok = types.GetProcessedTime(store, height)
	suite.Require().True(ok, "could not retrieve processed time for stored consensus state")
	suite.Require().Equal(uint64(expectedTime.UnixNano()), actualTime, "retrieved processed time is not expected value")

	// try to get processed time for height that doesn't exist in store
	_, ok = types.GetProcessedTime(store, clienttypes.NewHeight(1, 1))
	suite.Require().False(ok, "retrieved processed time for a non-existent consensus state")
}

func (suite *DymintTestSuite) TestIterationKey() {
	testHeights := []exported.Height{
		clienttypes.NewHeight(0, 1),
		clienttypes.NewHeight(0, 1234),
		clienttypes.NewHeight(7890, 4321),
		clienttypes.NewHeight(math.MaxUint64, math.MaxUint64),
	}
	for _, h := range testHeights {
		k := types.IterationKey(h)
		retrievedHeight := types.GetHeightFromIterationKey(k)
		suite.Require().Equal(h, retrievedHeight, "retrieving height from iteration key failed")
	}
}

func (suite *DymintTestSuite) TestIterateConsensusStates() {
	var dymintCounterpartyChain *ibctesting.TestChain
	if suite.chainB.TestChainClient.GetSelfClientType() == exported.Tendermint {
		// chainA must be Dymint
		dymintCounterpartyChain = suite.chainB
	} else {
		// chainB must be Dymint
		dymintCounterpartyChain = suite.chainA
	}

	nextValsHash := []byte("nextVals")

	// Set iteration keys and consensus states
	types.SetIterationKey(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), clienttypes.NewHeight(0, 1))
	dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(dymintCounterpartyChain.GetContext(), "testClient", clienttypes.NewHeight(0, 1), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-1")), nextValsHash))
	types.SetIterationKey(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), clienttypes.NewHeight(4, 9))
	dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(dymintCounterpartyChain.GetContext(), "testClient", clienttypes.NewHeight(4, 9), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash4-9")), nextValsHash))
	types.SetIterationKey(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), clienttypes.NewHeight(0, 10))
	dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(dymintCounterpartyChain.GetContext(), "testClient", clienttypes.NewHeight(0, 10), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-10")), nextValsHash))
	types.SetIterationKey(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), clienttypes.NewHeight(0, 4))
	dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(dymintCounterpartyChain.GetContext(), "testClient", clienttypes.NewHeight(0, 4), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-4")), nextValsHash))
	types.SetIterationKey(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), clienttypes.NewHeight(40, 1))
	dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(dymintCounterpartyChain.GetContext(), "testClient", clienttypes.NewHeight(40, 1), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash40-1")), nextValsHash))

	var testArr []string
	cb := func(height exported.Height) bool {
		testArr = append(testArr, height.String())
		return false
	}

	types.IterateConsensusStateAscending(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), cb)
	expectedArr := []string{"0-1", "0-4", "0-10", "4-9", "40-1"}
	suite.Require().Equal(expectedArr, testArr)
}

func (suite *DymintTestSuite) TestGetNeighboringConsensusStates() {
	var dymintCounterpartyChain *ibctesting.TestChain
	if suite.chainB.TestChainClient.GetSelfClientType() == exported.Tendermint {
		// chainA must be Dymint
		dymintCounterpartyChain = suite.chainB
	} else {
		// chainB must be Dymint
		dymintCounterpartyChain = suite.chainA
	}

	nextValsHash := []byte("nextVals")
	cs01 := types.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash0-1")), nextValsHash)
	cs04 := types.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash0-4")), nextValsHash)
	cs49 := types.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash4-9")), nextValsHash)
	height01 := clienttypes.NewHeight(0, 1)
	height04 := clienttypes.NewHeight(0, 4)
	height49 := clienttypes.NewHeight(4, 9)

	// Set iteration keys and consensus states
	types.SetIterationKey(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), height01)
	dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(dymintCounterpartyChain.GetContext(), "testClient", height01, cs01)
	types.SetIterationKey(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), height04)
	dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(dymintCounterpartyChain.GetContext(), "testClient", height04, cs04)
	types.SetIterationKey(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), height49)
	dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(dymintCounterpartyChain.GetContext(), "testClient", height49, cs49)

	prevCs01, ok := types.GetPreviousConsensusState(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), dymintCounterpartyChain.Codec, height01)
	suite.Require().Nil(prevCs01, "consensus state exists before lowest consensus state")
	suite.Require().False(ok)
	prevCs49, ok := types.GetPreviousConsensusState(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), dymintCounterpartyChain.Codec, height49)
	suite.Require().Equal(cs04, prevCs49, "previous consensus state is not returned correctly")
	suite.Require().True(ok)

	nextCs01, ok := types.GetNextConsensusState(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), dymintCounterpartyChain.Codec, height01)
	suite.Require().Equal(cs04, nextCs01, "next consensus state not returned correctly")
	suite.Require().True(ok)
	nextCs49, ok := types.GetNextConsensusState(dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), "testClient"), dymintCounterpartyChain.Codec, height49)
	suite.Require().Nil(nextCs49, "next consensus state exists after highest consensus state")
	suite.Require().False(ok)
}
