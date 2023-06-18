package tendermint_test

import (
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *TendermintTestSuite) TestVerifyUpgrade() {
	var (
		newChainID                                  string
		upgradedClient                              exported.ClientState
		upgradedConsState                           exported.ConsensusState
		lastHeight                                  clienttypes.Height
		path                                        *ibctesting.Path
		proofUpgradedClient, proofUpgradedConsState []byte
		upgradedClientBz, upgradedConsStateBz       []byte
		err                                         error
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			name: "successful upgrade",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: true,
		},
		{
			name: "successful upgrade to same revision",
			setup: func() {
				upgradedClient = ibctm.NewClientState(s.chainB.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(clienttypes.ParseChainID(s.chainB.ChainID), upgradedClient.GetLatestHeight().GetRevisionHeight()+10), commitmenttypes.GetSDKSpecs(), upgradePath)
				upgradedClient = upgradedClient.ZeroCustomFields()
				upgradedClientBz, err = clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradedClient)
				s.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: true,
		},

		{
			name: "unsuccessful upgrade: upgrade height revision height is more than the current client revision height",
			setup: func() {
				// upgrade Height is 10 blocks from now
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+10))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: committed client does not have zeroed custom fields",
			setup: func() {
				// non-zeroed upgrade client
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)
				upgradedClientBz, err = clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradedClient)
				s.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: chain-specified parameters do not match committed client",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// change upgradedClient client-specified parameters
				upgradedClient = ibctm.NewClientState("wrongchainID", ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: client-specified parameters do not match previous client",
			setup: func() {
				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// change upgradedClient client-specified parameters
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ubdPeriod, ubdPeriod+trustingPeriod, maxClockDrift+5, lastHeight, commitmenttypes.GetSDKSpecs(), upgradePath)

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: relayer-submitted consensus state does not match counterparty-committed consensus state",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// change submitted upgradedConsensusState
				upgradedConsState = &ibctm.ConsensusState{
					NextValidatorsHash: []byte("maliciousValidators"),
				}

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: client proof unmarshal failed",
			setup: func() {
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				proofUpgradedClient = []byte("proof")
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: consensus state proof unmarshal failed",
			setup: func() {
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				proofUpgradedConsState = []byte("proof")
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: client proof verification failed",
			setup: func() {
				// do not store upgraded client

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: consensus state proof verification failed",
			setup: func() {
				// do not store upgraded client

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: upgrade path is empty",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				// SetClientState with empty upgrade path
				tmClient, _ := cs.(*ibctm.ClientState)
				tmClient.UpgradePath = []string{""}
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, tmClient)
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: upgraded height is not greater than current height",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: consensus state for upgrade height cannot be found",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+100))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: client is expired",
			setup: func() {
				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				// expire chainB's client
				s.chainA.ExpireClient(ubdPeriod)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: updated unbonding period is equal to trusting period",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: final client is not valid",
			setup: func() {
				// new client has smaller unbonding period such that old trusting period is no longer valid
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, trustingPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)
				upgradedClientBz, err = clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradedClient)
				s.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		// reset suite
		s.SetupTest()
		path = ibctesting.NewPath(s.chainA, s.chainB)

		s.coordinator.SetupClients(path)

		clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
		revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

		var err error
		newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
		s.Require().NoError(err)

		upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)
		upgradedClient = upgradedClient.ZeroCustomFields()
		upgradedClientBz, err = clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradedClient)
		s.Require().NoError(err)

		upgradedConsState = &ibctm.ConsensusState{
			NextValidatorsHash: []byte("nextValsHash"),
		}
		upgradedConsStateBz, err = clienttypes.MarshalConsensusState(s.chainA.App.AppCodec(), upgradedConsState)
		s.Require().NoError(err)

		tc.setup()

		cs := s.chainA.GetClientState(path.EndpointA.ClientID)
		clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

		// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
		upgradedClient = upgradedClient.ZeroCustomFields()

		err = cs.VerifyUpgradeAndUpdateState(
			s.chainA.GetContext(),
			s.cdc,
			clientStore,
			upgradedClient,
			upgradedConsState,
			proofUpgradedClient,
			proofUpgradedConsState,
		)

		if tc.expPass {
			s.Require().NoError(err, "verify upgrade failed on valid case: %s", tc.name)

			clientState := s.chainA.GetClientState(path.EndpointA.ClientID)
			s.Require().NotNil(clientState, "verify upgrade failed on valid case: %s", tc.name)

			consensusState, found := s.chainA.GetConsensusState(path.EndpointA.ClientID, clientState.GetLatestHeight())
			s.Require().NotNil(consensusState, "verify upgrade failed on valid case: %s", tc.name)
			s.Require().True(found)
		} else {
			s.Require().Error(err, "verify upgrade passed on invalid case: %s", tc.name)
		}
	}
}
