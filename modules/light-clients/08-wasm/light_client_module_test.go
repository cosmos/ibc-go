package wasm_test

import (
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

const (
	tmClientID        = "07-tendermint-0"
	wasmClientID      = "08-wasm-100"
	malformedClientID = "malformed-clientid"
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
			"cannot parse malformed subject client ID",
			func() {
				subjectClientID = malformedClientID
			},
			host.ErrInvalidID,
		},
		{
			"subject client ID does not contain 08-wasm prefix",
			func() {
				subjectClientID = tmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = malformedClientID
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
		{
			"subject and substitute have equal latest height",
			func() {
				wasmClientState, ok := subjectClientState.(*wasmtypes.ClientState)
				suite.Require().True(ok)
				wasmClientState.LatestHeight = substituteClientState.GetLatestHeight().(clienttypes.Height)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subjectClientID, wasmClientState)
			},
			clienttypes.ErrInvalidHeight,
		},
		{
			"subject height is greater than substitute height",
			func() {
				wasmClientState, ok := subjectClientState.(*wasmtypes.ClientState)
				suite.Require().True(ok)
				wasmClientState.LatestHeight = substituteClientState.GetLatestHeight().Increment().(clienttypes.Height)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subjectClientID, wasmClientState)
			},
			clienttypes.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			expectedClientStateBz = nil

			subjectEndpoint = wasmtesting.NewWasmEndpoint(suite.chainA)
			err := subjectEndpoint.CreateClient()
			suite.Require().NoError(err)

			subjectClientState = subjectEndpoint.GetClientState()
			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectEndpoint.ClientID)

			substituteEndpoint = wasmtesting.NewWasmEndpoint(suite.chainA)
			err = substituteEndpoint.CreateClient()
			suite.Require().NoError(err)

			substituteClientState = substituteEndpoint.GetClientState()

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetRouter().GetRoute(subjectEndpoint.ClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.RecoverClient(suite.chainA.GetContext(), subjectEndpoint.ClientID, substituteEndpoint.ClientID)

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
