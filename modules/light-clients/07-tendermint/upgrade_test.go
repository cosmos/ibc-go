package tendermint_test

import (
	"errors"
	"time"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *TendermintTestSuite) TestVerifyUpgrade() {
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: nil,
		},
		{
			name: "successful upgrade to same revision",
			setup: func() {
				upgradedClient = ibctm.NewClientState(s.chainB.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(clienttypes.ParseChainID(s.chainB.ChainID), upgradedClient.(*ibctm.ClientState).LatestHeight.GetRevisionHeight()+10), commitmenttypes.GetSDKSpecs(), upgradePath)
				upgradedClient = upgradedClient.(*ibctm.ClientState).ZeroCustomFields()
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: nil,
		},
		{
			name: "successful upgrade with new unbonding period",
			setup: func() {
				newUnbondingPeriod := time.Hour * 24 * 7 * 2
				upgradedClient = ibctm.NewClientState(s.chainB.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, newUnbondingPeriod, maxClockDrift, clienttypes.NewHeight(clienttypes.ParseChainID(s.chainB.ChainID), upgradedClient.(*ibctm.ClientState).LatestHeight.GetRevisionHeight()+10), commitmenttypes.GetSDKSpecs(), upgradePath)
				upgradedClient = upgradedClient.(*ibctm.ClientState).ZeroCustomFields()
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: nil,
		},
		{
			name: "unsuccessful upgrade: upgrade path not set",
			setup: func() {
				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				// set upgrade path to empty
				tmCs.UpgradePath = []string{}
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, tmCs)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: clienttypes.ErrInvalidUpgradeClient,
		},
		{
			name: "unsuccessful upgrade: upgrade consensus state must be tendermint consensus state",
			setup: func() {
				upgradedConsState = &solomachine.ConsensusState{}

				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: clienttypes.ErrInvalidConsensus,
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: chain-specified parameters do not match committed client",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
				s.Require().NoError(err)

				// change upgradedClient client-specified parameters
				upgradedClient = ibctm.NewClientState("wrongchainID", ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)

				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: client-specified parameters do not match previous client",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
				s.Require().NoError(err)

				// change upgradedClient client-specified parameters
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ubdPeriod, ubdPeriod+trustingPeriod, maxClockDrift+5, lastHeight, commitmenttypes.GetSDKSpecs(), upgradePath)

				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
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
				s.coordinator.CommitBlock(s.chainB)
				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: client proof unmarshal failed",
			setup: func() {
				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				upgradedClientProof = []byte("proof")
			},
			expErr: errors.New("could not unmarshal client merkle proof"),
		},
		{
			name: "unsuccessful upgrade: consensus state proof unmarshal failed",
			setup: func() {
				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				upgradedConsensusStateProof = []byte("proof")
			},
			expErr: errors.New("could not unmarshal consensus state merkle proof"),
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: client state merkle path is empty",
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				// SetClientState with empty string upgrade path
				tmClient, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClient.UpgradePath = []string{""}
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, tmClient)
			},
			expErr: errors.New("client state proof failed"),
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: errors.New("consensus state proof failed"),
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "unsuccessful upgrade: consensus state not found for latest height",
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
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				revisionHeight := tmCs.LatestHeight.GetRevisionHeight()

				// set latest height to a height where consensus state does not exist
				tmCs.LatestHeight = clienttypes.NewHeight(tmCs.LatestHeight.GetRevisionNumber(), tmCs.LatestHeight.GetRevisionHeight()+5)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, tmCs)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), revisionHeight)
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), revisionHeight)
			},
			expErr: clienttypes.ErrConsensusStateNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// reset suite
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)

			path.SetupClients()

			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

			var err error
			newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
			s.Require().NoError(err)

			upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)

			if upgraded, ok := upgradedClient.(*ibctm.ClientState); ok {
				upgradedClient = upgraded.ZeroCustomFields()
			}

			upgradedClientBz, err = clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradedClient)
			s.Require().NoError(err)

			upgradedConsState = &ibctm.ConsensusState{
				NextValidatorsHash: []byte("nextValsHash"),
			}
			upgradedConsStateBz, err = clienttypes.MarshalConsensusState(s.chainA.App.AppCodec(), upgradedConsState)
			s.Require().NoError(err)

			tc.setup()

			cs, ok := s.chainA.GetClientState(path.EndpointA.ClientID).(*ibctm.ClientState)
			s.Require().True(ok)
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
			if upgraded, ok := upgradedClient.(*ibctm.ClientState); ok {
				upgradedClient = upgraded.ZeroCustomFields()
			}

			err = cs.VerifyUpgradeAndUpdateState(
				s.chainA.GetContext(),
				s.cdc,
				clientStore,
				upgradedClient,
				upgradedConsState,
				upgradedClientProof,
				upgradedConsensusStateProof,
			)

			if tc.expErr == nil {
				s.Require().NoError(err, "verify upgrade failed on valid case: %s", tc.name)

				clientState, ok := s.chainA.GetClientState(path.EndpointA.ClientID).(*ibctm.ClientState)
				s.Require().True(ok)
				s.Require().NotNil(clientState, "verify upgrade failed on valid case: %s", tc.name)

				consensusState, found := s.chainA.GetConsensusState(path.EndpointA.ClientID, clientState.LatestHeight)
				s.Require().NotNil(consensusState, "verify upgrade failed on valid case: %s", tc.name)
				s.Require().True(found)
			} else {
				s.Require().ErrorContains(err, tc.expErr.Error(), "verify upgrade passed on invalid case: %s", tc.name)
			}
		})
	}
}

func (s *TendermintTestSuite) TestVerifyUpgradeWithNewUnbonding() {
	s.SetupTest()
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
	s.Require().True(ok)

	newUnbondingPeriod := time.Hour * 24 * 7 * 2 // update the unbonding period to two weeks
	upgradeClient := ibctm.NewClientState(clientState.ChainId, ibctm.DefaultTrustLevel, trustingPeriod, newUnbondingPeriod, maxClockDrift, clienttypes.NewHeight(1, clientState.LatestHeight.GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)

	upgradedClientBz, err := clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradeClient.ZeroCustomFields())
	s.Require().NoError(err)

	upgradedConsState := &ibctm.ConsensusState{NextValidatorsHash: []byte("nextValsHash")} // mocked consensus state
	upgradedConsStateBz, err := clienttypes.MarshalConsensusState(s.chainA.App.AppCodec(), upgradedConsState)
	s.Require().NoError(err)

	// zero custom fields and store in chainB upgrade store
	upgradeHeight := clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1)) // upgrade is at next block height
	err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), upgradedClientBz)
	s.Require().NoError(err)
	err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), upgradedConsStateBz)
	s.Require().NoError(err)

	// commit upgrade store changes on chainB and update client on chainA
	s.coordinator.CommitBlock(s.chainB)

	err = path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	upgradedClientProof, _ := s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), uint64(s.chainB.LatestCommittedHeader.Header.Height))
	upgradedConsensusStateProof, _ := s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), uint64(s.chainB.LatestCommittedHeader.Header.Height))

	tmClientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
	s.Require().True(ok)

	clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
	err = tmClientState.VerifyUpgradeAndUpdateState(
		s.chainA.GetContext(),
		s.cdc,
		clientStore,
		upgradeClient,
		upgradedConsState,
		upgradedClientProof,
		upgradedConsensusStateProof,
	)
	s.Require().NoError(err)

	upgradedClient, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
	s.Require().True(ok)

	// assert the unbonding period and the trusting period have been updated correctly
	s.Require().Equal(newUnbondingPeriod, upgradedClient.UnbondingPeriod)

	// expected trusting period = trustingPeriod * newUnbonding / originalUnbonding (224 hours = 9 days and 8 hours)
	origUnbondingDec := sdkmath.LegacyNewDec(ubdPeriod.Nanoseconds())
	newUnbondingDec := sdkmath.LegacyNewDec(newUnbondingPeriod.Nanoseconds())
	trustingPeriodDec := sdkmath.LegacyNewDec(trustingPeriod.Nanoseconds())

	expTrustingPeriod := trustingPeriodDec.Mul(newUnbondingDec).Quo(origUnbondingDec)
	s.Require().Equal(time.Duration(expTrustingPeriod.TruncateInt64()), upgradedClient.TrustingPeriod)
}
