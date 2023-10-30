package types_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	tmtypes "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *TypesTestSuite) TestVerifyHeaderGrandpa() {
	var (
		ok          bool
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"successful verify header", func() {},
			true,
		},
		{
			"unsuccessful verify header: para id mismatch", func() {
				clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_para_id_mismatch"])
				suite.Require().NoError(err)

				clientState = &types.ClientState{
					Data:         clientStateData,
					CodeHash:     suite.codeHash,
					LatestHeight: clienttypes.NewHeight(2000, 39),
				}
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), defaultWasmClientID, clientState)
			},
			false,
		},
		{
			"unsuccessful verify header: head height < consensus height", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header_old"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), defaultWasmClientID)
			suite.Require().True(ok)
			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, defaultWasmClientID)

			data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
			suite.Require().NoError(err)
			clientMsg = &types.ClientMessage{
				Data: data,
			}

			tc.setup()
			err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, store, clientMsg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestUpdateStateGrandpa() {
	var (
		ok          bool
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success with height later than latest height",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
				// VerifyClientMessage must be run first
				err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, defaultWasmClientID), clientMsg)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"success with not verifying client message",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
			},
			true,
		},
		{
			"invalid ClientMessage type", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), defaultWasmClientID)
			suite.Require().True(ok)

			tc.malleate()

			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, defaultWasmClientID)

			if tc.expPass {
				consensusHeights := clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, store, clientMsg)

				clientStateBz := store.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

				suite.Require().Len(consensusHeights, 1)
				suite.Require().Equal(clienttypes.NewHeight(2000, 47), consensusHeights[0])
				suite.Require().Equal(consensusHeights[0], newClientState.(*types.ClientState).LatestHeight)
			} else {
				suite.Require().Panics(func() {
					clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, store, clientMsg)
				})
			}
		})
	}
}

func (suite *TypesTestSuite) TestUpdateState() {
	mockHeight := clienttypes.NewHeight(1, 1)

	var (
		clientMsg   exported.ClientMessage
		clientStore storetypes.KVStore
	)

	testCases := []struct {
		name           string
		malleate       func()
		expPanic       error
		expHeights     []exported.Height
		expClientState []byte
	}{
		{
			"success: no update",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.UpdateState)
					suite.Require().NotNil(msg.UpdateState.ClientMessage)
					suite.Require().Equal(msg.UpdateState.ClientMessage.Data, wasmtesting.MockClientStateBz)
					suite.Require().Nil(msg.VerifyMembership)
					suite.Require().Nil(msg.VerifyNonMembership)
					suite.Require().Nil(msg.UpdateStateOnMisbehaviour)
					suite.Require().Nil(msg.VerifyUpgradeAndUpdateState)

					suite.Require().Equal(env.Contract.Address, defaultWasmClientID)

					updateStateResp := types.UpdateStateResult{
						Heights: []clienttypes.Height{},
					}

					resp, err := json.Marshal(updateStateResp)
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.Response{
						Data: resp,
					}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			[]exported.Height{},
			nil,
		},
		{
			"success: update client",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					store.Set(host.ClientStateKey(), msg.UpdateState.ClientMessage.Data)
					updateStateResp := types.UpdateStateResult{
						Heights: []clienttypes.Height{mockHeight},
					}

					resp, err := json.Marshal(updateStateResp)
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.Response{
						Data: resp,
					}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			[]exported.Height{mockHeight},
			wasmtesting.MockClientStateBz,
		},
		{
			"failure: clientStore prefix does not include clientID",
			func() {
				clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, ibctesting.InvalidID)
			},
			errorsmod.Wrap(types.ErrWasmContractCallFailed, errorsmod.Wrap(errorsmod.Wrapf(types.ErrRetrieveClientID, "prefix does not contain a %s clientID", exported.Wasm), "failed to retrieve clientID for wasm contract call").Error()),
			nil,
			nil,
		},
		{
			"failure: invalid ClientMessage type",
			func() {
				// SudoCallback left nil because clientMsg is checked by 08-wasm before callbackFn is called.
				clientMsg = &tmtypes.Misbehaviour{}
			},
			fmt.Errorf("expected type %T, got %T", (*types.ClientMessage)(nil), (*tmtypes.Misbehaviour)(nil)),
			nil,
			nil,
		},
		{
			"failure: callbackFn returns error",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			errorsmod.Wrap(types.ErrWasmContractCallFailed, wasmtesting.ErrMockContract.Error()),
			nil,
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM() // reset

			clientMsg = &types.ClientMessage{
				Data: wasmtesting.MockClientStateBz,
			}

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)

			tc.malleate()

			clientState := endpoint.GetClientState()

			var heights []exported.Height
			updateState := func() {
				heights = clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, clientStore, clientMsg)
			}

			if tc.expPanic == nil {
				updateState()
				suite.Require().Equal(tc.expHeights, heights)

				if tc.expClientState != nil {
					clientStateBz := clientStore.Get(host.ClientStateKey())
					suite.Require().Equal(tc.expClientState, clientStateBz)
				}
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), updateState)
			}
		})
	}
}

func (suite *TypesTestSuite) TestUpdateStateOnMisbehaviour() {
	var clientMsg exported.ClientMessage

	testCases := []struct {
		name               string
		malleate           func()
		panicErr           error
		updatedClientState []byte
	}{
		{
			"success: no update",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
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

					return &wasmvmtypes.Response{
						Data: resp,
					}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			nil,
		},
		{
			"success: client state updated on valid misbehaviour",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					// set new client state in store
					store.Set(host.ClientStateKey(), msg.UpdateStateOnMisbehaviour.ClientMessage.Data)
					resp, err := json.Marshal(types.EmptyResult{})
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.Response{Data: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			wasmtesting.MockClientStateBz,
		},
		{
			"failure: invalid client message",
			func() {
				clientMsg = &tmtypes.Header{}
				// we will not register the callback here because this test case does not reach the VM
			},
			fmt.Errorf("expected type %T, got %T", (*types.ClientMessage)(nil), (*tmtypes.Header)(nil)),
			nil,
		},
		{
			"failure: err return from contract vm",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			errorsmod.Wrap(types.ErrWasmContractCallFailed, wasmtesting.ErrMockContract.Error()),
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientMsg = &types.ClientMessage{
				Data: wasmtesting.MockClientStateBz,
			}
			clientState := endpoint.GetClientState()

			tc.malleate()

			if tc.panicErr == nil {
				clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), store, clientMsg)
				if tc.updatedClientState != nil {
					suite.Require().Equal(tc.updatedClientState, store.Get(host.ClientStateKey()))
				}
			} else {
				suite.Require().PanicsWithError(tc.panicErr.Error(), func() {
					clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), store, clientMsg)
				})
			}
		})
	}
}
