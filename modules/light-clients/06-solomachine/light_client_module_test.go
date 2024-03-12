package solomachine_test

import (
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	smClientID   = "06-solomachine-100"
	wasmClientID = "08-wasm-0"
)

func (suite *SoloMachineTestSuite) TestRecoverClient() {
	var (
		subjectClientID, substituteClientID       string
		subjectClientState, substituteClientState *solomachine.ClientState
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
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 06-solomachine prefix",
			func() {
				substituteClientID = wasmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = smClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = smClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			cdc := suite.chainA.Codec
			ctx := suite.chainA.GetContext()

			subjectClientID = suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(ctx, exported.Solomachine)
			subject := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, substituteClientID, "testing", 1)
			subjectClientState = subject.ClientState()

			substituteClientID = suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(ctx, exported.Solomachine)
			substitute := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, substituteClientID, "testing", 1)
			substitute.Sequence++ // increase sequence so that latest height of substitute is > than subject's latest height
			substituteClientState = substitute.ClientState()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, substituteClientID)
			clientStore.Get(host.ClientStateKey())
			bz := clienttypes.MustMarshalClientState(cdc, substituteClientState)
			clientStore.Set(host.ClientStateKey(), bz)

			subjectClientState.IsFrozen = true
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(ctx, subjectClientID, subjectClientState)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetRouter().GetRoute(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err := lightClientModule.RecoverClient(ctx, subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// assert that status of subject client is now Active
				clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, subjectClientID)
				bz = clientStore.Get(host.ClientStateKey())
				smClientState := clienttypes.MustUnmarshalClientState(cdc, bz).(*solomachine.ClientState)

				suite.Require().Equal(substituteClientState.ConsensusState, smClientState.ConsensusState)
				suite.Require().Equal(substituteClientState.Sequence, smClientState.Sequence)
				suite.Require().Equal(exported.Active, smClientState.Status(ctx, clientStore, cdc))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *SoloMachineTestSuite) TestVerifyUpgradeAndUpdateState() {
	clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.chainA.GetContext(), exported.Solomachine)

	lightClientModule, found := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetRouter().GetRoute(clientID)
	suite.Require().True(found)

	err := lightClientModule.VerifyUpgradeAndUpdateState(suite.chainA.GetContext(), clientID, nil, nil, nil, nil)
	suite.Require().Error(err)
}
