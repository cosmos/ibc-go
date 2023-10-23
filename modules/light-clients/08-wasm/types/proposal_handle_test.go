package types_test

import (
	"encoding/base64"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

// var frozenHeight = clienttypes.NewHeight(0, 1)

// TestCheckSubstituteAndUpdateState only tests the interface to the contract, not the full logic of the contract.
func (suite *TypesTestSuite) TestCheckSubstituteAndUpdateStateGrandpa() {
	var (
		ok                                        bool
		subjectClientState, substituteClientState exported.ClientState
		subjectClientStore, substituteClientStore storetypes.KVStore
	)
	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"success",
			func() {},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()
			subjectClientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, defaultWasmClientID)
			suite.Require().True(ok)
			subjectClientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, defaultWasmClientID)

			substituteClientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, defaultWasmClientID)
			suite.Require().True(ok)

			consensusStateData, err := base64.StdEncoding.DecodeString(suite.testData["consensus_state_data"])
			suite.Require().NoError(err)
			substituteConsensusState := types.ConsensusState{
				Data: consensusStateData,
			}

			substituteClientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, "08-wasm-1")
			err = substituteClientState.Initialize(suite.ctx, suite.chainA.Codec, substituteClientStore, &substituteConsensusState)
			suite.Require().NoError(err)

			tc.setup()

			err = subjectClientState.CheckSubstituteAndUpdateState(
				suite.ctx,
				suite.chainA.Codec,
				subjectClientStore,
				substituteClientStore,
				substituteClientState,
			)
			if tc.expPass {
				suite.Require().NoError(err)

				// Verify that the substitute client state is in the subject client store
				clientStateBz := subjectClientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)
				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				suite.Require().Equal(substituteClientState.GetLatestHeight(), newClientState.GetLatestHeight())
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// func (suite *TypesTestSuite) TestCheckSubstituteAndUpdateStateBasicTendermint() {
// 	var (
// 		substituteClientState exported.ClientState
// 		substitutePath        *ibctesting.Path
// 	)
// 	testCases := []struct {
// 		name     string
// 		malleate func()
// 	}{
// 		{
// 			"solo machine used for substitute", func() {
// 				substituteClientState = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solo machine", "", 1).ClientState()
// 			},
// 		},
// 		{
// 			"non-matching substitute", func() {
// 				suite.coordinator.SetupClients(substitutePath)
// 				substituteWasmClientState := suite.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*types.ClientState)

// 				var clientStateData exported.ClientState
// 				err := suite.chainA.Codec.UnmarshalInterface(substituteWasmClientState.Data, &clientStateData)
// 				suite.Require().NoError(err)
// 				tmClientState := clientStateData.(*ibctm.ClientState)

// 				// change unbonding period so that test should fail
// 				tmClientState.UnbondingPeriod = time.Hour * 24 * 7

// 				tmClientStateBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), tmClientState)
// 				suite.Require().NoError(err)

// 				substituteWasmClientState.Data = tmClientStateBz

// 				substituteClientState = substituteWasmClientState
// 				substitutePath.EndpointA.SetClientState(substituteClientState)
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		suite.Run(tc.name, func() {
// 			suite.SetupWasmTendermint() // reset
// 			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
// 			substitutePath = ibctesting.NewPath(suite.chainA, suite.chainB)

// 			suite.coordinator.SetupClients(subjectPath)
// 			subjectClientState := suite.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*types.ClientState)

// 			var clientStateData exported.ClientState
// 			err := suite.chainA.Codec.UnmarshalInterface(subjectClientState.Data, &clientStateData)
// 			suite.Require().NoError(err)
// 			tmClientState := clientStateData.(*ibctm.ClientState)

// 			// expire subject client
// 			suite.coordinator.IncrementTimeBy(tmClientState.TrustingPeriod)
// 			suite.coordinator.CommitBlock(suite.chainA, suite.chainB)

// 			tc.malleate()

// 			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
// 			substituteClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substitutePath.EndpointA.ClientID)

// 			err = subjectClientState.CheckSubstituteAndUpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteClientState)
// 			suite.Require().Error(err)
// 		})
// 	}
// }

// func (suite *TypesTestSuite) TestCheckSubstituteAndUpdateStateTendermint() {
// 	testCases := []struct {
// 		name         string
// 		FreezeClient bool
// 		expPass      bool
// 	}{
// 		{
// 			name:         "PASS: update checks are deprecated, client is not frozen",
// 			FreezeClient: false,
// 			expPass:      true,
// 		},
// 		{
// 			name:         "PASS: update checks are deprecated, client is frozen",
// 			FreezeClient: true,
// 			expPass:      true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		suite.Run(tc.name, func() {
// 			suite.SetupWasmTendermint() // reset

// 			// construct subject using test case parameters
// 			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
// 			suite.coordinator.SetupClients(subjectPath)
// 			subjectWasmClientState := suite.chainA.GetClientState(subjectPath.EndpointA.ClientID).(*types.ClientState)

// 			var subjectWasmClientStateData exported.ClientState
// 			err := suite.chainA.Codec.UnmarshalInterface(subjectWasmClientState.Data, &subjectWasmClientStateData)
// 			suite.Require().NoError(err)
// 			subjectTmClientState := subjectWasmClientStateData.(*ibctm.ClientState)

// 			if tc.FreezeClient {
// 				subjectTmClientState.FrozenHeight = frozenHeight
// 			}

// 			subjectTmClientStateBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), subjectTmClientState)
// 			suite.Require().NoError(err)
// 			subjectWasmClientState.Data = subjectTmClientStateBz
// 			subjectPath.EndpointA.SetClientState(subjectWasmClientState)

// 			// construct the substitute to match the subject client

// 			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
// 			suite.coordinator.SetupClients(substitutePath)
// 			substituteWasmClientState := suite.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*types.ClientState)

// 			var substituteWasmClientStateData exported.ClientState
// 			err = suite.chainA.Codec.UnmarshalInterface(substituteWasmClientState.Data, &substituteWasmClientStateData)
// 			suite.Require().NoError(err)
// 			substituteTmClientState := substituteWasmClientStateData.(*ibctm.ClientState)

// 			// update trusting period of substitute client state
// 			substituteTmClientState.TrustingPeriod = time.Hour * 24 * 7

// 			substituteTmClientStateBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), substituteTmClientState)
// 			suite.Require().NoError(err)
// 			substituteWasmClientState.Data = substituteTmClientStateBz
// 			substitutePath.EndpointA.SetClientState(substituteWasmClientState)

// 			// update substitute a few times
// 			for i := 0; i < 3; i++ {
// 				err := substitutePath.EndpointA.UpdateClient()
// 				suite.Require().NoError(err)
// 				// skip a block
// 				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
// 			}

// 			// get updated substitute
// 			substituteWasmClientState = suite.chainA.GetClientState(substitutePath.EndpointA.ClientID).(*types.ClientState)
// 			err = suite.chainA.Codec.UnmarshalInterface(substituteWasmClientState.Data, &substituteWasmClientStateData)
// 			suite.Require().NoError(err)
// 			substituteTmClientState = substituteWasmClientStateData.(*ibctm.ClientState)

// 			// test that subject gets updated chain-id
// 			newChainID := "new-chain-id"
// 			substituteTmClientState.ChainId = newChainID

// 			substituteTmClientStateBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), substituteTmClientState)
// 			suite.Require().NoError(err)
// 			substituteWasmClientState.Data = substituteTmClientStateBz
// 			substitutePath.EndpointA.SetClientState(substituteWasmClientState)

// 			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
// 			substituteClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substitutePath.EndpointA.ClientID)

// 			expectedConsState := substitutePath.EndpointA.GetConsensusState(substituteWasmClientState.GetLatestHeight())
// 			expectedProcessedTime, found := ibctm.GetProcessedTime(substituteClientStore, substituteWasmClientState.GetLatestHeight())
// 			suite.Require().True(found)
// 			expectedProcessedHeight, found := GetProcessedHeight(substituteClientStore, substituteWasmClientState.GetLatestHeight())
// 			suite.Require().True(found)
// 			expectedIterationKey := ibctm.GetIterationKey(substituteClientStore, substituteWasmClientState.GetLatestHeight())

// 			err = subjectWasmClientState.CheckSubstituteAndUpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteWasmClientState)

// 			if tc.expPass {
// 				suite.Require().NoError(err)

// 				updatedWasmClient := subjectPath.EndpointA.GetClientState().(*types.ClientState)
// 				var clientStateData exported.ClientState
// 				err := suite.chainA.Codec.UnmarshalInterface(updatedWasmClient.Data, &clientStateData)
// 				suite.Require().NoError(err)
// 				updatedTmClientState := clientStateData.(*ibctm.ClientState)
// 				suite.Require().Equal(clienttypes.ZeroHeight(), updatedTmClientState.FrozenHeight)

// 				subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)

// 				// check that the correct consensus state was copied over
// 				suite.Require().Equal(substituteWasmClientState.GetLatestHeight(), updatedWasmClient.GetLatestHeight())
// 				subjectConsState := subjectPath.EndpointA.GetConsensusState(updatedWasmClient.GetLatestHeight())
// 				subjectProcessedTime, found := ibctm.GetProcessedTime(subjectClientStore, updatedWasmClient.GetLatestHeight())
// 				suite.Require().True(found)
// 				subjectProcessedHeight, found := GetProcessedHeight(subjectClientStore, updatedWasmClient.GetLatestHeight())
// 				suite.Require().True(found)
// 				subjectIterationKey := ibctm.GetIterationKey(subjectClientStore, updatedWasmClient.GetLatestHeight())

// 				suite.Require().Equal(expectedConsState, subjectConsState)
// 				suite.Require().Equal(expectedProcessedTime, subjectProcessedTime)
// 				suite.Require().Equal(expectedProcessedHeight, subjectProcessedHeight)
// 				suite.Require().Equal(expectedIterationKey, subjectIterationKey)

// 				suite.Require().Equal(newChainID, updatedTmClientState.ChainId)
// 				suite.Require().Equal(time.Hour*24*7, updatedTmClientState.TrustingPeriod)
// 			} else {
// 				suite.Require().Error(err)
// 			}
// 		})
// 	}
// }

func GetProcessedHeight(clientStore storetypes.KVStore, height exported.Height) (uint64, bool) {
	key := ibctm.ProcessedHeightKey(height)
	bz := clientStore.Get(key)
	if len(bz) == 0 {
		return 0, false
	}

	return sdk.BigEndianToUint64(bz), true
}
