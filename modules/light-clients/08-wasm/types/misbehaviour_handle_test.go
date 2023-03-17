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

				data, err = base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					ClientId: "08-wasm-0",
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
					ClientId: "08-wasm-0",
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
		clientMsg   exported.ClientMessage
		//clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		/*{
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
			},
			false,
		},
		{
			"consensus state already exists, already updated",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					ClientId: "08-wasm-0",
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
		},*/
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			tc.setup()

			err := suite.clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
			suite.Require().NoError(err)

			foundMisbehaviour := suite.clientState.CheckForMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().True(foundMisbehaviour)
			} else {
				suite.Require().False(foundMisbehaviour)
			}
		})
	}
}