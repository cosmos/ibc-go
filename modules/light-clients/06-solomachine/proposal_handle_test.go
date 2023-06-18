package solomachine_test

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *SoloMachineTestSuite) TestCheckSubstituteAndUpdateState() {
	var (
		subjectClientState    *solomachine.ClientState
		substituteClientState exported.ClientState
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {

		testCases := []struct {
			name     string
			malleate func()
			expPass  bool
		}{
			{
				"substitute is not the solo machine", func() {
					substituteClientState = &ibctm.ClientState{}
				}, false,
			},
			{
				"subject public key is nil", func() {
					subjectClientState.ConsensusState.PublicKey = nil
				}, false,
			},

			{
				"substitute public key is nil", func() {
					substituteClientState.(*solomachine.ClientState).ConsensusState.PublicKey = nil
				}, false,
			},
			{
				"subject and substitute use the same public key", func() {
					substituteClientState.(*solomachine.ClientState).ConsensusState.PublicKey = subjectClientState.ConsensusState.PublicKey
				}, false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				s.SetupTest()

				subjectClientState = sm.ClientState()
				substitute := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "substitute", "testing", 5)
				substituteClientState = substitute.ClientState()

				tc.malleate()

				subjectClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), sm.ClientID)
				substituteClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), substitute.ClientID)

				err := subjectClientState.CheckSubstituteAndUpdateState(s.chainA.GetContext(), s.chainA.App.AppCodec(), subjectClientStore, substituteClientStore, substituteClientState)

				if tc.expPass {
					s.Require().NoError(err)

					// ensure updated client state is set in store
					bz := subjectClientStore.Get(host.ClientStateKey())
					updatedClient := clienttypes.MustUnmarshalClientState(s.chainA.App.AppCodec(), bz).(*solomachine.ClientState)

					s.Require().Equal(substituteClientState.(*solomachine.ClientState).ConsensusState, updatedClient.ConsensusState)
					s.Require().Equal(substituteClientState.(*solomachine.ClientState).Sequence, updatedClient.Sequence)
					s.Require().Equal(false, updatedClient.IsFrozen)

				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}
