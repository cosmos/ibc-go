package keeper_test

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestClientUpdateProposal() {
	var (
		subject, substitute                       string
		subjectClientState, substituteClientState exported.ClientState
		content                                   govtypes.Content
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid update client proposal", func() {
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, true,
		},
		{
			"subject and substitute use different revision numbers", func() {
				tmClientState, ok := substituteClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				consState, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(s.chainA.GetContext(), substitute, tmClientState.LatestHeight)
				s.Require().True(found)
				newRevisionNumber := tmClientState.GetLatestHeight().GetRevisionNumber() + 1

				tmClientState.LatestHeight = types.NewHeight(newRevisionNumber, tmClientState.GetLatestHeight().GetRevisionHeight())

				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), substitute, tmClientState.LatestHeight, consState)
				clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), substitute)
				ibctm.SetProcessedTime(clientStore, tmClientState.LatestHeight, 100)
				ibctm.SetProcessedHeight(clientStore, tmClientState.LatestHeight, types.NewHeight(0, 1))
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitute, tmClientState)

				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, true,
		},
		{
			"cannot use solomachine as substitute for tendermint client", func() {
				solomachine := ibctesting.NewSolomachine(s.T(), s.cdc, "solo machine", "", 1)
				solomachine.Sequence = subjectClientState.GetLatestHeight().GetRevisionHeight() + 1
				substituteClientState = solomachine.ClientState()
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitute, substituteClientState)
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, false,
		},
		{
			"subject client does not exist", func() {
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, ibctesting.InvalidID, substitute)
			}, false,
		},
		{
			"substitute client does not exist", func() {
				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, ibctesting.InvalidID)
			}, false,
		},
		{
			"subject and substitute have equal latest height", func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClientState.LatestHeight = substituteClientState.GetLatestHeight().(types.Height)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subject, tmClientState)

				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, false,
		},
		{
			"update fails, client is not frozen or expired", func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClientState.FrozenHeight = types.ZeroHeight()
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subject, tmClientState)

				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, false,
		},
		{
			"substitute is frozen", func() {
				tmClientState, ok := substituteClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClientState.FrozenHeight = types.NewHeight(0, 1)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitute, tmClientState)

				content = types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			subjectPath := ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupClients(subjectPath)
			subject = subjectPath.EndpointA.ClientID
			subjectClientState = s.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupClients(substitutePath)
			substitute = substitutePath.EndpointA.ClientID

			// update substitute twice
			err := substitutePath.EndpointA.UpdateClient()
			s.Require().NoError(err)
			err = substitutePath.EndpointA.UpdateClient()
			s.Require().NoError(err)
			substituteClientState = s.chainA.GetClientState(substitute)

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClientState.AllowUpdateAfterMisbehaviour = true
			tmClientState.AllowUpdateAfterExpiry = true
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subject, tmClientState)

			tmClientState, ok = substituteClientState.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClientState.AllowUpdateAfterMisbehaviour = true
			tmClientState.AllowUpdateAfterExpiry = true
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitute, tmClientState)

			tc.malleate()

			updateProp, ok := content.(*types.ClientUpdateProposal)
			s.Require().True(ok)
			err = s.chainA.App.GetIBCKeeper().ClientKeeper.ClientUpdateProposal(s.chainA.GetContext(), updateProp)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestHandleUpgradeProposal() {
	var (
		upgradedClientState *ibctm.ClientState
		oldPlan, plan       upgradetypes.Plan
		content             govtypes.Content
		err                 error
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid upgrade proposal", func() {
				content, err = types.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, plan, upgradedClientState)
				s.Require().NoError(err)
			}, true,
		},
		{
			"valid upgrade proposal with previous IBC state", func() {
				oldPlan = upgradetypes.Plan{
					Name:   "upgrade IBC clients",
					Height: 100,
				}

				content, err = types.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, plan, upgradedClientState)
				s.Require().NoError(err)
			}, true,
		},
		{
			"cannot unpack client state", func() {
				protoAny, err := types.PackConsensusState(&ibctm.ConsensusState{})
				s.Require().NoError(err)
				content = &types.UpgradeProposal{
					Title:               ibctesting.Title,
					Description:         ibctesting.Description,
					Plan:                plan,
					UpgradedClientState: protoAny,
				}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()      // reset
			oldPlan.Height = 0 // reset

			path := ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupClients(path)
			upgradedClientState = s.chainA.GetClientState(path.EndpointA.ClientID).ZeroCustomFields().(*ibctm.ClientState)

			// use height 1000 to distinguish from old plan
			plan = upgradetypes.Plan{
				Name:   "upgrade IBC clients",
				Height: 1000,
			}

			tc.malleate()

			// set the old plan if it is not empty
			if oldPlan.Height != 0 {
				// set upgrade plan in the upgrade store
				store := s.chainA.GetContext().KVStore(s.chainA.GetSimApp().GetKey(upgradetypes.StoreKey))
				bz := s.chainA.App.AppCodec().MustMarshal(&oldPlan)
				store.Set(upgradetypes.PlanKey(), bz)

				bz, err := types.MarshalClientState(s.chainA.App.AppCodec(), upgradedClientState)
				s.Require().NoError(err)

				s.chainA.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainA.GetContext(), oldPlan.Height, bz) //nolint:errcheck
			}

			upgradeProp, ok := content.(*types.UpgradeProposal)
			s.Require().True(ok)
			err = s.chainA.App.GetIBCKeeper().ClientKeeper.HandleUpgradeProposal(s.chainA.GetContext(), upgradeProp)

			if tc.expPass {
				s.Require().NoError(err)

				// check that the correct plan is returned
				storedPlan, found := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(s.chainA.GetContext())
				s.Require().True(found)
				s.Require().Equal(plan, storedPlan)

				// check that old upgraded client state is cleared
				_, found = s.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(s.chainA.GetContext(), oldPlan.Height)
				s.Require().False(found)

				// check that client state was set
				storedClientState, found := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(s.chainA.GetContext(), plan.Height)
				s.Require().True(found)
				clientState, err := types.UnmarshalClientState(s.chainA.App.AppCodec(), storedClientState)
				s.Require().NoError(err)
				s.Require().Equal(upgradedClientState, clientState)
			} else {
				s.Require().Error(err)

				// check that the new plan wasn't stored
				storedPlan, found := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(s.chainA.GetContext())
				if oldPlan.Height != 0 {
					// NOTE: this is only true if the ScheduleUpgrade function
					// returns an error before clearing the old plan
					s.Require().True(found)
					s.Require().Equal(oldPlan, storedPlan)
				} else {
					s.Require().False(found)
					s.Require().Empty(storedPlan)
				}

				// check that client state was not set
				_, found = s.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(s.chainA.GetContext(), plan.Height)
				s.Require().False(found)

			}
		})
	}
}
