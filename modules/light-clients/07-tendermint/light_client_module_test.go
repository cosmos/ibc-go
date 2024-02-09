package tendermint_test

import (
	"crypto/sha256"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	tmClientID   = "07-tendermint-100"
	wasmClientID = "08-wasm-0"
)

func (suite *TendermintTestSuite) TestVerifyUpgradeAndUpdateState() {
	var (
		clientID                                              string
		path                                                  *ibctesting.Path
		upgradedClientState                                   exported.ClientState
		upgradedClientStateBz, upgradedConsStateBz            []byte
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
				zeroedUpgradedClient := upgradedClientState.ZeroCustomFields()
				zeroedUpgradedClientBz := suite.chainA.App.AppCodec().MustMarshal(zeroedUpgradedClient)
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), zeroedUpgradedClientBz)      //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
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
				clientID = wasmClientID
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
			"cannot unmarshal upgraded client state",
			func() {
				upgradedClientStateBz = []byte{}
			},
			clienttypes.ErrClientNotFound, // TODO: use right error here
		},
		{
			"cannot unmarshal upgraded consensus state",
			func() {
				upgradedConsStateBz = []byte{}
			},
			clienttypes.ErrClientNotFound, // TODO: use right error here
		},
		// {
		// 	"upgraded client state height is not greater than current height",
		// 	func() {
		// 		// last Height is at next block
		// 		lastHeight = clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

		// 		// zero custom fields and store in upgrade store
		// 		err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
		// 		suite.Require().NoError(err)
		// 		err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
		// 		suite.Require().NoError(err)

		// 		// change upgradedClient height to be lower than current client state height
		// 		tmClient := upgradedClient.(*ibctm.ClientState)
		// 		tmClient.LatestHeight = clienttypes.NewHeight(0, 1)
		// 		upgradedClient = tmClient

		// 		suite.coordinator.CommitBlock(suite.chainB)
		// 		err = path.EndpointA.UpdateClient()
		// 		suite.Require().NoError(err)

		// 		cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
		// 		suite.Require().True(found)

		// 		upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
		// 		upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
		// 	},
		//  ibcerrors.ErrInvalidHeight
		// },
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

			var err error
			newChainID, err := clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
			suite.Require().NoError(err)

			upgradedClientState = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)
			upgradedClientStateBz = cdc.MustMarshal(upgradedClientState)

			nextValsHash := sha256.Sum256([]byte("nextValsHash"))
			upgradedConsState := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("merkle-root")), nextValsHash[:])
			upgradedConsStateBz = cdc.MustMarshal(upgradedConsState)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetRouter().GetRoute(clientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				ctx,
				clientID,
				upgradedClientStateBz,
				upgradedConsStateBz,
				upgradedClientStateProof,
				upgradedConsensusStateProof,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				clientState := suite.chainA.GetClientState(clientID)
				suite.Require().NotNil(clientState)
				// TODO: check client state bytes matches upgraded client state bytes

				consensusState, found := suite.chainA.GetConsensusState(clientID, clientState.GetLatestHeight())
				suite.Require().True(found)
				suite.Require().NotNil(consensusState)
				tmConsensusState, ok := consensusState.(*ibctm.ConsensusState)
				suite.Require().True(ok)
				suite.Require().Equal(upgradedConsState.NextValidatorsHash, tmConsensusState.NextValidatorsHash)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
