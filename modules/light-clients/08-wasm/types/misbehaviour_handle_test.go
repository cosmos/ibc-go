package types_test

import (
	"encoding/json"
	fmt "fmt"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	errorsmod "cosmossdk.io/errors"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

func (suite *TypesTestSuite) TestVerifyClientMessage() {
	var clientMsg exported.ClientMessage

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: valid misbehaviour",
			func() {
				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return resp, types.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: invalid client message",
			func() {
				clientMsg = &ibctmtypes.Header{}

				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return resp, types.DefaultGasUsed, nil
				})
			},
			errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected type: %T, got: %T", exported.ClientMessage(&types.ClientMessage{}), &ibctmtypes.Header{}),
		},
		{
			"failure: error return from contract vm",
			func() {
				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return nil, 0, fmt.Errorf("callbackFn error")
				})
			},
			errorsmod.Wrapf(fmt.Errorf("callbackFn error"), "query to wasm contract failed"),
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

			clientMsg = &types.ClientMessage{
				Data: []byte{1},
			}

			tc.malleate()

			err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.App.AppCodec(), suite.store, clientMsg)

			if tc.expErr != nil {
				suite.Require().ErrorIs(err, tc.expErr)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

/* func (suite *TypesTestSuite) TestCheckForMisbehaviourGrandpa() {
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
			"valid update no misbehaviour",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}

				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"valid fork misbehaviour returns true",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
				// VerifyClientMessage must be run first
				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
				clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

				// Reset client state to the previous for the test
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, defaultWasmClientID, clientState)

				data, err = base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}

				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"invalid wasm misbehaviour",
			func() {
				clientMsg = &solomachine.Misbehaviour{}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, defaultWasmClientID)
			suite.Require().True(ok)

			tc.malleate()

			foundMisbehaviour := clientState.CheckForMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().True(foundMisbehaviour)
			} else {
				suite.Require().False(foundMisbehaviour)
			}
		})
	}
}*/

// func (suite *TypesTestSuite) TestCheckForMisbehaviourTendermint() {
// 	var (
// 		path          *ibctesting.Path
// 		clientMessage exported.ClientMessage
// 	)

// 	testCases := []struct {
// 		name     string
// 		malleate func()
// 		expPass  bool
// 	}{
// 		{
// 			"valid update no misbehaviour",
// 			func() {},
// 			false,
// 		},
// 		{
// 			"consensus state already exists, already updated",
// 			func() {
// 				wasmHeader, ok := clientMessage.(*types.ClientMessage)
// 				suite.Require().True(ok)

// 				var wasmData exported.ClientMessage
// 				err := suite.chainA.Codec.UnmarshalInterface(wasmHeader.Data, &wasmData)
// 				suite.Require().NoError(err)

// 				tmHeader, ok := wasmData.(*ibctm.Header)
// 				suite.Require().True(ok)

// 				tmConsensusState := &ibctm.ConsensusState{
// 					Timestamp:          tmHeader.GetTime(),
// 					Root:               commitmenttypes.NewMerkleRoot(tmHeader.Header.GetAppHash()),
// 					NextValidatorsHash: tmHeader.Header.NextValidatorsHash,
// 				}

// 				tmConsensusStateData, err := suite.chainA.Codec.MarshalInterface(tmConsensusState)
// 				suite.Require().NoError(err)
// 				wasmConsensusState := &types.ConsensusState{
// 					Data: tmConsensusStateData,
// 				}

// 				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(
// 					suite.chainA.GetContext(),
// 					path.EndpointA.ClientID,
// 					tmHeader.GetHeight(),
// 					wasmConsensusState,
// 				)
// 			},
// 			false,
// 		},
// 		{
// 			"invalid fork misbehaviour: identical headers", func() {
// 				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

// 				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
// 				suite.Require().True(found)

// 				err := path.EndpointA.UpdateClient()
// 				suite.Require().NoError(err)

// 				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

// 				misbehaviourHeader := suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
// 				tmMisbehaviour := &ibctm.Misbehaviour{
// 					Header1: misbehaviourHeader,
// 					Header2: misbehaviourHeader,
// 				}
// 				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
// 				suite.Require().NoError(err)
// 				clientMessage = &types.ClientMessage{
// 					Data: wasmData,
// 				}
// 			}, false,
// 		},
// 		{
// 			"invalid time misbehaviour: monotonically increasing time", func() {
// 				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

// 				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
// 				suite.Require().True(found)

// 				header1 := suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
// 				header2 := suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

// 				tmMisbehaviour := &ibctm.Misbehaviour{
// 					Header1: header1,
// 					Header2: header2,
// 				}
// 				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
// 				suite.Require().NoError(err)
// 				clientMessage = &types.ClientMessage{
// 					Data: wasmData,
// 				}
// 			}, false,
// 		},
// 		{
// 			"consensus state already exists, app hash mismatch",
// 			func() {
// 				wasmHeader, ok := clientMessage.(*types.ClientMessage)
// 				suite.Require().True(ok)

// 				var wasmData exported.ClientMessage
// 				err := suite.chainA.Codec.UnmarshalInterface(wasmHeader.Data, &wasmData)
// 				suite.Require().NoError(err)

// 				tmHeader, ok := wasmData.(*ibctm.Header)
// 				suite.Require().True(ok)

// 				tmConsensusState := &ibctm.ConsensusState{
// 					Timestamp:          tmHeader.GetTime(),
// 					Root:               commitmenttypes.NewMerkleRoot([]byte{}), // empty bytes
// 					NextValidatorsHash: tmHeader.Header.NextValidatorsHash,
// 				}

// 				tmConsensusStateData, err := suite.chainA.Codec.MarshalInterface(tmConsensusState)
// 				suite.Require().NoError(err)
// 				wasmConsensusState := &types.ConsensusState{
// 					Data: tmConsensusStateData,
// 				}

// 				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(
// 					suite.chainA.GetContext(),
// 					path.EndpointA.ClientID,
// 					tmHeader.GetHeight(),
// 					wasmConsensusState,
// 				)
// 			},
// 			true,
// 		},
// 		{
// 			"previous consensus state exists and header time is before previous consensus state time",
// 			func() {
// 				wasmHeader, ok := clientMessage.(*types.ClientMessage)
// 				suite.Require().True(ok)

// 				var wasmData exported.ClientMessage
// 				err := suite.chainA.Codec.UnmarshalInterface(wasmHeader.Data, &wasmData)
// 				suite.Require().NoError(err)

// 				tmHeader, ok := wasmData.(*ibctm.Header)
// 				suite.Require().True(ok)

// 				// offset header timestamp before previous consensus state timestamp
// 				tmHeader.Header.Time = tmHeader.GetTime().Add(-time.Hour)

// 				wasmHeader.Data, err = suite.chainA.Codec.MarshalInterface(tmHeader)
// 				suite.Require().NoError(err)
// 			},
// 			true,
// 		},
// 		{
// 			"next consensus state exists and header time is after next consensus state time",
// 			func() {
// 				wasmHeader, ok := clientMessage.(*types.ClientMessage)
// 				suite.Require().True(ok)

// 				var wasmData exported.ClientMessage
// 				err := suite.chainA.Codec.UnmarshalInterface(wasmHeader.Data, &wasmData)
// 				suite.Require().NoError(err)

// 				tmHeader, ok := wasmData.(*ibctm.Header)
// 				suite.Require().True(ok)

// 				// offset header timestamp before previous consensus state timestamp
// 				tmHeader.Header.Time = tmHeader.GetTime().Add(time.Hour)

// 				wasmHeader.Data, err = suite.chainA.Codec.MarshalInterface(tmHeader)
// 				suite.Require().NoError(err)
// 				// commit block and update client, adding a new consensus state
// 				suite.coordinator.CommitBlock(suite.chainB)

// 				err = path.EndpointA.UpdateClient()
// 				suite.Require().NoError(err)
// 			},
// 			true,
// 		},
// 		{
// 			"valid fork misbehaviour returns true",
// 			func() {
// 				header1, err := path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
// 				suite.Require().NoError(err)

// 				// commit block and update client
// 				suite.coordinator.CommitBlock(suite.chainB)
// 				err = path.EndpointA.UpdateClient()
// 				suite.Require().NoError(err)

// 				header2, err := path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
// 				suite.Require().NoError(err)

// 				// assign the same height, each header will have a different commit hash
// 				header1.Header.Height = header2.Header.Height
// 				header1.Commit.Height = header2.Commit.Height

// 				tmMisbehaviour := &ibctm.Misbehaviour{
// 					Header1: header1,
// 					Header2: header2,
// 				}

// 				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
// 				suite.Require().NoError(err)
// 				clientMessage = &types.ClientMessage{
// 					Data: wasmData,
// 				}
// 			},
// 			true,
// 		},
// 		{
// 			"valid time misbehaviour: not monotonically increasing time", func() {
// 				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

// 				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
// 				suite.Require().True(found)

// 				tmMisbehaviour := &ibctm.Misbehaviour{
// 					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
// 					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
// 				}

// 				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
// 				suite.Require().NoError(err)
// 				clientMessage = &types.ClientMessage{
// 					Data: wasmData,
// 				}
// 			}, true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		suite.Run(tc.name, func() {
// 			// reset suite to create fresh application state
// 			suite.SetupWasmTendermint()
// 			path = ibctesting.NewPath(suite.chainA, suite.chainB)

// 			err := path.EndpointA.CreateClient()
// 			suite.Require().NoError(err)

// 			// ensure counterparty state is committed
// 			suite.coordinator.CommitBlock(suite.chainB)
// 			clientMessage, _, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
// 			suite.Require().NoError(err)

// 			tc.malleate()

// 			clientState := path.EndpointA.GetClientState()
// 			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

// 			foundMisbehaviour := clientState.CheckForMisbehaviour(
// 				suite.chainA.GetContext(),
// 				suite.chainA.App.AppCodec(),
// 				clientStore, // pass in clientID prefixed clientStore
// 				clientMessage,
// 			)

// 			if tc.expPass {
// 				suite.Require().True(foundMisbehaviour)
// 			} else {
// 				suite.Require().False(foundMisbehaviour)
// 			}
// 		})
// 	}
// }
