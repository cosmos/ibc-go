package types_test

/*func (suite *WasmTestSuite) TestVerifyHeader() {
	var (
		clientMsg   exported.ClientMessage
		clientState *wasmtypes.ClientState
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
					clientMsg = &wasmtypes.Header{
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
}*/

/*func (suite *WasmTestSuite) TestUpdateState() {
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
}*/

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