package types_test

import (
	"encoding/json"
	"errors"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	storetypes "cosmossdk.io/store/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *TypesTestSuite) TestVerifyClientMessage() {
	var (
		clientMsg   exported.ClientMessage
		clientStore storetypes.KVStore
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: valid misbehaviour",
			func() {
				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					var msg *types.QueryMsg

					err := json.Unmarshal(queryMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.VerifyClientMessage)
					suite.Require().NotNil(msg.VerifyClientMessage.ClientMessage)
					suite.Require().Nil(msg.Status)
					suite.Require().Nil(msg.CheckForMisbehaviour)
					suite.Require().Nil(msg.TimestampAtHeight)
					suite.Require().Nil(msg.ExportMetadata)

					suite.Require().Equal(env.Contract.Address, defaultWasmClientID)

					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return resp, types.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: clientStore prefix does not include clientID",
			func() {
				clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, ibctesting.InvalidID)
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: invalid client message",
			func() {
				clientMsg = &ibctmtypes.Header{}

				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return resp, types.DefaultGasUsed, nil
				})
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: error return from contract vm",
			func() {
				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			wasmtesting.ErrMockContract,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, endpoint.ClientID)
			clientState := endpoint.GetClientState()

			clientMsg = &types.ClientMessage{
				Data: []byte{1},
			}

			tc.malleate()

			err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMsg)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *TypesTestSuite) TestCheckForMisbehaviour() {
	var clientMessage exported.ClientMessage

	testCases := []struct {
		name                 string
		malleate             func()
		expFoundMisbehaviour bool
	}{
		{
			"success: no misbehaviour",
			func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: false})
					suite.Require().NoError(err)
					return resp, types.DefaultGasUsed, nil
				})
			},
			false,
		},
		{
			"success: misbehaviour found", func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: true})
					suite.Require().NoError(err)
					return resp, types.DefaultGasUsed, nil
				})
			},
			true,
		},
		{
			"success: contract error, resp cannot be marshalled", func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp := "cannot be unmarshalled"
					return []byte(resp), types.DefaultGasUsed, nil
				})
			},
			false,
		},
		{
			"success: vm returns error, ", func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return nil, 0, errors.New("invalid block ID")
				})
			},
			false,
		},
		{
			"success: invalid client message", func() {
				clientMessage = &ibctmtypes.Header{}
				// we will not register the callback here because this test case does not reach the VM
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()
			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState := endpoint.GetClientState()
			clientMessage = &types.ClientMessage{
				Data: []byte{1},
			}

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, endpoint.ClientID)

			tc.malleate()

			foundMisbehaviour := clientState.CheckForMisbehaviour(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMessage)
			suite.Require().Equal(tc.expFoundMisbehaviour, foundMisbehaviour)
		})
	}
}
