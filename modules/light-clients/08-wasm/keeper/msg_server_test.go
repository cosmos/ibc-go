package keeper_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestMsgStoreCode() {
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

				_, err := GetSimApp(s.chainA).WasmClientKeeper.StoreCode(s.chainA.GetContext(), msg)
				s.Require().NoError(err)
			},
			types.ErrWasmCodeExists,
		},
		{
			"fails with zero-length wasm code",
			func() {
				msg = types.NewMsgStoreCode(signer, []byte{})
			},
			types.ErrWasmEmptyCode,
		},
		{
			"fails with checksum",
			func() {
				msg = types.NewMsgStoreCode(signer, []byte{0, 1, 3, 4})
			},
			errors.New("Wasm bytes do not start with Wasm magic number"),
		},
		{
			"fails with wasm code too large",
			func() {
				msg = types.NewMsgStoreCode(signer, wasmtesting.CreateMockContract([]byte(ibctesting.GenerateString(uint(types.MaxWasmSize)))))
			},
			types.ErrWasmCodeTooLarge,
		},
		{
			"fails with unauthorized signer",
			func() {
				signer = s.chainA.SenderAccount.GetAddress().String()
				msg = types.NewMsgStoreCode(signer, data)
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: checksum could not be pinned",
			func() {
				msg = types.NewMsgStoreCode(signer, data)

				s.mockVM.PinFn = func(_ wasmvm.Checksum) error {
					return wasmtesting.ErrMockVM
				}
			},
			wasmtesting.ErrMockVM,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			signer = authtypes.NewModuleAddress(govtypes.ModuleName).String()
			data = wasmtesting.Code

			tc.malleate()

			ctx := s.chainA.GetContext()
			res, err := GetSimApp(s.chainA).WasmClientKeeper.StoreCode(ctx, msg)
			events := ctx.EventManager().Events()

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().NotEmpty(res.Checksum)

				// Verify events
				expectedEvents := sdk.Events{
					sdk.NewEvent(
						"store_wasm_code",
						sdk.NewAttribute(types.AttributeKeyWasmChecksum, hex.EncodeToString(res.Checksum)),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
					),
				}

				for _, evt := range expectedEvents {
					s.Require().Contains(events, evt)
				}
			} else {
				s.Require().Contains(err.Error(), tc.expError.Error())
				s.Require().Nil(res)
				s.Require().Empty(events)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgMigrateContract() {
	oldChecksum, err := types.CreateChecksum(wasmtesting.Code)
	s.Require().NoError(err)

	newByteCode := wasmtesting.CreateMockContract([]byte("MockByteCode-TestMsgMigrateContract"))

	govAcc := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	var (
		newChecksum    []byte
		msg            *types.MsgMigrateContract
		expClientState *types.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: no update to client state",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newChecksum, []byte("{}"))

				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			nil,
		},
		{
			"success: update client state",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newChecksum, []byte("{}"))

				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					// the checksum written in the client state will later be overwritten by the message server.
					expClientStateBz := wasmtesting.CreateMockClientStateBz(s.chainA.App.AppCodec(), []byte("invalid checksum"))
					var ok bool
					expClientState, ok = clienttypes.MustUnmarshalClientState(s.chainA.App.AppCodec(), expClientStateBz).(*types.ClientState)
					s.Require().True(ok)
					store.Set(host.ClientStateKey(), expClientStateBz)

					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			nil,
		},
		{
			"failure: same checksum",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, oldChecksum, []byte("{}"))

				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					panic("unreachable")
				}
			},
			types.ErrWasmCodeExists,
		},
		{
			"failure: unauthorized signer",
			func() {
				msg = types.NewMsgMigrateContract(s.chainA.SenderAccount.GetAddress().String(), defaultWasmClientID, newChecksum, []byte("{}"))
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: invalid wasm checksum",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, []byte(ibctesting.InvalidID), []byte("{}"))
			},
			types.ErrWasmChecksumNotFound,
		},
		{
			"failure: invalid client id",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, ibctesting.InvalidID, newChecksum, []byte("{}"))
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: vm returns error",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newChecksum, []byte("{}"))

				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
				}
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newChecksum, []byte("{}"))

				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: incorrect state update",
			func() {
				msg = types.NewMsgMigrateContract(govAcc, defaultWasmClientID, newChecksum, []byte("{}"))

				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					// the checksum written in here will be overwritten
					store.Set(host.ClientStateKey(), []byte("changed client state"))

					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			var ok bool
			s.SetupWasmWithMockVM()

			_ = s.storeWasmCode(wasmtesting.Code)
			newChecksum = s.storeWasmCode(newByteCode)

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)

			// this is the old client state
			expClientState, ok = endpoint.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			tc.malleate()

			ctx := s.chainA.GetContext()
			res, err := GetSimApp(s.chainA).WasmClientKeeper.MigrateContract(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			if tc.expError == nil {
				expClientState.Checksum = newChecksum

				s.Require().NoError(err)
				s.Require().NotNil(res)

				// updated client state
				clientState, ok := endpoint.GetClientState().(*types.ClientState)
				s.Require().True(ok)

				s.Require().Equal(expClientState, clientState)

				// Verify events
				expectedEvents := sdk.Events{
					sdk.NewEvent(
						"migrate_contract",
						sdk.NewAttribute(types.AttributeKeyClientID, defaultWasmClientID),
						sdk.NewAttribute(types.AttributeKeyWasmChecksum, hex.EncodeToString(oldChecksum)),
						sdk.NewAttribute(types.AttributeKeyNewChecksum, hex.EncodeToString(newChecksum)),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
					),
				}.ToABCIEvents()

				for _, evt := range expectedEvents {
					s.Require().Contains(events, evt)
				}
			} else {
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Nil(res)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgRemoveChecksum() {
	checksum, err := types.CreateChecksum(wasmtesting.Code)
	s.Require().NoError(err)

	govAcc := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	var (
		msg          *types.MsgRemoveChecksum
		expChecksums []types.Checksum
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				msg = types.NewMsgRemoveChecksum(govAcc, checksum)

				expChecksums = []types.Checksum{}
			},
			nil,
		},
		{
			"success: many checksums",
			func() {
				msg = types.NewMsgRemoveChecksum(govAcc, checksum)

				expChecksums = []types.Checksum{}

				for i := range 20 {
					mockCode := wasmtesting.CreateMockContract([]byte{byte(i)})
					checksum, err := types.CreateChecksum(mockCode)
					s.Require().NoError(err)

					keeper := GetSimApp(s.chainA).WasmClientKeeper
					err = keeper.GetChecksums().Set(s.chainA.GetContext(), checksum)
					s.Require().NoError(err)

					expChecksums = append(expChecksums, checksum)
				}
			},
			nil,
		},
		{
			"failure: checksum is missing",
			func() {
				msg = types.NewMsgRemoveChecksum(govAcc, []byte{1})
			},
			types.ErrWasmChecksumNotFound,
		},
		{
			"failure: unauthorized signer",
			func() {
				msg = types.NewMsgRemoveChecksum(s.chainA.SenderAccount.GetAddress().String(), checksum)
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: code has could not be unpinned",
			func() {
				msg = types.NewMsgRemoveChecksum(govAcc, checksum)

				s.mockVM.UnpinFn = func(_ wasmvm.Checksum) error {
					return wasmtesting.ErrMockVM
				}
			},
			wasmtesting.ErrMockVM,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			_ = s.storeWasmCode(wasmtesting.Code)

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)

			tc.malleate()

			ctx := s.chainA.GetContext()
			res, err := GetSimApp(s.chainA).WasmClientKeeper.RemoveChecksum(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)

				checksums, err := GetSimApp(s.chainA).WasmClientKeeper.GetAllChecksums(s.chainA.GetContext())
				s.Require().NoError(err)

				// Check equality of checksums up to order
				s.Require().ElementsMatch(expChecksums, checksums)

				// Verify events
				s.Require().Len(events, 0)
			} else {
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Nil(res)
			}
		})
	}
}
