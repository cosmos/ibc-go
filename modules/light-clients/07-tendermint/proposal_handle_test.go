package tendermint_test

import (
	"time"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

var frozenHeight = clienttypes.NewHeight(0, 1)

func (suite *TendermintTestSuite) TestCheckSubstituteUpdateStateBasic() {
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
				substituteClientState = ibctesting.NewSolomachine(suite.T(), suite.cdc, "solo machine", "", 1).ClientState()
			},
		},
		{
			"non-matching substitute", func() {
				substitutePath.SetupClients()
				substituteClientState, ok := suite.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*ibctm.ClientState)
				suite.Require().True(ok)
				// change trusting period so that test should fail
				substituteClientState.TrustingPeriod = time.Hour * 24 * 7

				tmClientState := substituteClientState
				tmClientState.ChainId += "different chain"
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath = ibctesting.NewPath(suite.chainA, suite.chainB)

			subjectPath.SetupClients()
			subjectClientState, ok := suite.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*ibctm.ClientState)
			suite.Require().True(ok)

			// expire subject client
			suite.coordinator.IncrementTimeBy(subjectClientState.TrustingPeriod)
			suite.coordinator.CommitBlock(suite.chainA, suite.chainB)

			tc.malleate()

			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
			substituteClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substitutePath.EndpointA.ClientID)

			err := subjectClientState.CheckSubstituteAndUpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteClientState)
			suite.Require().Error(err)
		})
	}
}

func (suite *TendermintTestSuite) TestCheckSubstituteAndUpdateState() {
	testCases := []struct {
		name         string
		FreezeClient bool
		expPass      bool
	}{
		{
			name:         "PASS: update checks are deprecated, client is not frozen",
			FreezeClient: false,
			expPass:      true,
		},
		{
			name:         "PASS: update checks are deprecated, client is frozen",
			FreezeClient: true,
			expPass:      true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// construct subject using test case parameters
			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			subjectPath.SetupClients()
			subjectClientState, ok := suite.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*ibctm.ClientState)
			suite.Require().True(ok)

			if tc.FreezeClient {
				subjectClientState.FrozenHeight = frozenHeight
			}

			// construct the substitute to match the subject client

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath.SetupClients()
			substituteClientState, ok := suite.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*ibctm.ClientState)
			suite.Require().True(ok)
			// update trusting period of substitute client state
			substituteClientState.TrustingPeriod = time.Hour * 24 * 7
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitutePath.EndpointA.ClientID, substituteClientState)

			// update substitute a few times
			for i := 0; i < 3; i++ {
				err := substitutePath.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				// skip a block
				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
			}

			// get updated substitute
			substituteClientState, ok = suite.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*ibctm.ClientState)
			suite.Require().True(ok)

			// test that subject gets updated chain-id
			newChainID := "new-chain-id"
			substituteClientState.ChainId = newChainID

			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
			substituteClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substitutePath.EndpointA.ClientID)

			expectedConsState := substitutePath.EndpointA.GetConsensusState(substituteClientState.LatestHeight)
			expectedProcessedTime, found := ibctm.GetProcessedTime(substituteClientStore, substituteClientState.LatestHeight)
			suite.Require().True(found)
			expectedProcessedHeight, found := ibctm.GetProcessedTime(substituteClientStore, substituteClientState.LatestHeight)
			suite.Require().True(found)
			expectedIterationKey := ibctm.GetIterationKey(substituteClientStore, substituteClientState.LatestHeight)

			err := subjectClientState.CheckSubstituteAndUpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteClientState)

			if tc.expPass {
				suite.Require().NoError(err)

				updatedClient, ok := subjectPath.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().Equal(clienttypes.ZeroHeight(), updatedClient.FrozenHeight)

				subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)

				// check that the correct consensus state was copied over
				suite.Require().Equal(substituteClientState.LatestHeight, updatedClient.LatestHeight)
				subjectConsState := subjectPath.EndpointA.GetConsensusState(updatedClient.LatestHeight)
				subjectProcessedTime, found := ibctm.GetProcessedTime(subjectClientStore, updatedClient.LatestHeight)
				suite.Require().True(found)
				subjectProcessedHeight, found := ibctm.GetProcessedTime(substituteClientStore, updatedClient.LatestHeight)
				suite.Require().True(found)
				subjectIterationKey := ibctm.GetIterationKey(substituteClientStore, updatedClient.LatestHeight)

				suite.Require().Equal(expectedConsState, subjectConsState)
				suite.Require().Equal(expectedProcessedTime, subjectProcessedTime)
				suite.Require().Equal(expectedProcessedHeight, subjectProcessedHeight)
				suite.Require().Equal(expectedIterationKey, subjectIterationKey)

				suite.Require().Equal(newChainID, updatedClient.ChainId)
				suite.Require().Equal(time.Hour*24*7, updatedClient.TrustingPeriod)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TendermintTestSuite) TestIsMatchingClientState() {
	var (
		subjectPath, substitutePath               *ibctesting.Path
		subjectClientState, substituteClientState *ibctm.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"matching clients", func() {
				var ok bool
				subjectClientState, ok = suite.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*ibctm.ClientState)
				suite.Require().True(ok)
				substituteClientState, ok = suite.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*ibctm.ClientState)
				suite.Require().True(ok)
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
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			subjectPath = ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath = ibctesting.NewPath(suite.chainA, suite.chainB)
			subjectPath.SetupClients()
			substitutePath.SetupClients()

			tc.malleate()

			suite.Require().Equal(tc.expPass, ibctm.IsMatchingClientState(*subjectClientState, *substituteClientState))
		})
	}
}
