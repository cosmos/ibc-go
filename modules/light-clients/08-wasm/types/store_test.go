package types_test

import (
	errorsmod "cosmossdk.io/errors"
	prefixstore "cosmossdk.io/store/prefix"

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
		{
			"failure: prefix store does not contain prefix",
			func() {
				clientStore = prefixstore.NewStore(nil, nil)
			},
			errorsmod.Wrapf(types.ErrRetrieveClientID, "prefix field not found"),
		},
		{
			"failure: prefix does not contain slash separated path",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("not-a-slash-separated-path"))
			},
			errorsmod.Wrapf(types.ErrRetrieveClientID, "prefix does not contain a slash"),
		},
		{
			"failure: prefix only contains one slash",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("only-one-slash/"))
			},
			errorsmod.Wrapf(types.ErrRetrieveClientID, "prefix does not contain a slash"),
		},
		{
			"failure: prefix does not contain a wasm clientID",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("/not-client-id/"))
			},
			errorsmod.Wrapf(types.ErrRetrieveClientID, "prefix does not contain a clientID"),
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
