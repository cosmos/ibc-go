package types_test

import (
	fmt "fmt"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/01-dymint/types"
	tmtypes "github.com/cosmos/ibc-go/v3/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

var (
	frozenHeight = clienttypes.NewHeight(0, 1)
)

func (suite *DymintTestSuite) TestCheckSubstituteUpdateStateBasic() {
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
				suite.coordinator.SetupClients(substitutePath)
				substituteClientState = suite.chainA.GetClientState(substitutePath.EndpointA.ClientID)
				switch substituteClientState.ClientType() {
				case exported.Dymint:
					dmClientState, ok := substituteClientState.(*types.ClientState)
					suite.Require().True(ok)

					dmClientState.ChainId = dmClientState.ChainId + "different chain"
				case exported.Tendermint:
					tmClientState, ok := substituteClientState.(*tmtypes.ClientState)
					suite.Require().True(ok)

					tmClientState.ChainId = tmClientState.ChainId + "different chain"
				default:
					panic(fmt.Sprintf("client type %s is not supported", substituteClientState.ClientType()))
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {

			suite.SetupTest() // reset
			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath = ibctesting.NewPath(suite.chainA, suite.chainB)

			suite.coordinator.SetupClients(subjectPath)
			subjectClientState := suite.chainA.GetClientState(subjectPath.EndpointA.ClientID)
			switch subjectClientState.ClientType() {
			case exported.Dymint:
				subjectDMClientState := subjectClientState.(*types.ClientState)
				subjectDMClientState.AllowUpdateAfterMisbehaviour = true
				subjectDMClientState.AllowUpdateAfterExpiry = true
				// expire subject client
				suite.coordinator.IncrementTimeBy(subjectDMClientState.TrustingPeriod)
			case exported.Tendermint:
				subjectTMClientState := subjectClientState.(*tmtypes.ClientState)
				subjectTMClientState.AllowUpdateAfterMisbehaviour = true
				subjectTMClientState.AllowUpdateAfterExpiry = true
				// expire subject client
				suite.coordinator.IncrementTimeBy(subjectTMClientState.TrustingPeriod)
			default:
				panic(fmt.Sprintf("client type %s is not supported", subjectClientState.ClientType()))
			}

			suite.coordinator.CommitBlock(suite.chainA, suite.chainB)

			tc.malleate()

			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
			substituteClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substitutePath.EndpointA.ClientID)

			updatedClient, err := subjectClientState.CheckSubstituteAndUpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteClientState)
			suite.Require().Error(err)
			suite.Require().Nil(updatedClient)
		})
	}
}

// to expire clients, time needs to be fast forwarded on both chainA and chainB.
// this is to prevent headers from failing when attempting to update later.
func (suite *DymintTestSuite) TestCheckSubstituteAndUpdateState() {
	testCases := []struct {
		name                         string
		AllowUpdateAfterExpiry       bool
		AllowUpdateAfterMisbehaviour bool
		FreezeClient                 bool
		ExpireClient                 bool
		expPass                      bool
	}{
		{
			name:                         "not allowed to be updated, not frozen or expired",
			AllowUpdateAfterExpiry:       false,
			AllowUpdateAfterMisbehaviour: false,
			FreezeClient:                 false,
			ExpireClient:                 false,
			expPass:                      false,
		},
		{
			name:                         "not allowed to be updated, client is frozen",
			AllowUpdateAfterExpiry:       false,
			AllowUpdateAfterMisbehaviour: false,
			FreezeClient:                 true,
			ExpireClient:                 false,
			expPass:                      false,
		},
		{
			name:                         "not allowed to be updated, client is expired",
			AllowUpdateAfterExpiry:       false,
			AllowUpdateAfterMisbehaviour: false,
			FreezeClient:                 false,
			ExpireClient:                 true,
			expPass:                      false,
		},
		{
			name:                         "not allowed to be updated, client is frozen and expired",
			AllowUpdateAfterExpiry:       false,
			AllowUpdateAfterMisbehaviour: false,
			FreezeClient:                 true,
			ExpireClient:                 true,
			expPass:                      false,
		},
		{
			name:                         "allowed to be updated only after misbehaviour, not frozen or expired",
			AllowUpdateAfterExpiry:       false,
			AllowUpdateAfterMisbehaviour: true,
			FreezeClient:                 false,
			ExpireClient:                 false,
			expPass:                      false,
		},
		{
			name:                         "allowed to be updated only after misbehaviour, client is expired",
			AllowUpdateAfterExpiry:       false,
			AllowUpdateAfterMisbehaviour: true,
			FreezeClient:                 false,
			ExpireClient:                 true,
			expPass:                      false,
		},
		{
			name:                         "allowed to be updated only after expiry, not frozen or expired",
			AllowUpdateAfterExpiry:       true,
			AllowUpdateAfterMisbehaviour: false,
			FreezeClient:                 false,
			ExpireClient:                 false,
			expPass:                      false,
		},
		{
			name:                         "allowed to be updated only after expiry, client is frozen",
			AllowUpdateAfterExpiry:       true,
			AllowUpdateAfterMisbehaviour: false,
			FreezeClient:                 true,
			ExpireClient:                 false,
			expPass:                      false,
		},
		{
			name:                         "PASS: allowed to be updated only after misbehaviour, client is frozen",
			AllowUpdateAfterExpiry:       false,
			AllowUpdateAfterMisbehaviour: true,
			FreezeClient:                 true,
			ExpireClient:                 false,
			expPass:                      true,
		},
		{
			name:                         "PASS: allowed to be updated only after misbehaviour, client is frozen and expired",
			AllowUpdateAfterExpiry:       false,
			AllowUpdateAfterMisbehaviour: true,
			FreezeClient:                 true,
			ExpireClient:                 true,
			expPass:                      true,
		},
		{
			name:                         "PASS: allowed to be updated only after expiry, client is expired",
			AllowUpdateAfterExpiry:       true,
			AllowUpdateAfterMisbehaviour: false,
			FreezeClient:                 false,
			ExpireClient:                 true,
			expPass:                      true,
		},
		{
			name:                         "allowed to be updated only after expiry, client is frozen and expired",
			AllowUpdateAfterExpiry:       true,
			AllowUpdateAfterMisbehaviour: false,
			FreezeClient:                 true,
			ExpireClient:                 true,
			expPass:                      false,
		},
		{
			name:                         "allowed to be updated after expiry and misbehaviour, not frozen or expired",
			AllowUpdateAfterExpiry:       true,
			AllowUpdateAfterMisbehaviour: true,
			FreezeClient:                 false,
			ExpireClient:                 false,
			expPass:                      false,
		},
		{
			name:                         "PASS: allowed to be updated after expiry and misbehaviour, client is frozen",
			AllowUpdateAfterExpiry:       true,
			AllowUpdateAfterMisbehaviour: true,
			FreezeClient:                 true,
			ExpireClient:                 false,
			expPass:                      true,
		},
		{
			name:                         "PASS: allowed to be updated after expiry and misbehaviour, client is expired",
			AllowUpdateAfterExpiry:       true,
			AllowUpdateAfterMisbehaviour: true,
			FreezeClient:                 false,
			ExpireClient:                 true,
			expPass:                      true,
		},
		{
			name:                         "PASS: allowed to be updated after expiry and misbehaviour, client is frozen and expired",
			AllowUpdateAfterExpiry:       true,
			AllowUpdateAfterMisbehaviour: true,
			FreezeClient:                 true,
			ExpireClient:                 true,
			expPass:                      true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		// for each test case a header used for unexpiring clients and unfreezing
		// a client are each tested to ensure that unexpiry headers cannot update
		// a client when a unfreezing header is required.
		suite.Run(tc.name, func() {

			// start by testing unexpiring the client
			suite.SetupTest() // reset

			// construct subject using test case parameters
			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(subjectPath)

			subjectClientState := suite.chainA.GetClientState(subjectPath.EndpointA.ClientID)
			switch subjectClientState.ClientType() {
			case exported.Dymint:
				subjectDMClientState := subjectClientState.(*types.ClientState)
				subjectDMClientState.AllowUpdateAfterMisbehaviour = tc.AllowUpdateAfterMisbehaviour
				subjectDMClientState.AllowUpdateAfterExpiry = tc.AllowUpdateAfterExpiry

				// apply freezing or expiry as determined by the test case
				if tc.FreezeClient {
					subjectDMClientState.FrozenHeight = frozenHeight
				}
				if tc.ExpireClient {
					// expire subject client
					suite.coordinator.IncrementTimeBy(subjectDMClientState.TrustingPeriod)
					suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
				}
			case exported.Tendermint:
				subjectTMClientState := subjectClientState.(*tmtypes.ClientState)
				subjectTMClientState.AllowUpdateAfterMisbehaviour = tc.AllowUpdateAfterMisbehaviour
				subjectTMClientState.AllowUpdateAfterExpiry = tc.AllowUpdateAfterExpiry

				// apply freezing or expiry as determined by the test case
				if tc.FreezeClient {
					subjectTMClientState.FrozenHeight = frozenHeight
				}
				if tc.ExpireClient {
					// expire subject client
					suite.coordinator.IncrementTimeBy(subjectTMClientState.TrustingPeriod)
					suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
				}

			default:
				panic(fmt.Sprintf("client type %s is not supported", subjectClientState.ClientType()))
			}

			// construct the substitute to match the subject client
			// NOTE: the substitute is explicitly created after the freezing or expiry occurs,
			// primarily to prevent the substitute from becoming frozen. It also should be
			// the natural flow of events in practice. The subject will become frozen/expired
			// and a substitute will be created along with a governance proposal as a response

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(substitutePath)

			substituteClientState := suite.chainA.GetClientState(substitutePath.EndpointA.ClientID)
			switch substituteClientState.ClientType() {
			case exported.Dymint:
				substituteDMClientState := substituteClientState.(*types.ClientState)
				substituteDMClientState.AllowUpdateAfterMisbehaviour = tc.AllowUpdateAfterMisbehaviour
				substituteDMClientState.AllowUpdateAfterExpiry = tc.AllowUpdateAfterExpiry
			case exported.Tendermint:
				substituteTMClientState := substituteClientState.(*tmtypes.ClientState)
				substituteTMClientState.AllowUpdateAfterMisbehaviour = tc.AllowUpdateAfterMisbehaviour
				substituteTMClientState.AllowUpdateAfterExpiry = tc.AllowUpdateAfterExpiry
			default:
				panic(fmt.Sprintf("client type %s is not supported", subjectClientState.ClientType()))
			}

			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitutePath.EndpointA.ClientID, substituteClientState)

			// update substitute a few times
			for i := 0; i < 3; i++ {
				err := substitutePath.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				// skip a block
				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
			}

			// test that subject gets updated chain-id
			newChainID := "new-chain-id"
			// get updated substitute
			substituteClientState = suite.chainA.GetClientState(substitutePath.EndpointA.ClientID)
			switch substituteClientState.ClientType() {
			case exported.Dymint:
				substituteDMClientState := substituteClientState.(*types.ClientState)
				substituteDMClientState.ChainId = newChainID
			case exported.Tendermint:
				substituteTMClientState := substituteClientState.(*tmtypes.ClientState)
				substituteTMClientState.ChainId = newChainID
			default:
				panic(fmt.Sprintf("client type %s is not supported", subjectClientState.ClientType()))
			}

			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
			substituteClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substitutePath.EndpointA.ClientID)

			expectedConsState := substitutePath.EndpointA.GetConsensusState(substituteClientState.GetLatestHeight())
			expectedProcessedTime, found := types.GetProcessedTime(substituteClientStore, substituteClientState.GetLatestHeight())
			suite.Require().True(found)
			expectedProcessedHeight, found := types.GetProcessedTime(substituteClientStore, substituteClientState.GetLatestHeight())
			suite.Require().True(found)
			expectedIterationKey := types.GetIterationKey(substituteClientStore, substituteClientState.GetLatestHeight())

			updatedClient, err := subjectClientState.CheckSubstituteAndUpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteClientState)

			if tc.expPass {
				suite.Require().NoError(err)

				updatedClientChainId := newChainID
				FrozenHeight := clienttypes.ZeroHeight()
				switch updatedClient.ClientType() {
				case exported.Dymint:
					updatedClientChainId = updatedClient.(*types.ClientState).ChainId
					FrozenHeight = updatedClient.(*types.ClientState).FrozenHeight
				case exported.Tendermint:
					updatedClientChainId = updatedClient.(*tmtypes.ClientState).ChainId
					FrozenHeight = updatedClient.(*tmtypes.ClientState).FrozenHeight
				default:
					panic(fmt.Sprintf("client type %s is not supported", subjectClientState.ClientType()))
				}
				suite.Require().Equal(newChainID, updatedClientChainId)
				suite.Require().Equal(clienttypes.ZeroHeight(), FrozenHeight)

				subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)

				// check that the correct consensus state was copied over
				suite.Require().Equal(substituteClientState.GetLatestHeight(), updatedClient.GetLatestHeight())
				subjectConsState := subjectPath.EndpointA.GetConsensusState(updatedClient.GetLatestHeight())
				subjectProcessedTime, found := types.GetProcessedTime(subjectClientStore, updatedClient.GetLatestHeight())
				suite.Require().True(found)
				subjectProcessedHeight, found := types.GetProcessedTime(substituteClientStore, updatedClient.GetLatestHeight())
				suite.Require().True(found)
				subjectIterationKey := types.GetIterationKey(substituteClientStore, updatedClient.GetLatestHeight())

				suite.Require().Equal(expectedConsState, subjectConsState)
				suite.Require().Equal(expectedProcessedTime, subjectProcessedTime)
				suite.Require().Equal(expectedProcessedHeight, subjectProcessedHeight)
				suite.Require().Equal(expectedIterationKey, subjectIterationKey)

			} else {
				suite.Require().Error(err)
				suite.Require().Nil(updatedClient)
			}

		})
	}
}

func (suite *DymintTestSuite) TestIsMatchingClientState() {
	var (
		subjectPath, substitutePath               *ibctesting.Path
		subjectClientState, substituteClientState *types.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"matching clients", func() {
				switch suite.chainA.TestChainClient.GetSelfClientType() {
				case exported.Dymint:
					// ChainBs' counterparty client is Dymint
					subjectClientState = suite.chainB.GetClientState(subjectPath.EndpointB.ClientID).(*types.ClientState)
					substituteClientState = suite.chainB.GetClientState(substitutePath.EndpointB.ClientID).(*types.ClientState)
				case exported.Tendermint:
					// ChainAs' counterparty client is Dymint
					subjectClientState = suite.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*types.ClientState)
					substituteClientState = suite.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*types.ClientState)
				default:
					panic(fmt.Sprintf("client type %s is not supported", subjectClientState.ClientType()))
				}
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
			"not matching, trusting period is different", func() {
				subjectClientState.TrustingPeriod = time.Duration(time.Hour * 10)
				substituteClientState.TrustingPeriod = time.Duration(time.Hour * 1)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			subjectPath = ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(subjectPath)
			suite.coordinator.SetupClients(substitutePath)

			tc.malleate()

			suite.Require().Equal(tc.expPass, types.IsMatchingClientState(*subjectClientState, *substituteClientState))

		})
	}
}
