package types_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/keeper"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
	"github.com/stretchr/testify/suite"
)

type WasmTestSuite struct {
	suite.Suite
	coordinator    *ibctesting.Coordinator
	wasm           *ibctesting.Wasm // singlesig public key
	chainA         *ibctesting.TestChain
	chainB         *ibctesting.TestChain
	ctx            sdk.Context
	cdc            codec.Codec
	now            time.Time
	store          sdk.KVStore
	clientState    types.ClientState
	consensusState types.ConsensusState
	codeId         []byte
	testData       map[string]string
	wasmKeeper     keeper.Keeper
}

func (suite *WasmTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	suite.wasm = ibctesting.NewWasm(suite.T(), suite.chainA.Codec, "wasmsingle", "testing", 1)

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)

	// TODO: deprecate usage in favor of testing package
	checkTx := false
	app := simapp.Setup(checkTx)
	suite.cdc = app.AppCodec()
	suite.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	data, err := os.ReadFile("test_data/raw.json")
	suite.Require().NoError(err)
	err = json.Unmarshal(data, &suite.testData)
	suite.Require().NoError(err)

	suite.ctx = app.BaseApp.NewContext(checkTx, tmproto.Header{Height: 1, Time: suite.now}).WithGasMeter(sdk.NewInfiniteGasMeter())
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), exported.Wasm)

	os.MkdirAll("tmp", 0o755)
	suite.wasmKeeper = app.IBCKeeper.WasmClientKeeper
	data, err = os.ReadFile("test_data/ics10_grandpa_cw.wasm")
	suite.Require().NoError(err)

	msg := types.NewMsgPushNewWasmCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), data)
	response, err := suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().NoError(err)

	data, err = hex.DecodeString(suite.testData["client_state_a0"])
	suite.Require().NoError(err)

	clientState := types.ClientState{
		Data:   data,
		CodeId: response.CodeId,
		LatestHeight: clienttypes.Height{
			RevisionNumber: 1,
			RevisionHeight: 2,
		},
	}

	suite.clientState = clientState
	data, err = hex.DecodeString(suite.testData["consensus_state_a0"])
	suite.Require().NoError(err)
	consensusState := types.ConsensusState{
		Data:      data,
		CodeId:    clientState.CodeId,
		Timestamp: uint64(suite.now.UnixNano()),
		Root: &commitmenttypes.MerkleRoot{
			Hash: []byte{0},
		},
	}
	suite.consensusState = consensusState
	suite.codeId = clientState.CodeId
}

// // Panics
// func (suite *WasmTestSuite) TestCreateClient() {
// 	var (
// 		clientMsg   exported.ClientMessage
// 		clientState *wasm.ClientState
// 	)

// 	// test singlesig and multisig public keys
// 	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
// 		testCases := []struct {
// 			name    string
// 			setup   func()
// 			expPass bool
// 		}{
// 			{
// 				"create a WASM client",
// 				func() {
// 					data, err := hex.DecodeString(suite.testData["header_a0"])
// 					suite.Require().NoError(err)
// 					clientMsg = &wasm.Header{
// 						Data: data,
// 						Height: clienttypes.Height{
// 							RevisionNumber: 1,
// 							RevisionHeight: 2,
// 						},
// 					}
// 					println(wm.ClientID)
// 				},
// 				true,
// 			},
// 		}

// 		for _, tc := range testCases {
// 			tc := tc

// 			suite.Run(tc.name, func() {
// 				tc.setup()

// 				clientState = &suite.clientState
// 				_ = clientMsg
// 				_ = clientState

// 				path := ibctesting.NewPath(suite.chainA, suite.chainB)
// 				data, err := hex.DecodeString(suite.testData["header_a0"])
// 				suite.Require().NoError(err)
// 				configHeader := wasm.Header{
// 					Data: data,
// 					Height: clienttypes.Height{
// 						RevisionNumber: 1,
// 						RevisionHeight: 2,
// 					},
// 				}
// 				path.EndpointB.ClientConfig = ibctesting.NewWasmConfig(suite.consensusState, suite.clientState, configHeader)
// 				suite.coordinator.SetupConnections(path)
// 			})
// 		}
// 	}
// }

func (suite *WasmTestSuite) TestPushNewWasmCode() {
	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	data, err := os.ReadFile("test_data/example.wasm")
	suite.Require().NoError(err)

	// test pushing a valid wasm code
	msg := types.NewMsgPushNewWasmCode(signer, data)
	response, err := suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.CodeId)

	// test wasmcode duplication
	msg = types.NewMsgPushNewWasmCode(signer, data)
	_, err = suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().Error(err)

	// test invalid wasm code
	msg = types.NewMsgPushNewWasmCode(signer, []byte{})
	_, err = suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().Error(err)
}

func (suite *WasmTestSuite) TestQueryWasmCode() {
	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	data, err := os.ReadFile("test_data/example2.wasm")
	suite.Require().NoError(err)

	// push a new wasm code
	msg := types.NewMsgPushNewWasmCode(signer, data)
	response, err := suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.CodeId)

	// test invalid query request
	_, err = suite.wasmKeeper.WasmCode(suite.ctx, &types.WasmCodeQuery{})
	suite.Require().Error(err)

	_, err = suite.wasmKeeper.WasmCode(suite.ctx, &types.WasmCodeQuery{CodeId: "test"})
	suite.Require().Error(err)

	// test valid query request
	res, err := suite.wasmKeeper.WasmCode(suite.ctx, &types.WasmCodeQuery{CodeId: hex.EncodeToString(response.CodeId)})
	suite.Require().NoError(err)
	suite.Require().NotNil(res.Code)
}

func (suite *WasmTestSuite) TestVerifyClientMessageHeader() {
	var (
		clientMsg   exported.ClientMessage
		clientState *types.ClientState
	)

	// test singlesig and multisig public keys
	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful header",
				func() {
					data, err := hex.DecodeString(suite.testData["header_a0"])
					suite.Require().NoError(err)
					clientMsg = &types.Header{
						Data: data,
						Height: clienttypes.Height{
							RevisionNumber: 1,
							RevisionHeight: 2,
						},
					}
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				tc.setup()

				clientState = &suite.clientState
				err := clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}

// func (suite *WasmTestSuite) TestUpdateStateOnMisbehaviour() {
// 	var (
// 		clientMsg   exported.ClientMessage
// 		clientState *wasm.ClientState
// 	)

// 	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
// 		testCases := []struct {
// 			name    string
// 			setup   func()
// 			expPass bool
// 		}{
// 			{
// 				"successful update",
// 				func() {
// 					data, err := hex.DecodeString(suite.testData["header_a0"])
// 					suite.Require().NoError(err)
// 					clientMsg = &wasm.Header{
// 						Data: data,
// 						Height: clienttypes.Height{
// 							RevisionNumber: 1,
// 							RevisionHeight: 2,
// 						},
// 					}
// 					clientState = &suite.clientState
// 					println(wm.ClientID)
// 				},
// 				true,
// 			},
// 		}

// 		for _, tc := range testCases {
// 			tc := tc
// 			suite.Run(tc.name, func() {
// 				tc.setup()

// 				if tc.expPass {
// 					fmt.Println(clientMsg)
// 					suite.Require().NotPanics(func() {
// 						clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)
// 					})
// 				} else {
// 					suite.Require().Panics(func() {
// 						clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)
// 					})
// 				}
// 			})
// 		}
// 	}
// }

func (suite *WasmTestSuite) TestUpdateState() {
	var (
		clientMsg   exported.ClientMessage
		clientState *types.ClientState
	)

	// test singlesig and multisig public keys
	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful update",
				func() {
					data, err := hex.DecodeString(suite.testData["header_a0"])
					suite.Require().NoError(err)
					clientMsg = &types.Header{
						Data: data,
						Height: clienttypes.Height{
							RevisionNumber: 1,
							RevisionHeight: 2,
						},
					}
					clientState = &suite.clientState
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			suite.Run(tc.name, func() {
				tc.setup()

				if tc.expPass {
					consensusHeights := clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

					clientStateBz := suite.store.Get(host.ClientStateKey())
					suite.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

					suite.Require().Len(consensusHeights, 1)
					suite.Require().Equal(clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 89,
					}, consensusHeights[0])
					suite.Require().Equal(consensusHeights[0], newClientState.(*types.ClientState).LatestHeight)
				} else {
					suite.Require().Panics(func() {
						clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)
					})
				}
			})
		}
	}
}

// TODO: uncomment when test data is aquired
/*
func (suite *WasmTestSuite) TestVerifyNonMemership() {
	var (
		clientState *wasm.ClientState

		err    error
		height clienttypes.Height
		path   []byte
		proof  []byte
	)

	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful non-membership verification",
				func() {
					// testingPath = ibctesting.NewPath(suite.chainA, suite.chainB)

					clientState = &suite.clientState
					height = clienttypes.NewHeight(wm.GetHeight().GetRevisionNumber(), wm.GetHeight().GetRevisionHeight())

					merklePath := commitmenttypes.NewMerklePath("clients", "10-grandpa-cw", "clientType")

					path, err = suite.chainA.Codec.Marshal(&merklePath)
					suite.Require().NoError(err)

					proof = []byte("proof")

				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			suite.Run(tc.name, func() {
				tc.setup()

				err = clientState.VerifyNonMembership(
					suite.chainA.GetContext(), suite.store, suite.chainA.Codec,
					height, 0, 0,
					proof, path,
				)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}
*/

// TODO: uncomment when fisherman is merged
/*
func (suite *WasmTestSuite) TestVerifyMisbehaviour() {
	var (
		clientMsg   exported.ClientMessage
		clientState *wasm.ClientState
	)

	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful misbehaviour verification",
				func() {
					data, err := hex.DecodeString(suite.testData["misbehaviour_a0"])
					suite.Require().NoError(err)
					clientMsg = &wasm.Misbehaviour{
						ClientId: wm.ClientID,
						Data:     data,
					}
					clientState = &suite.clientState
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			suite.Run(tc.name, func() {
				tc.setup()
				println(clientMsg, clientState)
				err := clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}
*/

// TODO: uncomment when test data is aquired
/*
func (suite *WasmTestSuite) TestVerifyMemership() {
	var (
		clientState *wasm.ClientState

		err    error
		height clienttypes.Height
		path   []byte
		proof  []byte
		// testingPath *ibctesting.Path
	)

	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful membership verification",
				func() {
					// testingPath = ibctesting.NewPath(suite.chainA, suite.chainB)

					clientState = &suite.clientState
					height = clienttypes.NewHeight(wm.GetHeight().GetRevisionNumber(), wm.GetHeight().GetRevisionHeight())

					merklePath := commitmenttypes.NewMerklePath("clients", "10-grandpa-cw", "clientType")

					path, err = suite.chainA.Codec.Marshal(&merklePath)
					suite.Require().NoError(err)

					proof = []byte("proof")

				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			suite.Run(tc.name, func() {
				tc.setup()

				err = clientState.VerifyMembership(
					suite.chainA.GetContext(), suite.store, suite.chainA.Codec,
					height, 0, 0,
					proof, path, []byte("data"),
				)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}
*/

func (suite *WasmTestSuite) TestWasm() {
	suite.Run("Init contract", func() {
		suite.SetupTest()
	})
}

func TestWasmTestSuite(t *testing.T) {
	suite.Run(t, new(WasmTestSuite))
}
