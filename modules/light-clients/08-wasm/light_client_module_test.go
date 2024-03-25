package wasm_test

import (
	"encoding/json"
	"fmt"
	"time"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	tmClientID   = "07-tendermint-0"
	wasmClientID = "08-wasm-0"
	// Used for checks where look ups for valid client id should fail.
	unusedWasmClientID = "08-wasm-100"
)

func (suite *WasmTestSuite) TestStatus() {
	var clientID string

	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{
			"success",
			func() {},
			exported.Active,
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			status := lightClientModule.Status(suite.chainA.GetContext(), clientID)
			suite.Require().Equal(tc.expStatus, status)
		})
	}
}

func (suite *WasmTestSuite) TestTimestampAtHeight() {
	var clientID string
	expectedTimestamp := uint64(time.Now().UnixNano())

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					var payload types.QueryMsg
					err := json.Unmarshal(queryMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.TimestampAtHeight)
					suite.Require().Nil(payload.CheckForMisbehaviour)
					suite.Require().Nil(payload.Status)
					suite.Require().Nil(payload.VerifyClientMessage)

					resp, err := json.Marshal(types.TimestampAtHeightResult{Timestamp: expectedTimestamp})
					suite.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			clientState, ok := endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			height := clientState.LatestHeight

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			timestamp, err := lightClientModule.TimestampAtHeight(suite.chainA.GetContext(), clientID, height)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expectedTimestamp, timestamp)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
				suite.Require().Equal(uint64(0), timestamp)
			}
		})
	}
}

func (suite *WasmTestSuite) TestInitialize() {
	var (
		consensusState exported.ConsensusState
		clientState    exported.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: new mock client",
			func() {},
			nil,
		},

		{
			"failure: cannot unmarshal client state",
			func() {
				clientState = &solomachine.ClientState{Sequence: 20}
			},
			fmt.Errorf("proto: wrong wireType = 0 for field Data"),
		},
		{
			"failure: client state is invalid",
			func() {
				clientState = &types.ClientState{}
			},
			types.ErrInvalidData,
		},
		{
			"failure: invalid consensus state",
			func() {
				// set upgraded consensus state to solomachine consensus state
				consensusState = &solomachine.ConsensusState{}
			},
			types.ErrInvalidData,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			wrappedClientStateBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wasmtesting.MockTendermitClientState)
			wrappedClientConsensusStateBz := clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientConsensusState)
			clientState = types.NewClientState(wrappedClientStateBz, suite.checksum, wasmtesting.MockTendermitClientState.LatestHeight)
			consensusState = types.NewConsensusState(wrappedClientConsensusStateBz)

			clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.chainA.GetContext(), clientState.ClientType())

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			// Marshal client state and consensus state:
			clientStateBz := suite.chainA.Codec.MustMarshal(clientState)
			consensusStateBz := suite.chainA.Codec.MustMarshal(consensusState)

			err := lightClientModule.Initialize(suite.chainA.GetContext(), clientID, clientStateBz, consensusStateBz)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorContains(err, tc.expError.Error())
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyMembership() {
	var (
		clientState      *types.ClientState
		expClientStateBz []byte
		path             exported.Path
		proof            []byte
		proofHeight      exported.Height
		value            []byte
		clientID         string
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expClientStateBz = GetSimApp(suite.chainA).GetIBCKeeper().ClientKeeper.MustMarshalClientState(clientState)
				suite.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			var ok bool
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			path = commitmenttypes.NewMerklePath("/ibc/key/path")
			proof = wasmtesting.MockValidProofBz
			proofHeight = clienttypes.NewHeight(0, 1)
			value = []byte("value")

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)
			clientState, ok = endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyMembership(suite.chainA.GetContext(), clientID, proofHeight, 0, 0, proof, path, value)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "unexpected error in VerifyMembership")
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyNonMembership() {
	var (
		clientState      *types.ClientState
		expClientStateBz []byte
		path             exported.Path
		proof            []byte
		proofHeight      exported.Height
		clientID         string
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expClientStateBz = GetSimApp(suite.chainA).GetIBCKeeper().ClientKeeper.MustMarshalClientState(clientState)
				suite.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyNonMembership.Height)
					suite.Require().Equal(path, payload.VerifyNonMembership.Path)
					suite.Require().Equal(proof, payload.VerifyNonMembership.Proof)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			var ok bool
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			path = commitmenttypes.NewMerklePath("/ibc/key/path")
			proof = wasmtesting.MockInvalidProofBz
			proofHeight = clienttypes.NewHeight(0, 1)

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)
			clientState, ok = endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyNonMembership(suite.chainA.GetContext(), clientID, proofHeight, 0, 0, proof, path)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "unexpected error in VerifyNonMembership")
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyClientMessage() {
	var (
		clientMsg exported.ClientMessage
		clientID  string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: valid misbehaviour",
			func() {
				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					var msg *types.QueryMsg

					err := json.Unmarshal(queryMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.VerifyClientMessage)
					suite.Require().NotNil(msg.VerifyClientMessage.ClientMessage)
					suite.Require().Nil(msg.Status)
					suite.Require().Nil(msg.CheckForMisbehaviour)
					suite.Require().Nil(msg.TimestampAtHeight)

					suite.Require().Equal(env.Contract.Address, wasmClientID)

					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			clientMsg = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientHeader),
			}

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(suite.chainA.GetContext(), clientID, clientMsg)
			// err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMsg)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyUpgradeAndUpdateState() {
	var (
		upgradedClient              exported.ClientState
		upgradedConsState           exported.ConsensusState
		upgradedClientProof         []byte
		upgradedConsensusStateProof []byte
		clientID                    string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: successful upgrade",
			func() {
				suite.mockVM.RegisterSudoCallback(types.VerifyUpgradeAndUpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg

					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					expectedUpgradedClient, ok := upgradedClient.(*types.ClientState)
					suite.Require().True(ok)
					expectedUpgradedConsensus, ok := upgradedConsState.(*types.ConsensusState)
					suite.Require().True(ok)

					// verify payload values
					suite.Require().Equal(expectedUpgradedClient.Data, payload.VerifyUpgradeAndUpdateState.UpgradeClientState)
					suite.Require().Equal(expectedUpgradedConsensus.Data, payload.VerifyUpgradeAndUpdateState.UpgradeConsensusState)
					suite.Require().Equal(upgradedClientProof, payload.VerifyUpgradeAndUpdateState.ProofUpgradeClient)
					suite.Require().Equal(upgradedConsensusStateProof, payload.VerifyUpgradeAndUpdateState.ProofUpgradeConsensusState)

					// verify other Sudo fields are nil
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyMembership)
					suite.Require().Nil(payload.VerifyNonMembership)

					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					// set new client state and consensus state
					wrappedUpgradedClient := clienttypes.MustUnmarshalClientState(suite.chainA.App.AppCodec(), expectedUpgradedClient.Data).(*ibctm.ClientState)
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), upgradedClient))
					store.Set(host.ConsensusStateKey(wrappedUpgradedClient.LatestHeight), clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState))

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"invalid client state",
			func() {
				upgradedClient = &solomachine.ClientState{Sequence: 20}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"invalid height",
			func() {
				upgradedClient = &types.ClientState{LatestHeight: clienttypes.ZeroHeight()}
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			clientState, ok := endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			newLatestHeight := clienttypes.NewHeight(2, 10)
			wrappedUpgradedClient := wasmtesting.CreateMockTendermintClientState(newLatestHeight)
			wrappedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wrappedUpgradedClient)
			upgradedClient = types.NewClientState(wrappedUpgradedClientBz, clientState.Checksum, newLatestHeight)

			wrappedUpgradedConsensus := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), []byte("new-nextValsHash"))
			wrappedUpgradedConsensusBz := clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), wrappedUpgradedConsensus)
			upgradedConsState = types.NewConsensusState(wrappedUpgradedConsensusBz)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), wasmClientID)

			upgradedClientProof = wasmtesting.MockUpgradedClientStateProofBz
			upgradedConsensusStateProof = wasmtesting.MockUpgradedConsensusStateProofBz

			newClient := suite.chainA.Codec.MustMarshal(upgradedClient)
			newConsensusState := suite.chainA.Codec.MustMarshal(upgradedConsState)

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				suite.chainA.GetContext(),
				clientID,
				newClient,
				newConsensusState,
				upgradedClientProof,
				upgradedConsensusStateProof,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// verify new client state and consensus state
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)
				suite.Require().Equal(upgradedClient, clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz))

				consStateBz := clientStore.Get(host.ConsensusStateKey(upgradedClient.(*types.ClientState).LatestHeight))
				suite.Require().NotEmpty(consStateBz)
				suite.Require().Equal(upgradedConsState, clienttypes.MustUnmarshalConsensusState(suite.chainA.Codec, consStateBz))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *WasmTestSuite) TestCheckForMisbehaviour() {
	var (
		clientMessage exported.ClientMessage
		clientID      string
	)

	testCases := []struct {
		name              string
		malleate          func()
		foundMisbehaviour bool
		expPanic          error
	}{
		{
			"success: no misbehaviour",
			func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: false})
					suite.Require().NoError(err)
					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			false,
			nil,
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			false, // not applicable
			fmt.Errorf("08-wasm-100: %s", clienttypes.ErrClientNotFound),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()
			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			clientMessage = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientMisbehaviour),
			}

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			var foundMisbehaviour bool
			foundMisbehaviourFunc := func() {
				foundMisbehaviour = lightClientModule.CheckForMisbehaviour(suite.chainA.GetContext(), clientID, clientMessage)
			}

			if tc.expPanic == nil {
				foundMisbehaviourFunc()
				suite.Require().Equal(tc.foundMisbehaviour, foundMisbehaviour)
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), foundMisbehaviourFunc)
			}
		})
	}
}

func (suite *WasmTestSuite) TestUpdateState() {
	var (
		clientMsg             exported.ClientMessage
		expectedClientStateBz []byte
		clientID              string
	)

	testCases := []struct {
		name       string
		malleate   func()
		expPanic   error
		expHeights []exported.Height
	}{
		{
			"success: no update",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.UpdateState)
					suite.Require().NotNil(msg.UpdateState.ClientMessage)
					suite.Require().Equal(msg.UpdateState.ClientMessage, clienttypes.MustMarshalClientMessage(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientHeader))
					suite.Require().Nil(msg.VerifyMembership)
					suite.Require().Nil(msg.VerifyNonMembership)
					suite.Require().Nil(msg.UpdateStateOnMisbehaviour)
					suite.Require().Nil(msg.VerifyUpgradeAndUpdateState)

					suite.Require().Equal(env.Contract.Address, wasmClientID)

					updateStateResp := types.UpdateStateResult{
						Heights: []clienttypes.Height{},
					}

					resp, err := json.Marshal(updateStateResp)
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			[]exported.Height{},
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			fmt.Errorf("08-wasm-100: %s", clienttypes.ErrClientNotFound),
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM() // reset
			expectedClientStateBz = nil

			clientMsg = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientHeader),
			}

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			var heights []exported.Height
			updateState := func() {
				heights = lightClientModule.UpdateState(suite.chainA.GetContext(), clientID, clientMsg)
			}

			if tc.expPanic == nil {
				updateState()
				suite.Require().Equal(tc.expHeights, heights)

				if expectedClientStateBz != nil {
					clientStateBz := clientStore.Get(host.ClientStateKey())
					suite.Require().Equal(expectedClientStateBz, clientStateBz)
				}
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), updateState)
			}
		})
	}
}

func (suite *WasmTestSuite) TestUpdateStateOnMisbehaviour() {
	var (
		clientMsg             exported.ClientMessage
		expectedClientStateBz []byte
		clientID              string
	)

	testCases := []struct {
		name               string
		malleate           func()
		panicErr           error
		updatedClientState []byte
	}{
		{
			"success: no update",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var msg types.SudoMsg

					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.UpdateStateOnMisbehaviour)
					suite.Require().NotNil(msg.UpdateStateOnMisbehaviour.ClientMessage)
					suite.Require().Nil(msg.UpdateState)
					suite.Require().Nil(msg.UpdateState)
					suite.Require().Nil(msg.VerifyMembership)
					suite.Require().Nil(msg.VerifyNonMembership)
					suite.Require().Nil(msg.VerifyUpgradeAndUpdateState)

					resp, err := json.Marshal(types.EmptyResult{})
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			nil,
		},
		{
			"cannot find subject client state",
			func() {
				clientID = unusedWasmClientID
			},
			fmt.Errorf("08-wasm-100: %s", clienttypes.ErrClientNotFound),
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()
			expectedClientStateBz = nil

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientMsg = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientMisbehaviour),
			}

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			updateFunc := func() {
				lightClientModule.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), clientID, clientMsg)
			}

			if tc.panicErr == nil {
				updateFunc()
				if expectedClientStateBz != nil {
					suite.Require().Equal(expectedClientStateBz, store.Get(host.ClientStateKey()))
				}
			} else {
				suite.Require().PanicsWithError(tc.panicErr.Error(), updateFunc)
			}
		})
	}
}

func (suite *WasmTestSuite) TestRecoverClient() {
	var (
		expectedClientStateBz               []byte
		subjectClientID, substituteClientID string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		// TODO(02-client routing): add successful test when light client module does not call into 08-wasm ClientState
		// {
		// 	"success",
		// 	func() {
		// 	},
		// 	nil,
		// },
		{
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 08-wasm prefix",
			func() {
				substituteClientID = tmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			expectedClientStateBz = nil

			subjectEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := subjectEndpoint.CreateClient()
			suite.Require().NoError(err)
			subjectClientID = subjectEndpoint.ClientID

			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectClientID)

			substituteEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err = substituteEndpoint.CreateClient()
			suite.Require().NoError(err)
			substituteClientID = substituteEndpoint.ClientID

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.RecoverClient(suite.chainA.GetContext(), subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := subjectClientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expectedClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *WasmTestSuite) TestLatestHeight() {
	var clientID string

	testCases := []struct {
		name      string
		malleate  func()
		expHeight clienttypes.Height
	}{
		{
			"success",
			func() {
			},
			clienttypes.NewHeight(1, 5),
		},
		{
			"cannot find substitute client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ZeroHeight(),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			subjectEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := subjectEndpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = subjectEndpoint.ClientID

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			height := lightClientModule.LatestHeight(suite.chainA.GetContext(), clientID)

			suite.Require().Equal(tc.expHeight, height)
		})
	}
}
