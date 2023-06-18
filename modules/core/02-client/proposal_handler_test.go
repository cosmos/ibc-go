package client_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	client "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *ClientTestSuite) TestNewClientUpdateProposalHandler() {
	var (
		content govtypes.Content
		err     error
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid update client proposal", func() {
				subjectPath := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(subjectPath)
				subjectClientState := s.chainA.GetClientState(subjectPath.EndpointA.ClientID)

				substitutePath := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(substitutePath)

				// update substitute twice
				err = substitutePath.EndpointA.UpdateClient()
				s.Require().NoError(err)
				err = substitutePath.EndpointA.UpdateClient()
				s.Require().NoError(err)
				substituteClientState := s.chainA.GetClientState(substitutePath.EndpointA.ClientID)

				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClientState.AllowUpdateAfterMisbehaviour = true
				tmClientState.FrozenHeight = tmClientState.LatestHeight
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subjectPath.EndpointA.ClientID, tmClientState)

				// replicate changes to substitute (they must match)
				tmClientState, ok = substituteClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClientState.AllowUpdateAfterMisbehaviour = true
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitutePath.EndpointA.ClientID, tmClientState)

				content = clienttypes.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subjectPath.EndpointA.ClientID, substitutePath.EndpointA.ClientID)
			}, true,
		},
		{
			"nil proposal", func() {
				content = nil
			}, false,
		},
		{
			"unsupported proposal type", func() {
				content = &distributiontypes.CommunityPoolSpendProposal{ //nolint:staticcheck
					Title:       ibctesting.Title,
					Description: ibctesting.Description,
					Recipient:   s.chainA.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin("communityfunds", sdkmath.NewInt(10))),
				}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			tc.malleate()

			proposalHandler := client.NewClientProposalHandler(s.chainA.App.GetIBCKeeper().ClientKeeper)

			err = proposalHandler(s.chainA.GetContext(), content)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
