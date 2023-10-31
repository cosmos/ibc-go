package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *KeeperTestSuite) TestMsgStoreCode() {
	var (
		msg    *types.MsgStoreCode
		signer string
		data   []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				msg = types.NewMsgStoreCode(signer, data)
			},
			nil,
		},
		{
			"fails with duplicate wasm code",
			func() {
				msg = types.NewMsgStoreCode(signer, data)

				_, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
			},
			types.ErrWasmCodeExists,
		},
		{
			"fails with invalid wasm code",
			func() {
				msg = types.NewMsgStoreCode(signer, []byte{})
			},
			types.ErrWasmEmptyCode,
		},
		{
			"fails with unauthorized signer",
			func() {
				signer = suite.chainA.SenderAccount.GetAddress().String()
				msg = types.NewMsgStoreCode(signer, data)
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			signer = authtypes.NewModuleAddress(govtypes.ModuleName).String()
			data, _ = os.ReadFile("../test_data/ics10_grandpa_cw.wasm.gz")

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(ctx, msg)
			events := ctx.EventManager().Events()

			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotEmpty(res.Checksum)

				// Verify events
				expectedEvents := sdk.Events{
					sdk.NewEvent(
						"store_wasm_code",
						sdk.NewAttribute(types.AttributeKeyWasmCodeHash, hex.EncodeToString(res.Checksum)),
					),
				}

				for _, evt := range expectedEvents {
					suite.Require().Contains(events, evt)
				}
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
				suite.Require().Empty(events)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgMigrateContract() {
	oldCodeHash := sha256.Sum256(wasmtesting.Code)

	mockByteCode := []byte("MockByteCode-TestMsgMigrateContract")

	govAcc := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	var (
		newCodeHash    []byte
		msg            *types.MsgMigrateContract
		expClientState *types.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newCodeHash, []byte("{}"))

				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.Response{Data: data}, wasmtesting.DefaultGasUsed, nil
				}
			},
			nil,
		},
		{
			"success: update state",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newCodeHash, []byte("{}"))

				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					// the code hash written in here will be overwritten
					expClientState = types.NewClientState([]byte{1}, []byte(ibctesting.InvalidID), clienttypes.NewHeight(2000, 2))
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), expClientState))

					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.Response{Data: data}, wasmtesting.DefaultGasUsed, nil
				}
			},
			nil,
		},
		{
			"success: same code hash",
			func() {
				newCodeHash = oldCodeHash[:]
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newCodeHash, []byte("{}"))

				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					panic("unreachable")
				}
			},
			nil,
		},
		{
			"failure: unauthorized signer",
			func() {
				msg = types.NewMsgMigrateContract(suite.chainA.SenderAccount.GetAddress().String(), defaultWasmClientID, newCodeHash, []byte("{}"))
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: invalid wasm code hash",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, []byte(ibctesting.InvalidID), []byte("{}"))
			},
			types.ErrWasmCodeHashNotFound,
		},
		{
			"failure: invalid client id",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, ibctesting.InvalidID, newCodeHash, []byte("{}"))
			},
			types.ErrClientNotFound,
		},
		{
			"failure: contract returns error",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newCodeHash, []byte("{}"))

				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, types.ErrInvalid
				}
			},
			types.ErrInvalid,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			newCodeHash = storeWasmCode(suite, mockByteCode)

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			oldClientState, ok := endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			expClientState = oldClientState

			tc.malleate()

			res, err := GetSimApp(suite.chainA).WasmClientKeeper.MigrateContract(suite.chainA.GetContext(), msg)

			// updated client state
			clientState, ok := endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			if tc.expError == nil {
				expClientState.CodeHash = newCodeHash

				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				suite.Require().Equal(expClientState, clientState)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)

				suite.Require().Equal(expClientState, clientState)
			}
		})
	}
}
