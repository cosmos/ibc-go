package tendermint_test

import (
	"math"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	tendermint "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *TendermintTestSuite) TestGetConsensusState() {
	var (
		height exported.Height
		path   *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
		expPanic bool
	}{
		{
			"success", func() {}, true, false,
		},
		{
			"consensus state not found", func() {
				// use height with no consensus state set
				height = height.Increment()
			}, false, false,
		},
		{
			"not a consensus state interface", func() {
				// marshal an empty client state and set as consensus state
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
				clientStateBz := clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), &tendermint.ClientState{})
				store.Set(host.ConsensusStateKey(height), clientStateBz)
			}, false, true,
		},
		{
			"invalid consensus state (solomachine)", func() {
				// marshal and set solomachine consensus state
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
				consensusStateBz := clienttypes.MustMarshalConsensusState(s.chainA.App.AppCodec(), &solomachine.ConsensusState{})
				store.Set(host.ConsensusStateKey(height), consensusStateBz)
			}, false, true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			path.Setup()

			height = path.EndpointA.GetClientLatestHeight()

			tc.malleate() // change vars as necessary

			if tc.expPanic {
				s.Require().Panics(func() {
					store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
					tendermint.GetConsensusState(store, s.chainA.Codec, height)
				})

				return
			}

			store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
			consensusState, found := tendermint.GetConsensusState(store, s.chainA.Codec, height)

			if tc.expPass {
				s.Require().True(found)

				expConsensusState, found := s.chainA.GetConsensusState(path.EndpointA.ClientID, height)
				s.Require().True(found)
				s.Require().Equal(expConsensusState, consensusState)
			} else {
				s.Require().False(found)
				s.Require().Nil(consensusState)
			}
		})
	}
}

func (s *TendermintTestSuite) TestGetProcessedTime() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.UpdateTime()

	expectedTime := s.chainA.ProposedHeader.Time

	// Verify ProcessedTime on CreateClient
	err := path.EndpointA.CreateClient()
	s.Require().NoError(err)

	height := path.EndpointA.GetClientLatestHeight()

	store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
	actualTime, ok := tendermint.GetProcessedTime(store, height)
	s.Require().True(ok, "could not retrieve processed time for stored consensus state")
	s.Require().Equal(uint64(expectedTime.UnixNano()), actualTime, "retrieved processed time is not expected value")

	s.coordinator.UpdateTime()
	// coordinator increments time before updating client
	expectedTime = s.chainA.ProposedHeader.Time.Add(ibctesting.TimeIncrement)

	// Verify ProcessedTime on UpdateClient
	err = path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	height = path.EndpointA.GetClientLatestHeight()

	store = s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
	actualTime, ok = tendermint.GetProcessedTime(store, height)
	s.Require().True(ok, "could not retrieve processed time for stored consensus state")
	s.Require().Equal(uint64(expectedTime.UnixNano()), actualTime, "retrieved processed time is not expected value")

	// try to get processed time for height that doesn't exist in store
	_, ok = tendermint.GetProcessedTime(store, clienttypes.NewHeight(1, 1))
	s.Require().False(ok, "retrieved processed time for a non-existent consensus state")
}

func (s *TendermintTestSuite) TestIterationKey() {
	testHeights := []exported.Height{
		clienttypes.NewHeight(0, 1),
		clienttypes.NewHeight(0, 1234),
		clienttypes.NewHeight(7890, 4321),
		clienttypes.NewHeight(math.MaxUint64, math.MaxUint64),
	}
	for _, h := range testHeights {
		k := tendermint.IterationKey(h)
		retrievedHeight := tendermint.GetHeightFromIterationKey(k)
		s.Require().Equal(h, retrievedHeight, "retrieving height from iteration key failed")
	}
}

func (s *TendermintTestSuite) TestIterateConsensusStates() {
	nextValsHash := []byte("nextVals")

	// Set iteration keys and consensus states
	tendermint.SetIterationKey(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), clienttypes.NewHeight(0, 1))
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), "testClient", clienttypes.NewHeight(0, 1), tendermint.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-1")), nextValsHash))
	tendermint.SetIterationKey(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), clienttypes.NewHeight(4, 9))
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), "testClient", clienttypes.NewHeight(4, 9), tendermint.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash4-9")), nextValsHash))
	tendermint.SetIterationKey(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), clienttypes.NewHeight(0, 10))
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), "testClient", clienttypes.NewHeight(0, 10), tendermint.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-10")), nextValsHash))
	tendermint.SetIterationKey(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), clienttypes.NewHeight(0, 4))
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), "testClient", clienttypes.NewHeight(0, 4), tendermint.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash0-4")), nextValsHash))
	tendermint.SetIterationKey(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), clienttypes.NewHeight(40, 1))
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), "testClient", clienttypes.NewHeight(40, 1), tendermint.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash40-1")), nextValsHash))

	var testArr []string
	cb := func(height exported.Height) bool {
		testArr = append(testArr, height.String())
		return false
	}

	tendermint.IterateConsensusStateAscending(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), cb)
	expectedArr := []string{"0-1", "0-4", "0-10", "4-9", "40-1"}
	s.Require().Equal(expectedArr, testArr)
}

func (s *TendermintTestSuite) TestGetNeighboringConsensusStates() {
	nextValsHash := []byte("nextVals")
	cs01 := tendermint.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash0-1")), nextValsHash)
	cs04 := tendermint.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash0-4")), nextValsHash)
	cs49 := tendermint.NewConsensusState(time.Now().UTC(), commitmenttypes.NewMerkleRoot([]byte("hash4-9")), nextValsHash)
	height01 := clienttypes.NewHeight(0, 1)
	height04 := clienttypes.NewHeight(0, 4)
	height49 := clienttypes.NewHeight(4, 9)

	// Set iteration keys and consensus states
	tendermint.SetIterationKey(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), height01)
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), "testClient", height01, cs01)
	tendermint.SetIterationKey(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), height04)
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), "testClient", height04, cs04)
	tendermint.SetIterationKey(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), height49)
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), "testClient", height49, cs49)

	prevCs01, ok := tendermint.GetPreviousConsensusState(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), s.chainA.Codec, height01)
	s.Require().Nil(prevCs01, "consensus state exists before lowest consensus state")
	s.Require().False(ok)
	prevCs49, ok := tendermint.GetPreviousConsensusState(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), s.chainA.Codec, height49)
	s.Require().Equal(cs04, prevCs49, "previous consensus state is not returned correctly")
	s.Require().True(ok)

	nextCs01, ok := tendermint.GetNextConsensusState(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), s.chainA.Codec, height01)
	s.Require().Equal(cs04, nextCs01, "next consensus state not returned correctly")
	s.Require().True(ok)
	nextCs49, ok := tendermint.GetNextConsensusState(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "testClient"), s.chainA.Codec, height49)
	s.Require().Nil(nextCs49, "next consensus state exists after highest consensus state")
	s.Require().False(ok)
}
