package types_test

import (
	"encoding/base64"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *WasmTestSuite) TestVerifyMisbehaviour() {
	var (
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"successful misbehaviour verification",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Header{
					Data: data,
					Height: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 39,
					},
				}
				// VerifyClientMessage must be run first
				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
				clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

				// Reset client state to the previous for the test
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, "08-wasm-0", clientState)

				data, err = base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data:     data,
				}
			},
			true,
		},
		{
			"trusted consensus state does not exist",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data:     data,
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
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			clientState = suite.clientState
			tc.setup()

			err := clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *WasmTestSuite) TestCheckForMisbehaviour() {
	var (
		clientMsg exported.ClientMessage 
		clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"valid update no misbehaviour",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Header{
					Data: data,
					Height: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 39,
					},
				}
				
				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
			},
			false,
		},
		// Need misbehaviour_identical_headers data
		/*{
			"identical misbehaviour data/headers",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour_identical_headers"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					ClientId: "08-wasm-0",
					Data:     data,
				}
				
				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
			},
			false,
		},*/
		/*{ // Create consensus state data with empty bytes of app hash / merkle root, set it as consensus state, check misbehaviour using valid header
			"consensus state already exists, app hash mismatch", 
			func() {},
			true,
		},*/
		{
			"valid fork misbehaviour returns true",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Header{
					Data: data,
					Height: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 39,
					},
				}
				// VerifyClientMessage must be run first
				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
				clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

				// Reset client state to the previous for the test
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, "08-wasm-0", clientState)

				data, err = base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data:     data,
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
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			clientState = suite.clientState
			tc.setup()
			foundMisbehaviour := clientState.CheckForMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().True(foundMisbehaviour)
			} else {
				suite.Require().False(foundMisbehaviour)
			}
		})
	}
}
