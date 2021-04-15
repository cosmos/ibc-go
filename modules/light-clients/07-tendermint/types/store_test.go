package types_test

import (
	"time"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	solomachinetypes "github.com/cosmos/ibc-go/modules/light-clients/06-solomachine/types"
	"github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *TendermintTestSuite) TestGetConsensusState() {
	var (
		height  exported.Height
		clientA string
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
				store := suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), clientA)
				clientStateBz := suite.chainA.App.IBCKeeper.ClientKeeper.MustMarshalClientState(&types.ClientState{})
				store.Set(host.ConsensusStateKey(height), clientStateBz)
			}, false,
		},
		{
			"invalid consensus state (solomachine)", func() {
				// marshal and set solomachine consensus state
				store := suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), clientA)
				consensusStateBz := suite.chainA.App.IBCKeeper.ClientKeeper.MustMarshalConsensusState(&solomachinetypes.ConsensusState{})
				store.Set(host.ConsensusStateKey(height), consensusStateBz)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			clientA, _, _, _, _, _ = suite.coordinator.Setup(suite.chainA, suite.chainB, channeltypes.UNORDERED)
			clientState := suite.chainA.GetClientState(clientA)
			height = clientState.GetLatestHeight()

			tc.malleate() // change vars as necessary

			store := suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), clientA)
			consensusState, err := types.GetConsensusState(store, suite.chainA.Codec, height)

			if tc.expPass {
				suite.Require().NoError(err)
				expConsensusState, found := suite.chainA.GetConsensusState(clientA, height)
				suite.Require().True(found)
				suite.Require().Equal(expConsensusState, consensusState)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(consensusState)
			}
		})
	}
}

func (suite *TendermintTestSuite) TestGetProcessedTime() {
	// Verify ProcessedTime on CreateClient
	// coordinator increments time before creating client
	expectedTime := suite.chainA.CurrentHeader.Time.Add(ibctesting.TimeIncrement)

	clientA, err := suite.coordinator.CreateClient(suite.chainA, suite.chainB, exported.Tendermint)
	suite.Require().NoError(err)

	clientState := suite.chainA.GetClientState(clientA)
	height := clientState.GetLatestHeight()

	store := suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), clientA)
	actualTime, ok := types.GetProcessedTime(store, height)
	suite.Require().True(ok, "could not retrieve processed time for stored consensus state")
	suite.Require().Equal(uint64(expectedTime.UnixNano()), actualTime, "retrieved processed time is not expected value")

	// Verify ProcessedTime on UpdateClient
	// coordinator increments time before updating client
	expectedTime = suite.chainA.CurrentHeader.Time.Add(ibctesting.TimeIncrement)

	err = suite.coordinator.UpdateClient(suite.chainA, suite.chainB, clientA, exported.Tendermint)
	suite.Require().NoError(err)

	clientState = suite.chainA.GetClientState(clientA)
	height = clientState.GetLatestHeight()

	store = suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), clientA)
	actualTime, ok = types.GetProcessedTime(store, height)
	suite.Require().True(ok, "could not retrieve processed time for stored consensus state")
	suite.Require().Equal(uint64(expectedTime.UnixNano()), actualTime, "retrieved processed time is not expected value")

	// try to get processed time for height that doesn't exist in store
	_, ok = types.GetProcessedTime(store, clienttypes.NewHeight(1, 1))
	suite.Require().False(ok, "retrieved processed time for a non-existent consensus state")
}

func (suite *TendermintTestSuite) TestIterateConsensusStates() {
	nextValsHash := []byte("nextVals")

	// Set iteration keys and consensus states
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), clienttypes.NewHeight(0, 1))
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), "testClient", clienttypes.NewHeight(0, 1), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-1")), nextValsHash))
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), clienttypes.NewHeight(4, 9))
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), "testClient", clienttypes.NewHeight(4, 9), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash4-9")), nextValsHash))
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), clienttypes.NewHeight(0, 10))
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), "testClient", clienttypes.NewHeight(0, 10), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-10")), nextValsHash))
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), clienttypes.NewHeight(0, 4))
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), "testClient", clienttypes.NewHeight(0, 4), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-4")), nextValsHash))
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), clienttypes.NewHeight(40, 1))
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), "testClient", clienttypes.NewHeight(40, 1), types.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash40-1")), nextValsHash))

	var testArr []string
	cb := func(cs types.ConsensusState) bool {
		testArr = append(testArr, string(cs.Root.GetHash()))
		return false
	}

	types.IterateConsensusStateAscending(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), suite.chainA.Codec, cb)
	expectedArr := []string{"hash0-1", "hash0-4", "hash0-10", "hash4-9", "hash40-1"}
	suite.Require().Equal(expectedArr, testArr)
}

func (suite *TendermintTestSuite) TestGetNeighboringConsensusStates() {
	nextValsHash := []byte("nextVals")
	cs01 := types.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash0-1")), nextValsHash)
	cs04 := types.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash0-4")), nextValsHash)
	cs49 := types.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash4-9")), nextValsHash)
	height01 := clienttypes.NewHeight(0, 1)
	height04 := clienttypes.NewHeight(0, 4)
	height49 := clienttypes.NewHeight(4, 9)

	// Set iteration keys and consensus states
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), height01)
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), "testClient", height01, cs01)
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), height04)
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), "testClient", height04, cs04)
	types.SetIterationKey(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), height49)
	suite.chainA.App.IBCKeeper.ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), "testClient", height49, cs49)

	prevCs01, ok := types.GetPreviousConsensusState(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), suite.chainA.Codec, height01)
	suite.Require().Nil(prevCs01, "consensus state exists before lowest consensus state")
	suite.Require().False(ok)
	prevCs49, ok := types.GetPreviousConsensusState(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), suite.chainA.Codec, height49)
	suite.Require().Equal(cs04, prevCs49, "previous consensus state is not returned correctly")
	suite.Require().True(ok)

	nextCs01, ok := types.GetNextConsensusState(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), suite.chainA.Codec, height01)
	suite.Require().Equal(cs04, nextCs01, "next consensus state not returned correctly")
	suite.Require().True(ok)
	nextCs49, ok := types.GetNextConsensusState(suite.chainA.App.IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), "testClient"), suite.chainA.Codec, height49)
	suite.Require().Nil(nextCs49, "next consensus state exists after highest consensus state")
	suite.Require().False(ok)
}
