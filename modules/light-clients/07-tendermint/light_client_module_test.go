package tendermint_test

import (
	"crypto/sha256"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

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
		subjectClientID, substituteClientID string
		subjectClientState                  exported.ClientState
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
		upgradedClientStateAny, upgradedConsensusStateAny     *codectypes.Any
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
				upgradeHeight := clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				zeroedUpgradedClient := upgradedClientState.(*ibctm.ClientState).ZeroCustomFields()
				zeroedUpgradedClientAny, err := codectypes.NewAnyWithValue(zeroedUpgradedClient)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(zeroedUpgradedClientAny))
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedConsensusStateAny))
				suite.Require().NoError(err)

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				upgradedClientStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
			},
			nil,
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
				upgradedClientStateAny = &codectypes.Any{
					Value: []byte("invalid client state bytes"),
				}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"upgraded consensus state is not tendermint consensus state",
			func() {
				upgradedConsensusStateAny = &codectypes.Any{
					Value: []byte("invalid consensus state bytes"),
				}
			},
			clienttypes.ErrInvalidConsensus,
		},
		{
			"upgraded client state height is not greater than current height",
			func() {
				// upgrade height is at next block
				upgradeHeight := clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				zeroedUpgradedClient := upgradedClientState.(*ibctm.ClientState).ZeroCustomFields()
				zeroedUpgradedClientAny, err := codectypes.NewAnyWithValue(zeroedUpgradedClient)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(zeroedUpgradedClientAny))
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedConsensusStateAny))
				suite.Require().NoError(err)

				// change upgraded client state height to be lower than current client state height
				tmClient := upgradedClientState.(*ibctm.ClientState)

				newLatestheight, ok := path.EndpointA.GetClientLatestHeight().Decrement()
				suite.Require().True(ok)

				tmClient.LatestHeight = newLatestheight.(clienttypes.Height)
				upgradedClientStateAny, err = codectypes.NewAnyWithValue(tmClient)
				suite.Require().NoError(err)

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				upgradedClientStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
			},
			ibcerrors.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			clientID = path.EndpointA.ClientID
			clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
			revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

			newUnbondindPeriod := ubdPeriod + trustingPeriod
			newChainID, err := clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
			suite.Require().NoError(err)

			upgradedClientState = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, newUnbondindPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)
			upgradedClientStateAny, err = codectypes.NewAnyWithValue(upgradedClientState)
			suite.Require().NoError(err)

			nextValsHash := sha256.Sum256([]byte("new-nextValsHash"))
			upgradedConsensusState := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), nextValsHash[:])

			upgradedConsensusStateAny, err = codectypes.NewAnyWithValue(upgradedConsensusState)
			suite.Require().NoError(err)

			tc.malleate()

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetRouter().GetRoute(clientID)
			suite.Require().True(found)

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				suite.chainA.GetContext(),
				clientID,
				upgradedClientStateAny.Value,
				upgradedConsensusStateAny.Value,
				upgradedClientStateProof,
				upgradedConsensusStateProof,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				expClientState := path.EndpointA.GetClientState()
				expClientStateBz := suite.chainA.Codec.MustMarshal(expClientState)
				suite.Require().Equal(upgradedClientStateAny.Value, expClientStateBz)

				expConsensusState := ibctm.NewConsensusState(upgradedConsensusState.Timestamp, commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)), upgradedConsensusState.NextValidatorsHash)
				expConsensusStateBz := suite.chainA.Codec.MustMarshal(expConsensusState)

				consensusStateBz := suite.chainA.Codec.MustMarshal(path.EndpointA.GetConsensusState(path.EndpointA.GetClientLatestHeight()))
				suite.Require().Equal(expConsensusStateBz, consensusStateBz)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
