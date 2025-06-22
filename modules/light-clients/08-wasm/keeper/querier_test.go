package keeper_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/keeper"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

type CustomQuery struct {
	Echo *QueryEcho `json:"echo,omitempty"`
}

type QueryEcho struct {
	Data string `json:"data"`
}

func mockCustomQuerier() func(sdk.Context, json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var customQuery CustomQuery
		err := json.Unmarshal([]byte(request), &customQuery)
		if err != nil {
			return nil, wasmtesting.ErrMockContract
		}

		if customQuery.Echo != nil {
			data, err := json.Marshal(customQuery.Echo.Data)
			return data, err
		}

		return nil, wasmtesting.ErrMockContract
	}
}

func (s *KeeperTestSuite) TestCustomQuery() {
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: custom query",
			func() {
				querierPlugin := keeper.QueryPlugins{
					Custom: mockCustomQuerier(),
				}

				GetSimApp(s.chainA).WasmClientKeeper.SetQueryPlugins(querierPlugin)
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					echo := CustomQuery{
						Echo: &QueryEcho{
							Data: "hello world",
						},
					}
					echoJSON, err := json.Marshal(echo)
					s.Require().NoError(err)

					resp, err := querier.Query(wasmvmtypes.QueryRequest{
						Custom: json.RawMessage(echoJSON),
					}, math.MaxUint64)
					s.Require().NoError(err)

					var respData string
					err = json.Unmarshal(resp, &respData)
					s.Require().NoError(err)
					s.Require().Equal("hello world", respData)

					resp, err = json.Marshal(types.StatusResult{Status: exported.Active.String()})
					s.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: default query",
			func() {
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := querier.Query(wasmvmtypes.QueryRequest{Custom: json.RawMessage("{}")}, math.MaxUint64)
					s.Require().ErrorIs(err, wasmvmtypes.UnsupportedRequest{Kind: "Custom queries are not allowed"})
					s.Require().Nil(resp)

					return nil, wasmtesting.DefaultGasUsed, err
				})
			},
			types.ErrVMError,
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

			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper

			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), endpoint.ClientID)
			clientState, ok := endpoint.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			res, err := wasmClientKeeper.WasmQuery(s.chainA.GetContext(), endpoint.ClientID, clientStore, clientState, types.QueryMsg{Status: &types.StatusMsg{}})

			if tc.expError == nil {
				s.Require().Nil(err)
				s.Require().NotNil(res)
			} else {
				s.Require().Nil(res)
				s.Require().ErrorIs(err, tc.expError)
			}

			// reset query plugins after each test
			wasmClientKeeper.SetQueryPlugins(keeper.NewDefaultQueryPlugins(GetSimApp(s.chainA).GRPCQueryRouter()))
		})
	}
}

func (s *KeeperTestSuite) TestStargateQuery() {
	typeURL := "/ibc.lightclients.wasm.v1.Query/Checksums"

	var (
		endpoint          *wasmtesting.WasmEndpoint
		checksum          []byte
		expDiscardedState = false
		proofKey          = []byte("mock-key")
		testKey           = []byte("test-key")
		value             = []byte("mock-value")
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: custom query",
			func() {
				querierPlugin := keeper.QueryPlugins{
					Stargate: keeper.AcceptListStargateQuerier([]string{typeURL}, GetSimApp(s.chainA).GRPCQueryRouter()),
				}

				GetSimApp(s.chainA).WasmClientKeeper.SetQueryPlugins(querierPlugin)

				s.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					queryRequest := types.QueryChecksumsRequest{}
					bz, err := queryRequest.Marshal()
					s.Require().NoError(err)

					resp, err := querier.Query(wasmvmtypes.QueryRequest{
						Stargate: &wasmvmtypes.StargateQuery{
							Path: typeURL,
							Data: bz,
						},
					}, math.MaxUint64)
					s.Require().NoError(err)

					var respData types.QueryChecksumsResponse
					err = respData.Unmarshal(resp)
					s.Require().NoError(err)

					expChecksum := hex.EncodeToString(checksum)

					s.Require().Len(respData.Checksums, 1)
					s.Require().Equal(expChecksum, respData.Checksums[0])

					store.Set(testKey, value)

					result, err := json.Marshal(types.TimestampAtHeightResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: result}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			// The following test sets a mock proof key and value in the ibc store and registers a query callback on the Status msg.
			// The Status handler will then perform the QueryVerifyMembershipRequest using the wasmvm.Querier.
			// As the VerifyMembership query rpc will call directly back into the same client, we also register a callback for VerifyMembership.
			// Here we decode the proof and verify the mock proof key and value are set in the ibc store.
			// This exercises the full flow through the grpc handler and into the light client for verification, handling encoding and routing.
			// Furthermore we write a test key and assert that the state changes made by this handler were discarded by the cachedCtx at the grpc handler.
			"success: verify membership query",
			func() {
				querierPlugin := keeper.QueryPlugins{
					Stargate: keeper.AcceptListStargateQuerier([]string{""}, GetSimApp(s.chainA).GRPCQueryRouter()),
				}

				GetSimApp(s.chainA).WasmClientKeeper.SetQueryPlugins(querierPlugin)

				store := s.chainA.GetContext().KVStore(GetSimApp(s.chainA).GetKey(exported.StoreKey))
				store.Set(proofKey, value)

				s.coordinator.CommitBlock(s.chainA)
				proof, proofHeight := endpoint.QueryProofAtHeight(proofKey, uint64(s.chainA.GetContext().BlockHeight()))

				merklePath := commitmenttypes.NewMerklePath(proofKey)
				merklePath, err := commitmenttypes.ApplyPrefix(s.chainA.GetPrefix(), merklePath)
				s.Require().NoError(err)

				s.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					queryRequest := clienttypes.QueryVerifyMembershipRequest{
						ClientId:    endpoint.ClientID,
						Proof:       proof,
						ProofHeight: proofHeight,
						MerklePath:  merklePath,
						Value:       value,
					}

					bz, err := queryRequest.Marshal()
					s.Require().NoError(err)

					resp, err := querier.Query(wasmvmtypes.QueryRequest{
						Stargate: &wasmvmtypes.StargateQuery{
							Path: "/ibc.core.client.v1.Query/VerifyMembership",
							Data: bz,
						},
					}, math.MaxUint64)
					s.Require().NoError(err)

					var respData clienttypes.QueryVerifyMembershipResponse
					err = respData.Unmarshal(resp)
					s.Require().NoError(err)

					s.Require().True(respData.Success)

					result, err := json.Marshal(types.TimestampAtHeightResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: result}, wasmtesting.DefaultGasUsed, nil
				})

				s.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					s.Require().NoError(err)

					var merkleProof commitmenttypes.MerkleProof
					err = s.chainA.Codec.Unmarshal(payload.VerifyMembership.Proof, &merkleProof)
					s.Require().NoError(err)

					root := commitmenttypes.NewMerkleRoot(s.chainA.App.LastCommitID().Hash)
					err = merkleProof.VerifyMembership(commitmenttypes.GetSDKSpecs(), root, merklePath, payload.VerifyMembership.Value)
					s.Require().NoError(err)

					bz, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					expDiscardedState = true
					store.Set(testKey, value)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: default querier",
			func() {
				s.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					queryRequest := types.QueryChecksumsRequest{}
					bz, err := queryRequest.Marshal()
					s.Require().NoError(err)

					resp, err := querier.Query(wasmvmtypes.QueryRequest{
						Stargate: &wasmvmtypes.StargateQuery{
							Path: typeURL,
							Data: bz,
						},
					}, math.MaxUint64)
					s.Require().ErrorIs(err, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", typeURL)})
					s.Require().Nil(resp)

					store.Set(testKey, value)

					return nil, wasmtesting.DefaultGasUsed, err
				})
			},
			wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", typeURL)},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			expDiscardedState = false
			s.SetupWasmWithMockVM()
			checksum = s.storeWasmCode(wasmtesting.Code)

			endpoint = wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)

			tc.malleate()

			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper

			payload := types.QueryMsg{
				TimestampAtHeight: &types.TimestampAtHeightMsg{
					Height: clienttypes.NewHeight(1, 100),
				},
			}

			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), endpoint.ClientID)
			clientState, ok := endpoint.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			// NOTE: we register query callbacks against: types.TimestampAtHeightMsg{}
			// in practise, this can against any client state msg, however registering against types.StatusMsg{} introduces recursive loops
			// due to test case: "success: verify membership query"
			res, err := wasmClientKeeper.WasmQuery(s.chainA.GetContext(), endpoint.ClientID, clientStore, clientState, payload)

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
			} else {
				s.Require().Nil(res)
				// use error contains as wasmvm errors do not implement errors.Is method
				s.Require().ErrorContains(err, tc.expError.Error())
			}

			clientStore = s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), endpoint.ClientID)
			if expDiscardedState {
				s.Require().False(clientStore.Has(testKey))
			} else {
				s.Require().True(clientStore.Has(testKey))
			}

			// reset query plugins after each test
			wasmClientKeeper.SetQueryPlugins(keeper.NewDefaultQueryPlugins(GetSimApp(s.chainA).GRPCQueryRouter()))
		})
	}
}
