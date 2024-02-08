package tendermint_test

import (
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	tmClientID        = "07-tendermint-100"
	wasmClientID      = "08-wasm-0"
	malformedClientID = "malformed-clientid"
)

func (suite *TendermintTestSuite) TestRecoverClient() {
	var (
		subjectClientID, substituteClientID       string
		subjectClientState, substituteClientState exported.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
			},
			nil,
		},
		{
			"cannot parse malformed subject client ID",
			func() {
				subjectClientID = malformedClientID
			},
			host.ErrInvalidID,
		},
		{
			"subject client ID does not contain 07-tendermint prefix",
			func() {
				subjectClientID = wasmClientID
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
			"substitute client ID does not contain 07-tendermint prefix",
			func() {
				substituteClientID = wasmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = tmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = tmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"subject and substitute have equal latest height",
			func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				suite.Require().True(ok)
				tmClientState.LatestHeight = substituteClientState.GetLatestHeight().(clienttypes.Height)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subjectClientID, tmClientState)
			},
			clienttypes.ErrInvalidHeight,
		},
		{
			"subject height is greater than substitute height",
			func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				suite.Require().True(ok)
				tmClientState.LatestHeight = substituteClientState.GetLatestHeight().Increment().(clienttypes.Height)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subjectClientID, tmClientState)
			},
			clienttypes.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			ctx := suite.chainA.GetContext()

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			subjectPath.SetupClients()
			subjectClientID = subjectPath.EndpointA.ClientID
			subjectClientState = suite.chainA.GetClientState(subjectClientID)

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath.SetupClients()
			substituteClientID = substitutePath.EndpointA.ClientID
			substituteClientState = suite.chainA.GetClientState(substituteClientID)

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(ctx, subjectPath.EndpointA.ClientID, tmClientState)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetRouter().GetRoute(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err := lightClientModule.RecoverClient(ctx, subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// assert that status of subject client is now Active
				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, subjectClientID)
				tmClientState := subjectPath.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().Equal(exported.Active, tmClientState.Status(ctx, clientStore, suite.chainA.App.AppCodec()))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
