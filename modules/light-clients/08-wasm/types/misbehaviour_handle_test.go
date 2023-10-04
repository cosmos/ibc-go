package types_test

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
//	testCases := []struct {
//		name     string
//		malleate func()
//		expPass  bool
//	}{
//		{
//			"valid fork misbehaviour", func() {
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
//			},
//			true,
//		},
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
//	}
//
//	for _, tc := range testCases {
//		suite.Run(tc.name, func() {
//			suite.SetupWasmTendermint()
//			path = ibctesting.NewPath(suite.chainA, suite.chainB)
//
//			err := path.EndpointA.CreateClient()
//			suite.Require().NoError(err)
//
//			tc.malleate()
//
//			clientState := path.EndpointA.GetClientState()
//			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)
//
//			err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, misbehaviour)
//
//			if tc.expPass {
//				suite.Require().NoError(err)
//			} else {
//				suite.Require().Error(err)
//			}
//		})
//	}
// }

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
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, grandpaClientID, clientState)

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
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, grandpaClientID)
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
