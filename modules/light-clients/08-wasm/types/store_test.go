package types_test

import (
	prefixstore "cosmossdk.io/store/prefix"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func (suite *TypesTestSuite) TestGetClientID() {
	clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), defaultWasmClientID)

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
			"success: clientID retrieved from updateProposalWrappedStore",
			func() {
				clientStore = types.NewUpdateProposalWrappedStore(clientStore, nil, nil, nil)
			},
			nil,
		},
		{
			"failure: clientStore is nil",
			func() {
				clientStore = nil
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: prefix store does not contain prefix",
			func() {
				clientStore = prefixstore.NewStore(nil, nil)
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: prefix does not contain slash separated path",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("not-a-slash-separated-path"))
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: prefix only contains one slash",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("only-one-slash/"))
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: prefix does not contain a wasm clientID",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("/not-client-id/"))
			},
			types.ErrRetrieveClientID,
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
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
