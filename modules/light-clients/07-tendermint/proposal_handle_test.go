package tendermint_test

import (
	"time"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

var frozenHeight = clienttypes.NewHeight(0, 1)

func (s *TendermintTestSuite) TestCheckSubstituteUpdateStateBasic() {
	var (
		substituteClientState exported.ClientState
		substitutePath        *ibctesting.Path
	)
	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"solo machine used for substitute", func() {
				substituteClientState = ibctesting.NewSolomachine(s.T(), s.cdc, "solo machine", "", 1).ClientState()
			},
		},
		{
			"non-matching substitute", func() {
				substitutePath.SetupClients()
				substituteClientState, ok := s.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*ibctm.ClientState)
				s.Require().True(ok)
				// change trusting period so that test should fail
				substituteClientState.TrustingPeriod = time.Hour * 24 * 7

				tmClientState := substituteClientState
				tmClientState.ChainId += "different chain"
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			subjectPath := ibctesting.NewPath(s.chainA, s.chainB)
			substitutePath = ibctesting.NewPath(s.chainA, s.chainB)

			subjectPath.SetupClients()
			subjectClientState, ok := s.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*ibctm.ClientState)
			s.Require().True(ok)

			// expire subject client
			s.coordinator.IncrementTimeBy(subjectClientState.TrustingPeriod)
			s.coordinator.CommitBlock(s.chainA, s.chainB)

			tc.malleate()

			subjectClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), subjectPath.EndpointA.ClientID)
			substituteClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), substitutePath.EndpointA.ClientID)

			err := subjectClientState.CheckSubstituteAndUpdateState(s.chainA.GetContext(), s.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteClientState)
			s.Require().Error(err)
		})
	}
}

func (s *TendermintTestSuite) TestCheckSubstituteAndUpdateState() {
	testCases := []struct {
		name         string
		FreezeClient bool
		expError     error
	}{
		{
			name:         "PASS: update checks are deprecated, client is not frozen",
			FreezeClient: false,
			expError:     nil,
		},
		{
			name:         "PASS: update checks are deprecated, client is frozen",
			FreezeClient: true,
			expError:     nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			// construct subject using test case parameters
			subjectPath := ibctesting.NewPath(s.chainA, s.chainB)
			subjectPath.SetupClients()
			subjectClientState, ok := s.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*ibctm.ClientState)
			s.Require().True(ok)

			if tc.FreezeClient {
				subjectClientState.FrozenHeight = frozenHeight
			}

			// construct the substitute to match the subject client

			substitutePath := ibctesting.NewPath(s.chainA, s.chainB)
			substitutePath.SetupClients()
			substituteClientState, ok := s.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*ibctm.ClientState)
			s.Require().True(ok)
			// update trusting period of substitute client state
			substituteClientState.TrustingPeriod = time.Hour * 24 * 7
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitutePath.EndpointA.ClientID, substituteClientState)

			// update substitute a few times
			for range 3 {
				err := substitutePath.EndpointA.UpdateClient()
				s.Require().NoError(err)
				// skip a block
				s.coordinator.CommitBlock(s.chainA, s.chainB)
			}

			// get updated substitute
			substituteClientState, ok = s.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*ibctm.ClientState)
			s.Require().True(ok)

			// test that subject gets updated chain-id
			newChainID := "new-chain-id"
			substituteClientState.ChainId = newChainID

			subjectClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), subjectPath.EndpointA.ClientID)
			substituteClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), substitutePath.EndpointA.ClientID)

			expectedConsState := substitutePath.EndpointA.GetConsensusState(substituteClientState.LatestHeight)
			expectedProcessedTime, found := ibctm.GetProcessedTime(substituteClientStore, substituteClientState.LatestHeight)
			s.Require().True(found)
			expectedProcessedHeight, found := ibctm.GetProcessedTime(substituteClientStore, substituteClientState.LatestHeight)
			s.Require().True(found)
			expectedIterationKey := ibctm.GetIterationKey(substituteClientStore, substituteClientState.LatestHeight)

			err := subjectClientState.CheckSubstituteAndUpdateState(s.chainA.GetContext(), s.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteClientState)

			if tc.expError == nil {
				s.Require().NoError(err)

				updatedClient, ok := subjectPath.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				s.Require().Equal(clienttypes.ZeroHeight(), updatedClient.FrozenHeight)

				subjectClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), subjectPath.EndpointA.ClientID)

				// check that the correct consensus state was copied over
				s.Require().Equal(substituteClientState.LatestHeight, updatedClient.LatestHeight)
				subjectConsState := subjectPath.EndpointA.GetConsensusState(updatedClient.LatestHeight)
				subjectProcessedTime, found := ibctm.GetProcessedTime(subjectClientStore, updatedClient.LatestHeight)
				s.Require().True(found)
				subjectProcessedHeight, found := ibctm.GetProcessedTime(substituteClientStore, updatedClient.LatestHeight)
				s.Require().True(found)
				subjectIterationKey := ibctm.GetIterationKey(substituteClientStore, updatedClient.LatestHeight)

				s.Require().Equal(expectedConsState, subjectConsState)
				s.Require().Equal(expectedProcessedTime, subjectProcessedTime)
				s.Require().Equal(expectedProcessedHeight, subjectProcessedHeight)
				s.Require().Equal(expectedIterationKey, subjectIterationKey)

				s.Require().Equal(newChainID, updatedClient.ChainId)
				s.Require().Equal(time.Hour*24*7, updatedClient.TrustingPeriod)
			} else {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expError.Error())
			}
		})
	}
}

func (s *TendermintTestSuite) TestIsMatchingClientState() {
	var (
		subjectPath, substitutePath               *ibctesting.Path
		subjectClientState, substituteClientState *ibctm.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		isMatch  bool
	}{
		{
			"matching clients", func() {
				var ok bool
				subjectClientState, ok = s.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*ibctm.ClientState)
				s.Require().True(ok)
				substituteClientState, ok = s.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*ibctm.ClientState)
				s.Require().True(ok)
			}, true,
		},
		{
			"matching, frozen height is not used in check for equality", func() {
				subjectClientState.FrozenHeight = frozenHeight
				substituteClientState.FrozenHeight = clienttypes.ZeroHeight()
			}, true,
		},
		{
			"matching, latest height is not used in check for equality", func() {
				subjectClientState.LatestHeight = clienttypes.NewHeight(0, 10)
				substituteClientState.FrozenHeight = clienttypes.ZeroHeight()
			}, true,
		},
		{
			"matching, chain id is different", func() {
				subjectClientState.ChainId = "bitcoin"
				substituteClientState.ChainId = "ethereum"
			}, true,
		},
		{
			"matching, trusting period is different", func() {
				subjectClientState.TrustingPeriod = time.Hour * 10
				substituteClientState.TrustingPeriod = time.Hour * 1
			}, true,
		},
		{
			"not matching, trust level is different", func() {
				subjectClientState.TrustLevel = ibctm.Fraction{2, 3}
				substituteClientState.TrustLevel = ibctm.Fraction{1, 3}
			}, false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			subjectPath = ibctesting.NewPath(s.chainA, s.chainB)
			substitutePath = ibctesting.NewPath(s.chainA, s.chainB)
			subjectPath.SetupClients()
			substitutePath.SetupClients()

			tc.malleate()

			s.Require().Equal(tc.isMatch, ibctm.IsMatchingClientState(*subjectClientState, *substituteClientState))
		})
	}
}
