package types_test

import (
	"encoding/base64"
	"encoding/json"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	storetypes "cosmossdk.io/store/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

/* func (suite *TypesTestSuite) TestVerifyMisbehaviourGrandpa() {
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
			"successful misbehaviour verification",
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
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, grandpaClientID, clientState)

				data, err = base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
			},
			true,
		},
		{
			"trusted consensus state does not exist",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
			},
			false,
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
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, grandpaClientID)
			suite.Require().True(ok)

			tc.malleate()

			err := clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}*/

// func (suite *TypesTestSuite) TestVerifyMisbehaviourTendermint() {
//	// Setup different validators and signers for testing different types of updates
//	altPrivVal := ibctestingmock.NewPV()
//	altPubKey, err := altPrivVal.GetPubKey()
//	suite.Require().NoError(err)
//
//	// create modified heights to use for test-cases
//	altVal := tmtypes.NewValidator(altPubKey, 100)
//
//	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
//	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})
//	altSigners := getAltSigners(altVal, altPrivVal)
//
//	var (
//		path         *ibctesting.Path
//		misbehaviour exported.ClientMessage
//	)
//
// testCases := []struct {
// 	name     string
// 	malleate func()
// 	expPass  bool
// }{
// 	{
// 		"valid fork misbehaviour", func() {
// 			trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

// 			trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
// 			suite.Require().True(found)

// 			err := path.EndpointA.UpdateClient()
// 			suite.Require().NoError(err)

// 			height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

// 			tmMisbehaviour := &ibctm.Misbehaviour{
// 				Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Second), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
// 				Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
// 			}
// 			wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
// 			suite.Require().NoError(err)
// 			misbehaviour = &types.ClientMessage{
// 				Data: wasmData,
// 			}
// 		},
// 		true,
// 	},
//		{
//			"valid time misbehaviour", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			},
//			true,
//		},
//		{
//			"valid time misbehaviour, header 1 time stricly less than header 2 time", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Second), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			},
//			true,
//		},
//		{
//			"valid misbehavior at height greater than last consensusState", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Second), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, true,
//		},
//		{
//			"valid misbehaviour with different trusted heights", func() {
//				trustedHeight1 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals1, found := suite.chainB.GetValsAtHeight(int64(trustedHeight1.RevisionHeight))
//				suite.Require().True(found)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				trustedHeight2 := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals2, found := suite.chainB.GetValsAtHeight(int64(trustedHeight2.RevisionHeight))
//				suite.Require().True(found)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight1, suite.chainB.CurrentHeader.Time.Add(time.Second), suite.chainB.Vals, suite.chainB.NextVals, trustedVals1, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight2, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals2, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			},
//			true,
//		},
//		{
//			"valid misbehaviour at a previous revision", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Second), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//
//				// increment revision number
//				err = path.EndpointB.UpgradeChain()
//				suite.Require().NoError(err)
//			},
//			true,
//		},
//		{
//			"valid misbehaviour at a future revision", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				futureRevision := fmt.Sprintf("%s-%d", strings.TrimSuffix(suite.chainB.ChainID, fmt.Sprintf("-%d", clienttypes.ParseChainID(suite.chainB.ChainID))), height.GetRevisionNumber()+1)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Second), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(futureRevision, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			},
//			true,
//		},
//		{
//			"valid misbehaviour with trusted heights at a previous revision", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				// increment revision of chainID
//				err := path.EndpointB.UpgradeChain()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			},
//			true,
//		},
//		{
//			"consensus state's valset hash different from misbehaviour should still pass", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				// Create bothValSet with both suite validator and altVal
//				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altValSet.Proposer))
//				bothSigners := suite.chainB.Signers
//				bothSigners[altValSet.Proposer.Address.String()] = altPrivVal
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, true,
//		},
//		{
//			"invalid misbehaviour: misbehaviour from different chain", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader("evmos", int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, false,
//		},
//		{
//			"misbehaviour trusted validators does not match validator hash in trusted consensus state", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, altValSet, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, false,
//		},
//		{
//			"trusted consensus state does not exist", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight.Increment().(clienttypes.Height), suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, false,
//		},
//		{
//			"invalid tendermint misbehaviour", func() {
//				misbehaviour = &solomachine.Misbehaviour{}
//			}, false,
//		},
//		{
//			"trusting period expired", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				suite.chainA.ExpireClient(path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig).TrustingPeriod)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, false,
//		},
//		{
//			"header 1 valset has too much change", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight+1), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, false,
//		},
//		{
//			"header 2 valset has too much change", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight+1), trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, false,
//		},
//		{
//			"both header 1 and header 2 valsets have too much change", func() {
//				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
//				suite.Require().True(found)
//
//				err := path.EndpointA.UpdateClient()
//				suite.Require().NoError(err)
//
//				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
//
//				tmMisbehaviour := &ibctm.Misbehaviour{
//					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight+1), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), altValSet, suite.chainB.NextVals, trustedVals, altSigners),
//					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight+1), trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, suite.chainB.NextVals, trustedVals, altSigners),
//				}
//				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
//				suite.Require().NoError(err)
//				misbehaviour = &types.ClientMessage{
//					Data: wasmData,
//				}
//			}, false,
//		},
// 	}

// 	for _, tc := range testCases {
// 		suite.Run(tc.name, func() {
// 			// reset suite to create fresh application state
// 			suite.SetupWasmWithMockVM()

// 			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
// 			err := endpoint.CreateClient()
// 			suite.Require().NoError(err)

// 			tc.malleate()

// 			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
// 			clientState := endpoint.GetClientState()
// 			foundMisbehaviour := clientState.CheckForMisbehaviour(
// 				suite.chainA.GetContext(),
// 				suite.chainA.App.AppCodec(),
// 				clientStore, // pass in clientID prefixed clientStore
// 				clientMessage,
// 			)

// 			suite.Require().Equal(tc.foundMisbehaviour, foundMisbehaviour)
// 		})
// 	}
// }

func (suite *TypesTestSuite) TestCheckForMisbehaviourTendermint() {
	var (
		clientState       exported.ClientState
		clientStore       storetypes.KVStore
		clientMessage     exported.ClientMessage
		foundMisbehaviour bool
	)

	testCases := []struct {
		name              string
		checkMisbehaviour func()
		expPass           bool
		foundMisbehaviour bool
	}{
		{
			"no misbehaviour",
			func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: false})
					suite.Assert().NoError(err)
					return resp, types.DefaultGasUsed, nil
				})

				foundMisbehaviour = clientState.CheckForMisbehaviour(
					suite.chainA.GetContext(),
					suite.chainA.App.AppCodec(),
					clientStore,
					clientMessage,
				)
			},
			true,
			false,
		},
		{
			"misbehaviour found", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMessage = &types.ClientMessage{
					Data: data,
				}

				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: false})
					suite.Assert().NoError(err)
					return []byte(resp), types.DefaultGasUsed, nil
				})

				foundMisbehaviour = clientState.CheckForMisbehaviour(
					suite.chainA.GetContext(),
					suite.chainA.App.AppCodec(),
					clientStore,
					clientMessage,
				)
			},
			true,
			true,
		},
		{
			"query failed, vm panic", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMessage = &types.ClientMessage{
					Data: data,
				}

				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp := "cannot be unmarshalled"
					return []byte(resp), types.DefaultGasUsed, nil
				})

				foundMisbehaviour = clientState.CheckForMisbehaviour(
					suite.chainA.GetContext(),
					suite.chainA.App.AppCodec(),
					clientStore,
					clientMessage,
				)
			},
			false,
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

			clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientState = endpoint.GetClientState()

			if tc.expPass {
				suite.Require().NotPanics(tc.checkMisbehaviour, "unexpected panic")
				suite.Require().Equal(tc.foundMisbehaviour, foundMisbehaviour)
			} else {
				suite.Require().Panicsf(tc.checkMisbehaviour, "failed to unmarshal result of wasm query")
			}
		})
	}
}
