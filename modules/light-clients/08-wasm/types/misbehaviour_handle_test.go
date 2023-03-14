package types_test

import (
	//"encoding/base64"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	//wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
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
		/*{
			"successful misbehaviour verification",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					ClientId: "08-wasm-0",
					Data:     data,
				}
				clientState = suite.clientState
			},
			true,
		},*/
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			err := suite.clientState.Initialize(suite.ctx, suite.chainA.Codec, suite.store, &suite.consensusState)
			suite.Require().NoError(err)
			tc.setup()
			err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}