package types_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

func (suite *TypesTestSuite) TestMigrateContract() {
	var (
		oldHash        [32]byte
		newHash        [32]byte
		payload        []byte
		expClientState *types.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: new and old code hash are different",
			func() {
				newHash = sha256.Sum256([]byte{1, 2, 3})
				err := ibcwasm.CodeHashes.Set(suite.chainA.GetContext(), newHash[:])
				suite.Require().NoError(err)

				payload = []byte{1}
				expChecksum := wasmvmtypes.ForceNewChecksum(hex.EncodeToString(newHash[:]))

				suite.mockVM.MigrateFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					suite.Require().Equal(expChecksum, codeID)
					suite.Require().Equal(defaultWasmClientID, env.Contract.Address)
					suite.Require().Equal(payload, msg)

					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.Response{Data: data}, wasmtesting.DefaultGasUsed, nil
				}
			},
			nil,
		},
		{
			"success: update client state",
			func() {
				newHash = sha256.Sum256([]byte{1, 2, 3})
				err := ibcwasm.CodeHashes.Set(suite.chainA.GetContext(), newHash[:])
				suite.Require().NoError(err)

				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					expClientState = types.NewClientState([]byte{1}, newHash[:], clienttypes.NewHeight(2000, 2))
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), expClientState))

					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.Response{Data: data}, wasmtesting.DefaultGasUsed, nil
				}
			},
			nil,
		},
		{
			"failure: new and old code hash are the same",
			func() {
				// this should not be called
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					panic("unreachable")
				}
			},
			types.ErrWasmCodeExists,
		},
		{
			"failure: code hash not found",
			func() {
				newHash = sha256.Sum256([]byte{1, 2, 3})
			},
			types.ErrWasmCodeHashNotFound,
		},
		{
			"failure: contract returns error",
			func() {
				newHash = sha256.Sum256([]byte{1, 2, 3})
				err := ibcwasm.CodeHashes.Set(suite.chainA.GetContext(), newHash[:])
				suite.Require().NoError(err)

				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockContract
				}
			},
			wasmtesting.ErrMockContract,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			oldHash = sha256.Sum256(wasmtesting.Code)
			newHash = oldHash

			endpointA := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpointA.CreateClient()
			suite.Require().NoError(err)

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpointA.ClientID)
			clientState, ok := endpointA.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			expClientState = clientState

			tc.malleate()

			err = clientState.MigrateContract(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, endpointA.ClientID, newHash[:], payload)

			// updated client state
			clientState, ok = endpointA.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expClientState, clientState)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
