package types_test

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func (suite *TypesTestSuite) TestGetClientID() {
	clientStore := suite.store

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: clientID retrieved",
			func() {},
			nil,
		},
		{
			"failure: clientStore is nil",
			func() {
				clientStore = nil
			},
			errorsmod.Wrapf(types.ErrRetrieveClientID, "clientStore is not a prefix store"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()
			clientID, err := types.GetClientID(clientStore)

			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().Equal(defaultWasmClientID, clientID)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
