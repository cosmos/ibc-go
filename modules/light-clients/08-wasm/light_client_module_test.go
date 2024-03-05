package wasm_test

import (
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	tmClientID   = "07-tendermint-0"
	wasmClientID = "08-wasm-100"
)

func (suite *WasmTestSuite) TestRecoverClient() {
	var (
		expectedClientStateBz                     []byte
		subjectClientID, substituteClientID       string
		subjectClientState, substituteClientState exported.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		// TODO(02-client routing): add successful test when light client module does not call into 08-wasm ClientState
		// {
		// 	"success",
		// 	func() {
		// 	},
		// 	nil,
		// },
		{
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 08-wasm prefix",
			func() {
				substituteClientID = tmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = wasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = wasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			expectedClientStateBz = nil

			subjectEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := subjectEndpoint.CreateClient()
			suite.Require().NoError(err)
			subjectClientID = subjectEndpoint.ClientID

			subjectClientState = subjectEndpoint.GetClientState()
			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectClientID)

			substituteEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err = substituteEndpoint.CreateClient()
			suite.Require().NoError(err)
			substituteClientID = substituteEndpoint.ClientID

			substituteClientState = substituteEndpoint.GetClientState()

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetRouter().GetRoute(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.RecoverClient(suite.chainA.GetContext(), subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := subjectClientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expectedClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
