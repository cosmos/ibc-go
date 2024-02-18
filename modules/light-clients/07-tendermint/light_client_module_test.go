package tendermint_test

import (
	"crypto/sha256"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

var (
	tmClientID          = clienttypes.FormatClientIdentifier(exported.Tendermint, 100)
	solomachineClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 0)
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
				subjectClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"subject client ID does not contain 07-tendermint prefix",
			func() {
				subjectClientID = solomachineClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 07-tendermint prefix",
			func() {
				substituteClientID = solomachineClientID
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

func (suite *TendermintTestSuite) TestVerifyUpgradeAndUpdateState() {
	var (
		clientID                                              string
		path                                                  *ibctesting.Path
		upgradedClientState                                   exported.ClientState
		upgradedClientStateBz, upgradedConsensusStateBz       []byte
		upgradedClientStateProof, upgradedConsensusStateProof []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				// upgrade height is at next block
				lastHeight := clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				zeroedUpgradedClient := upgradedClientState.(*ibctm.ClientState).ZeroCustomFields()
				zeroedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), zeroedUpgradedClient)
				err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), zeroedUpgradedClientBz)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsensusStateBz)
				suite.Require().NoError(err)

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), clientID)
				suite.Require().True(found)

				upgradedClientStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			nil,
		},
		{
			"cannot parse malformed client ID",
			func() {
				clientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"client type is not 07-tendermint",
			func() {
				clientID = solomachineClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find client state",
			func() {
				clientID = tmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"upgraded client state is not for tendermint client state",
			func() {
				upgradedClientStateBz = []byte{}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"upgraded consensus state is not tendermint consensus state",
			func() {
				upgradedConsensusStateBz = []byte{}
			},
			clienttypes.ErrInvalidConsensus,
		},
		{
			"upgraded client state height is not greater than current height",
			func() {
				// last Height is at next block
				lastHeight := clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				zeroedUpgradedClient := upgradedClientState.(*ibctm.ClientState).ZeroCustomFields()
				zeroedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), zeroedUpgradedClient)
				err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), zeroedUpgradedClientBz)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsensusStateBz)
				suite.Require().NoError(err)

				// change upgraded client state height to be lower than current client state height
				clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
				tmClient := upgradedClientState.(*ibctm.ClientState)
				newLatestheight, ok := clientState.GetLatestHeight().Decrement()
				suite.Require().True(ok)
				tmClient.LatestHeight = newLatestheight.(clienttypes.Height)
				upgradedClientStateBz = clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), tmClient)

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				upgradedClientStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			ibcerrors.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			cdc := suite.chainA.App.AppCodec()
			ctx := suite.chainA.GetContext()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()
			clientID = path.EndpointA.ClientID
			clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
			revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

			newUnbondindPeriod := ubdPeriod + trustingPeriod
			upgradedClientState = ibctm.NewClientState(clientState.ChainId, ibctm.DefaultTrustLevel, trustingPeriod, newUnbondindPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)
			upgradedClientStateBz = clienttypes.MustMarshalClientState(cdc, upgradedClientState)

			nextValsHash := sha256.Sum256([]byte("new-nextValsHash"))
			upgradedConsensusState := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), nextValsHash[:])
			upgradedConsensusStateBz = clienttypes.MustMarshalConsensusState(cdc, upgradedConsensusState)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetRouter().GetRoute(clientID)
			suite.Require().True(found)

			tc.malleate()

			err := lightClientModule.VerifyUpgradeAndUpdateState(
				ctx,
				clientID,
				upgradedClientStateBz,
				upgradedConsensusStateBz,
				upgradedClientStateProof,
				upgradedConsensusStateProof,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				clientState := suite.chainA.GetClientState(clientID)
				suite.Require().NotNil(clientState)
				clientStateBz := clienttypes.MustMarshalClientState(cdc, upgradedClientState)
				suite.Require().Equal(upgradedClientStateBz, clientStateBz)

				consensusState, found := suite.chainA.GetConsensusState(clientID, clientState.GetLatestHeight())
				suite.Require().True(found)
				suite.Require().NotNil(consensusState)
				tmConsensusState, ok := consensusState.(*ibctm.ConsensusState)
				suite.Require().True(ok)
				suite.Require().Equal(upgradedConsensusState.NextValidatorsHash, tmConsensusState.NextValidatorsHash)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
