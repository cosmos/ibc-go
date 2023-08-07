package types_test

import (
	"time"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	//host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	//solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// TestVerifyUpgrade currently only tests the interface into the contract.
// Test code is used in the grandpa contract.
// New client state, consensus state, and client metadata is expected to be set in the contract on success
/*func (suite *TypesTestSuite) TestVerifyUpgradeGrandpa() {
	var (
		upgradedClient         exported.ClientState
		upgradedConsState      exported.ConsensusState
		proofUpgradedClient    []byte
		proofUpgradedConsState []byte
		err                    error
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		// TODO: fails with check upgradedClient.GetLatestHeight().GT(lastHeight) in VerifyUpgradeAndUpdateState
		// {
		// 	"successful upgrade",
		// 	func() {},
		// 	true,
		// },
		{
			"unsuccessful upgrade: invalid new client state",
			func() {
				upgradedClient = &solomachine.ClientState{}
			},
			false,
		},
		{
			"unsuccessful upgrade: invalid new consensus state",
			func() {
				upgradedConsState = &solomachine.ConsensusState{}
			},
			false,
		},
		{
			"unsuccessful upgrade: invalid client state proof",
			func() {
				proofUpgradedClient = []byte("invalid client state proof")
			},
			false,
		},
		{
			"unsuccessful upgrade: invalid consensus state proof",
			func() {
				proofUpgradedConsState = []byte("invalid consensus state proof")
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupWasmGrandpaWithChannel()
			clientState, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, grandpaClientID)
			suite.Require().True(ok)
			upgradedClient = clientState
			upgradedConsState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetLatestClientConsensusState(suite.ctx, grandpaClientID)
			suite.Require().True(ok)
			proofUpgradedClient = []byte("upgraded client state proof")
			proofUpgradedConsState = []byte("upgraded consensus state proof")

			tc.setup()

			err = clientState.VerifyUpgradeAndUpdateState(
				suite.ctx,
				suite.chainA.Codec,
				suite.store,
				upgradedClient,
				upgradedConsState,
				proofUpgradedClient,
				proofUpgradedConsState,
			)

			if tc.expPass {
				suite.Require().NoError(err)
				clientStateBz := suite.store.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)
				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				// Stubbed code will increment client state
				suite.Require().Equal(clientState.GetLatestHeight().Increment(), newClientState.GetLatestHeight())
			} else {
				suite.Require().Error(err)
			}
		})
	}
}*/

func (suite *TypesTestSuite) TestVerifyUpgradeTendermint() {
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: true,
		},
		{
			name: "successful upgrade to same revision",
			setup: func() {
				var tmUpgradedClient exported.ClientState
				tmUpgradedClient = ibctm.NewClientState(suite.chainB.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(clienttypes.ParseChainID(suite.chainB.ChainID), upgradedClient.GetLatestHeight().GetRevisionHeight()+10), commitmenttypes.GetSDKSpecs(), upgradePath)
				tmUpgradedClient = tmUpgradedClient.ZeroCustomFields()
				tmUpgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), tmUpgradedClient)
				suite.Require().NoError(err)

				upgradedClient = types.NewClientState(tmUpgradedClientBz, suite.codeHash, clienttypes.NewHeight(tmUpgradedClient.GetLatestHeight().GetRevisionNumber(), tmUpgradedClient.GetLatestHeight().GetRevisionHeight()))
				upgradedClientBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: true,
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: committed client does not have zeroed custom fields",
			setup: func() {
				// non-zeroed upgrade client
				var tmUpgradedClient exported.ClientState //nolint:gosimple // ignore error for test
				tmUpgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)
				tmUpgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), tmUpgradedClient)
				suite.Require().NoError(err)

				upgradedClient = types.NewClientState(tmUpgradedClientBz, suite.codeHash, clienttypes.NewHeight(tmUpgradedClient.GetLatestHeight().GetRevisionNumber(), tmUpgradedClient.GetLatestHeight().GetRevisionHeight()))
				upgradedClientBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// commit upgrade store changes and update clients

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: chain-specified parameters do not match committed client",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// change upgradedClient client-specified parameters
				var tmUpgradedClient exported.ClientState
				tmUpgradedClient = ibctm.NewClientState("wrongchainID", ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)
				tmUpgradedClient = tmUpgradedClient.ZeroCustomFields()
				tmUpgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), tmUpgradedClient)
				suite.Require().NoError(err)

				upgradedClient = types.NewClientState(tmUpgradedClientBz, suite.codeHash, clienttypes.NewHeight(tmUpgradedClient.GetLatestHeight().GetRevisionNumber(), tmUpgradedClient.GetLatestHeight().GetRevisionHeight()))

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: client-specified parameters do not match previous client",
			setup: func() {
				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				// change upgradedClient client-specified parameters
				var tmUpgradedClient exported.ClientState
				tmUpgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ubdPeriod, ubdPeriod+trustingPeriod, maxClockDrift+5, lastHeight, commitmenttypes.GetSDKSpecs(), upgradePath)
				tmUpgradedClient = tmUpgradedClient.ZeroCustomFields()
				tmUpgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), tmUpgradedClient)
				suite.Require().NoError(err)

				upgradedClient = types.NewClientState(tmUpgradedClientBz, suite.codeHash, clienttypes.NewHeight(tmUpgradedClient.GetLatestHeight().GetRevisionNumber(), tmUpgradedClient.GetLatestHeight().GetRevisionHeight()))

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: relayer-submitted consensus state does not match counterparty-committed consensus state",
			setup: func() {
				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for test
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				var tmUpgradedConsState exported.ConsensusState //nolint:gosimple // ignore error for test
				tmUpgradedConsState = &ibctm.ConsensusState{
					Timestamp:          time.Now(),
					Root:               commitmenttypes.NewMerkleRoot([]byte("malicious root hash")),
					NextValidatorsHash: []byte("01234567890123456789012345678901"),
				}
				tmUpgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), tmUpgradedConsState)
				suite.Require().NoError(err)

				upgradedConsState = &types.ConsensusState{
					Data: tmUpgradedConsStateBz,
				}

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: client proof unmarshal failed",
			setup: func() {
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for test

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				proofUpgradedClient = []byte("proof")
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: consensus state proof unmarshal failed",
			setup: func() {
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz) //nolint:errcheck // ignore error for test

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				proofUpgradedConsState = []byte("proof")
			},
			expPass: false,
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: upgrade path is empty",
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				// SetClientState with empty upgrade path
				wasmClient, _ := cs.(*types.ClientState)
				var wasmClientData exported.ClientState
				err = suite.chainA.Codec.UnmarshalInterface(wasmClient.Data, &wasmClientData)
				suite.Require().NoError(err)
				tmClient := wasmClientData.(*ibctm.ClientState)
				tmClient.UpgradePath = []string{""}

				tmClientBz, err := suite.chainA.Codec.MarshalInterface(tmClient)
				suite.Require().NoError(err)
				wasmClient.Data = tmClientBz

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, wasmClient)
			},
			expPass: false,
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
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

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
		{
			name: "unsuccessful upgrade: final client is not valid",
			setup: func() {
				// new client has smaller unbonding period such that old trusting period is no longer valid
				var tmUpgradedClient exported.ClientState
				tmUpgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, trustingPeriod, maxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), upgradePath)
				tmUpgradedClient = tmUpgradedClient.ZeroCustomFields()
				tmUpgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), tmUpgradedClient)
				suite.Require().NoError(err)

				upgradedClient = types.NewClientState(tmUpgradedClientBz, suite.codeHash, clienttypes.NewHeight(tmUpgradedClient.GetLatestHeight().GetRevisionNumber(), tmUpgradedClient.GetLatestHeight().GetRevisionHeight()))
				upgradedClientBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)

				// upgrade Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)

				proofUpgradedClient, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupWasmTendermint()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			suite.coordinator.SetupClients(path)

			clientState := path.EndpointA.GetClientState().(*types.ClientState)
			var clientStateData exported.ClientState
			err = suite.chainA.Codec.UnmarshalInterface(clientState.Data, &clientStateData)
			suite.Require().NoError(err)
			tmClientState := clientStateData.(*ibctm.ClientState)

			revisionNumber := clienttypes.ParseChainID(tmClientState.ChainId)

			newChainID, err = clienttypes.SetRevisionNumber(tmClientState.ChainId, revisionNumber+1)
			suite.Require().NoError(err)

			var tmUpgradedClient exported.ClientState
			tmUpgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)
			tmUpgradedClient = tmUpgradedClient.ZeroCustomFields()
			tmUpgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), tmUpgradedClient)
			suite.Require().NoError(err)

			upgradedClient = types.NewClientState(tmUpgradedClientBz, clientState.CodeHash, clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1))
			upgradedClientBz, err = clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
			suite.Require().NoError(err)

			var tmUpgradedConsState exported.ConsensusState //nolint:gosimple // ignore error for test
			tmUpgradedConsState = &ibctm.ConsensusState{
				Timestamp:          time.Now(),
				Root:               commitmenttypes.NewMerkleRoot([]byte("root hash")),
				NextValidatorsHash: []byte("01234567890123456789012345678901"),
			}
			tmUpgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), tmUpgradedConsState)
			suite.Require().NoError(err)

			upgradedConsState = &types.ConsensusState{
				Data: tmUpgradedConsStateBz,
			}
			upgradedConsStateBz, err = clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
			suite.Require().NoError(err)

			tc.setup()

			cs := suite.chainA.GetClientState(path.EndpointA.ClientID)
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			err = cs.VerifyUpgradeAndUpdateState(
				suite.chainA.GetContext(),
				suite.chainA.Codec,
				clientStore,
				upgradedClient,
				upgradedConsState,
				proofUpgradedClient,
				proofUpgradedConsState,
			)

			if tc.expPass {
				suite.Require().NoError(err, "verify upgrade failed on valid case: %s", tc.name)

				clientState := suite.chainA.GetClientState(path.EndpointA.ClientID)
				suite.Require().NotNil(clientState, "verify upgrade failed on valid case: %s", tc.name)

				consensusState, found := suite.chainA.GetConsensusState(path.EndpointA.ClientID, clientState.GetLatestHeight())
				suite.Require().NotNil(consensusState, "verify upgrade failed on valid case: %s", tc.name)
				suite.Require().True(found)
			} else {
				suite.Require().Error(err, "verify upgrade passed on invalid case: %s", tc.name)
			}
		})
	}
}
