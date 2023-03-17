package types_test

import (
	"encoding/base64"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

func (suite *WasmTestSuite) TestVerifyHeader() {
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
			"successful verifyt header", func() {},
			true,
		},
		{
			"unsuccessful verify header: para id mismatch", func() {
				cs, err := base64.StdEncoding.DecodeString(suite.testData["client_state_para_id_mismatch"])
				suite.Require().NoError(err)

				clientState = &wasmtypes.ClientState{
					Data: cs,
					CodeId: suite.codeId,
					LatestHeight: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 36,
					},
				}
			},
			false,
		},
		{
			"unsuccessful verify header: head height < consensus height", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header_old"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Header{
					Data: data,
					Height: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 29,
					},
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			clientState = suite.clientState
			data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
			suite.Require().NoError(err)
			clientMsg = &wasmtypes.Header{
				Data: data,
				Height: clienttypes.Height{
					RevisionNumber: 2000,
					RevisionHeight: 39,
				},
			}

			tc.setup()
			err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *WasmTestSuite) TestUpdateState() {
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
			"success with height later than latest height",
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
			},
			true,
		},
		{
			"failure with not verifying client message",
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
			"invalid ClientMessage type", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					ClientId: "08-wasm-0",
					Data: data,
				}
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

			if tc.expPass {
				consensusHeights := clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

				clientStateBz := suite.store.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

				suite.Require().Len(consensusHeights, 1)
				suite.Require().Equal(clienttypes.Height{
					RevisionNumber: 2000,
					RevisionHeight: 39,
				}, consensusHeights[0])
				suite.Require().Equal(consensusHeights[0], newClientState.(*wasmtypes.ClientState).LatestHeight)
			} else {
				suite.Require().Panics(func() {
					clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
			}
		})
	}
}

func (suite *WasmTestSuite) TestUpdateStateOnMisbehaviour() {
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
			"successful update",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data: data,
					ClientId: "08-wasm-0",
				}
				clientState = suite.clientState
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			tc.setup()

			if tc.expPass {
				suite.Require().NotPanics(func() {
					clientState.UpdateStateOnMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
				clientStateBz := suite.store.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				status := newClientState.Status(suite.ctx, suite.store, suite.chainA.Codec)
				suite.Require().Equal(exported.Frozen, status)
			} else {
				suite.Require().Panics(func() {
					clientState.UpdateStateOnMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
			}
		})
	}
}