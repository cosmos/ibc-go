package solomachine_test

import (
	fmt "fmt"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	smClientID   = "06-solomachine-100"
	wasmClientID = "08-wasm-0"
)

func (suite *SoloMachineTestSuite) TestInitialize() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
		malleatedConsensus := sm.ClientState().ConsensusState
		malleatedConsensus.Timestamp += 10

		testCases := []struct {
			name        string
			consState   exported.ConsensusState
			clientState exported.ClientState
			expErr      error
		}{
			{
				"valid consensus state",
				sm.ConsensusState(),
				sm.ClientState(),
				nil,
			},
			{
				"nil consensus state",
				nil,
				sm.ClientState(),
				clienttypes.ErrInvalidConsensus,
			},
			{
				"invalid consensus state: Tendermint consensus state",
				&ibctm.ConsensusState{},
				sm.ClientState(),
				fmt.Errorf("proto: wrong wireType = 0 for field TypeUrl"),
			},
			{
				"invalid consensus state: consensus state does not match consensus state in client",
				malleatedConsensus,
				sm.ClientState(),
				clienttypes.ErrInvalidConsensus,
			},
			{
				"invalid client state: sequence is zero",
				sm.ConsensusState(),
				solomachine.NewClientState(0, sm.ConsensusState()),
				clienttypes.ErrInvalidClient,
			},
			{
				"invalid client state: Tendermint client state",
				sm.ConsensusState(),
				&ibctm.ClientState{},
				fmt.Errorf("proto: wrong wireType = 2 for field IsFrozen"),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()

				clientStateBz := suite.chainA.Codec.MustMarshal(tc.clientState)
				consStateBz := suite.chainA.Codec.MustMarshal(tc.consState)

				clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.chainA.GetContext(), exported.Solomachine)

				lcm, found := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.Route(clientID)
				suite.Require().True(found)

				err := lcm.Initialize(suite.chainA.GetContext(), clientID, clientStateBz, consStateBz)
				store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)

				expPass := tc.expErr == nil
				if expPass {
					suite.Require().NoError(err, "valid testcase: %s failed", tc.name)
					suite.Require().True(store.Has(host.ClientStateKey()))
				} else {
					suite.Require().ErrorContains(err, tc.expErr.Error())
					suite.Require().False(store.Has(host.ClientStateKey()))
				}
			})
		}
	}
}

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

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(subjectClientID)
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
				suite.Require().Equal(exported.Active, lightClientModule.Status(ctx, subjectClientID))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *SoloMachineTestSuite) TestVerifyUpgradeAndUpdateState() {
	clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.chainA.GetContext(), exported.Solomachine)

	lightClientModule, found := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.Route(clientID)
	suite.Require().True(found)

	err := lightClientModule.VerifyUpgradeAndUpdateState(suite.chainA.GetContext(), clientID, nil, nil, nil, nil)
	suite.Require().Error(err)
}
