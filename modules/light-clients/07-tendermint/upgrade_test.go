package tendermint_test

import (
	"errors"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *TendermintTestSuite) TestVerifyUpgrade() {
	var (
		newChainID                                       string
		upgradedClient                                   exported.ClientState
		upgradedConsState                                exported.ConsensusState
		lastHeight                                       clienttypes.Height
		path                                             *ibctesting.Path
		upgradedClientProof, upgradedConsensusStateProof []byte
		upgradedClientBz, upgradedConsStateBz            []byte
		err                                              error
	)

	testCases := []struct {
		name   string
		setup  func()
		expErr error
	}{
		{
			name: "successful upgrade",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: nil,
		},
		{
			name: "successful upgrade to same revision",
			setup: func() {
				upgradedClient = ibctm.NewClientState(suite.chainB.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(clienttypes.ParseChainID(suite.chainB.ChainID), upgradedClient.(*ibctm.ClientState).LatestHeight.GetRevisionHeight()+10), commitmenttypes.GetSDKSpecs(), upgradePath)
				upgradedClient = upgradedClient.(*ibctm.ClientState).ZeroCustomFields()
				upgradedClientBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: nil,
		},
		{
			name: "unsuccessful upgrade: upgrade path not set",
			setup: func() {
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				// set upgrade path to empty
				tmCs.UpgradePath = []string{}
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, tmCs)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: clienttypes.ErrInvalidUpgradeClient,
		},
		{
			name: "unsuccessful upgrade: upgrade consensus state must be tendermint consensus state",
			setup: func() {
				upgradedConsState = &solomachine.ConsensusState{}

				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: clienttypes.ErrInvalidConsensus,
		},
		{
			name: "unsuccessful upgrade: upgrade height revision height is more than the current client revision height",
			setup: func() {
				// upgrade Height is 10 blocks from now
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+10))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: committed client does not have zeroed custom fields",
			setup: func() {
				// non-zeroed upgrade client
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)
				upgradedClientBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: chain-specified parameters do not match committed client",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
				suite.Require().NoError(err)

				// change upgradedClient client-specified parameters
				upgradedClient = ibctm.NewClientState("wrongchainID", ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: client-specified parameters do not match previous client",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
				suite.Require().NoError(err)

				// change upgradedClient client-specified parameters
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ubdPeriod, ubdPeriod+trustingPeriod, maxClockDrift+5, lastHeight, commitmenttypes.GetSDKSpecs(), upgradePath)

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: upgrade client is not tendermint",
			setup: func() {
				upgradedClient = &solomachine.ClientState{}
			},
			expErr: clienttypes.ErrInvalidClientType,
		},
		{
			name: "unsuccessful upgrade: relayer-submitted consensus state does not match counterparty-committed consensus state",
			setup: func() {
				// change submitted upgradedConsensusState
				upgradedConsState = &ibctm.ConsensusState{
					NextValidatorsHash: []byte("maliciousValidators"),
				}

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: client proof unmarshal failed",
			setup: func() {
				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				upgradedClientProof = []byte("proof")
			},
			expErr: errors.New("could not unmarshal client merkle proof"),
		},
		{
			name: "unsuccessful upgrade: consensus state proof unmarshal failed",
			setup: func() {
				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				upgradedConsensusStateProof = []byte("proof")
			},
			expErr: errors.New("could not unmarshal consensus state merkle proof"),
		},
		{
			name: "unsuccessful upgrade: client proof verification failed",
			setup: func() {
				// do not store upgraded client

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: consensus state proof verification failed",
			setup: func() {
				// do not store upgraded client

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: client state merkle path is empty",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				// SetClientState with empty string upgrade path
				tmClient, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)
				tmClient.UpgradePath = []string{""}
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, tmClient)
			},
			expErr: errors.New("client state proof failed"),
		},
		{
			name: "unsuccessful upgrade: upgraded height is not greater than current height",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: errors.New("consensus state proof failed"),
		},
		{
			name: "unsuccessful upgrade: consensus state for upgrade height cannot be found",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+100))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: client is expired",
			setup: func() {
				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// expire chainB's client
				suite.chainA.ExpireClient(ubdPeriod)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: updated unbonding period is equal to trusting period",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: final client is not valid",
			setup: func() {
				// new client has smaller unbonding period such that old trusting period is no longer valid
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, trustingPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)
				upgradedClientBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: consensus state not found for latest height",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				revisionHeight := tmCs.LatestHeight.GetRevisionHeight()

				// set latest height to a height where consensus state does not exist
				tmCs.LatestHeight = clienttypes.NewHeight(tmCs.LatestHeight.GetRevisionNumber(), tmCs.LatestHeight.GetRevisionHeight()+5)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, tmCs)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), revisionHeight)
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), revisionHeight)
			},
			expErr: clienttypes.ErrConsensusStateNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			path.SetupClients()

			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

			var err error
			newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
			suite.Require().NoError(err)

			upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)

			if upgraded, ok := upgradedClient.(*ibctm.ClientState); ok {
				upgradedClient = upgraded.ZeroCustomFields()
			}

			upgradedClientBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
			suite.Require().NoError(err)

			upgradedConsState = &ibctm.ConsensusState{
				NextValidatorsHash: []byte("nextValsHash"),
			}
			upgradedConsStateBz, err = clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
			suite.Require().NoError(err)

			tc.setup()

			cs, ok := suite.chainA.GetClientState(path.EndpointA.ClientID).(*ibctm.ClientState)
			suite.Require().True(ok)
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
			if upgraded, ok := upgradedClient.(*ibctm.ClientState); ok {
				upgradedClient = upgraded.ZeroCustomFields()
			}

			err = cs.VerifyUpgradeAndUpdateState(
				suite.chainA.GetContext(),
				suite.cdc,
				clientStore,
				upgradedClient,
				upgradedConsState,
				upgradedClientProof,
				upgradedConsensusStateProof,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err, "verify upgrade failed on valid case: %s", tc.name)

				clientState, ok := suite.chainA.GetClientState(path.EndpointA.ClientID).(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().NotNil(clientState, "verify upgrade failed on valid case: %s", tc.name)

				consensusState, found := suite.chainA.GetConsensusState(path.EndpointA.ClientID, clientState.LatestHeight)
				suite.Require().NotNil(consensusState, "verify upgrade failed on valid case: %s", tc.name)
				suite.Require().True(found)
			} else {
				suite.Require().ErrorContains(err, tc.expErr.Error(), "verify upgrade passed on invalid case: %s", tc.name)
			}
		})
	}
}
