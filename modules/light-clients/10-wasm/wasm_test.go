package wasm_test

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	_go "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v5/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
	"github.com/cosmos/ibc-go/v5/modules/core/exported"
	wasm "github.com/cosmos/ibc-go/v5/modules/light-clients/10-wasm"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
	"github.com/cosmos/ibc-go/v5/testing/simapp"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

type WasmTestSuite struct {
	suite.Suite
	coordinator *ibctesting.Coordinator
	wasm        *ibctesting.Wasm // singlesig public key
	// Tendermint chain
	chainA *ibctesting.TestChain
	// Grandpa chain
	chainB         *ibctesting.TestChain
	ctx            sdk.Context
	cdc            codec.Codec
	now            time.Time
	store          sdk.KVStore
	clientState    wasm.ClientState
	consensusState wasm.ConsensusState
	codeId         []byte
	testData       map[string]string
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
	wasmConfig := wasm.VMConfig{
		DataDir:           "tmp",
		SupportedFeatures: []string{"storage", "iterator"},
		MemoryLimitMb:     uint32(math.Pow(2, 12)),
		PrintDebug:        true,
		CacheSizeMb:       uint32(math.Pow(2, 8)),
	}
	validationConfig := wasm.ValidationConfig{
		MaxSizeAllowed: int(math.Pow(2, 26)),
	}
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), exported.Wasm)
	data, err = hex.DecodeString(suite.testData["client_state_a0"])
	suite.Require().NoError(err)
	
	os.MkdirAll("tmp", 0o755)
	wasm.CreateVM(&wasmConfig, &validationConfig)
	data, err = os.ReadFile("ics10_grandpa_cw.wasm")
	suite.Require().NoError(err)
	
	// Currently pushing to client-specific store, this will change to a keeper, but okay for now/testing (single wasm client)
	codeId, err := wasm.PushNewWasmCode(suite.store, data)
	suite.Require().NoError(err)

	clientState := wasm.ClientState{
		Data: data,
		CodeId: codeId,
		LatestHeight: clienttypes.Height{
			RevisionNumber: 1,
			RevisionHeight: 2,
		},
		ProofSpecs: []*_go.ProofSpec{
			{
				LeafSpec: &_go.LeafOp{
					Hash:         _go.HashOp_SHA256,
					Length:       _go.LengthOp_FIXED32_BIG,
					PrehashValue: _go.HashOp_SHA256,
					Prefix:       []byte{0},
				},
				InnerSpec: &_go.InnerSpec{
					ChildOrder:      []int32{0, 1},
					ChildSize:       33,
					MinPrefixLength: 4,
					MaxPrefixLength: 12,
					EmptyChild:      nil,
					Hash:            _go.HashOp_SHA256,
				},
				MaxDepth: 0,
				MinDepth: 0,
			},
		},
		Repository: "test",
	}

	suite.clientState = clientState
	data, err = hex.DecodeString(suite.testData["consensus_state_a0"])
	suite.Require().NoError(err)
	consensusState := wasm.ConsensusState{
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

func (suite *WasmTestSuite) TestVerifyClientMessageHeader() {
	var (
		clientMsg   exported.ClientMessage
		clientState *wasm.ClientState
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
					clientMsg = &wasm.Header{
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

func (suite *WasmTestSuite) TestUpdateState() {
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
				"successful update",
				func() {
					data, err := hex.DecodeString(suite.testData["header_a0"])
					suite.Require().NoError(err)
					clientMsg = &wasm.Header{
						Data: data,
						Height: &clienttypes.Height{
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
					suite.Require().Equal(&clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 89,
					}, consensusHeights[0])
					suite.Require().Equal(consensusHeights[0], newClientState.(*wasm.ClientState).LatestHeight)
				} else {
					suite.Require().Panics(func() {
						clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)
					})
				}
			})
		}
	}
}

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
