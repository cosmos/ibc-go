package keeper_test

import (
	"fmt"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestCreateClient() {
	var (
		clientState    []byte
		consensusState []byte
	)

	testCases := []struct {
		msg        string
		malleate   func()
		clientType string
		expPass    bool
	}{
		{
			"success: 07-tendermint client type supported",
			func() {
				tmClientState := ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				clientState = suite.chainA.App.AppCodec().MustMarshal(tmClientState)
				consensusState = suite.chainA.App.AppCodec().MustMarshal(suite.consensusState)
			},
			exported.Tendermint,
			true,
		},
		{
			"failure: 07-tendermint client status is not active",
			func() {
				tmClientState := ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				tmClientState.FrozenHeight = ibctm.FrozenHeight
				clientState = suite.chainA.App.AppCodec().MustMarshal(tmClientState)
				consensusState = suite.chainA.App.AppCodec().MustMarshal(suite.consensusState)
			},
			exported.Tendermint,
			false,
		},
		{
			"success: 06-solomachine client type supported",
			func() {
				smClientState := solomachine.NewClientState(1, &solomachine.ConsensusState{PublicKey: suite.solomachine.ConsensusState().PublicKey, Diversifier: suite.solomachine.Diversifier, Timestamp: suite.solomachine.Time})
				smConsensusState := &solomachine.ConsensusState{PublicKey: suite.solomachine.ConsensusState().PublicKey, Diversifier: suite.solomachine.Diversifier, Timestamp: suite.solomachine.Time}
				clientState = suite.chainA.App.AppCodec().MustMarshal(smClientState)
				consensusState = suite.chainA.App.AppCodec().MustMarshal(smConsensusState)
			},
			exported.Solomachine,
			true,
		},
		{
			"failure: 09-localhost client type not supported",
			func() {},
			exported.Localhost,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			clientState, consensusState = []byte{}, []byte{}

			tc.malleate()

			clientID, err := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.CreateClient(suite.chainA.GetContext(), tc.clientType, clientState, consensusState)

			// assert correct behaviour based on expected error
			clientState, found := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetClientState(suite.chainA.GetContext(), clientID)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(clientID)
				suite.Require().True(found)
				suite.Require().NotEmpty(clientState)
			} else {
				suite.Require().Error(err)
				suite.Require().Empty(clientID)
				suite.Require().False(found)
				suite.Require().Empty(clientState)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateClientTendermint() {
	var (
		path         *ibctesting.Path
		updateHeader *ibctm.Header
	)

	// Must create header creation functions since suite.header gets recreated on each test case
	createFutureUpdateFn := func(trustedHeight clienttypes.Height) *ibctm.Header {
		header, err := path.EndpointB.Chain.IBCClientHeader(path.EndpointB.Chain.LatestCommittedHeader, trustedHeight)
		suite.Require().NoError(err)
		return header
	}
	createPastUpdateFn := func(fillHeight, trustedHeight clienttypes.Height) *ibctm.Header {
		consState, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(suite.chainA.GetContext(), path.EndpointA.ClientID, trustedHeight)
		suite.Require().True(found)

		return suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(fillHeight.RevisionHeight), trustedHeight, consState.(*ibctm.ConsensusState).Timestamp.Add(time.Second*5),
			suite.chainB.Vals, suite.chainB.Vals, suite.chainB.Vals, suite.chainB.Signers)
	}

	cases := []struct {
		name      string
		malleate  func()
		expPass   bool
		expFreeze bool
	}{
		{"valid update", func() {
			trustedHeight := path.EndpointA.GetClientLatestHeight()

			// store intermediate consensus state to check that trustedHeight does not need to be highest consensus state before header height
			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			updateHeader = createFutureUpdateFn(trustedHeight.(clienttypes.Height))
		}, true, false},
		{"valid past update", func() {
			trustedHeight := path.EndpointA.GetClientLatestHeight()

			currHeight := suite.chainB.ProposedHeader.Height
			fillHeight := clienttypes.NewHeight(trustedHeight.GetRevisionNumber(), uint64(currHeight))

			// commit a couple blocks to allow client to fill in gaps
			suite.coordinator.CommitBlock(suite.chainB) // this height is not filled in yet
			suite.coordinator.CommitBlock(suite.chainB) // this height is filled in by the update below

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			// ensure fill height not set
			_, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(suite.chainA.GetContext(), path.EndpointA.ClientID, fillHeight)
			suite.Require().False(found)

			// updateHeader will fill in consensus state between prevConsState and suite.consState
			// clientState should not be updated
			updateHeader = createPastUpdateFn(fillHeight, trustedHeight.(clienttypes.Height))
		}, true, false},
		{"valid duplicate update", func() {
			height1 := clienttypes.NewHeight(1, 1)

			// store previous consensus state
			prevConsState := &ibctm.ConsensusState{
				Timestamp:          suite.past,
				NextValidatorsHash: suite.chainB.Vals.Hash(),
			}
			path.EndpointA.SetConsensusState(prevConsState, height1)

			height5 := clienttypes.NewHeight(1, 5)
			// store next consensus state to check that trustedHeight does not need to be highest consensus state before header height
			nextConsState := &ibctm.ConsensusState{
				Timestamp:          suite.past.Add(time.Minute),
				NextValidatorsHash: suite.chainB.Vals.Hash(),
			}
			path.EndpointA.SetConsensusState(nextConsState, height5)

			// update client state latest height
			clientState := path.EndpointA.GetClientState()
			clientState.(*ibctm.ClientState).LatestHeight = height5
			path.EndpointA.SetClientState(clientState)

			height3 := clienttypes.NewHeight(1, 3)
			// updateHeader will fill in consensus state between prevConsState and suite.consState
			// clientState should not be updated
			updateHeader = createPastUpdateFn(height3, height1)
			// set updateHeader's consensus state in store to create duplicate UpdateClient scenario
			path.EndpointA.SetConsensusState(updateHeader.ConsensusState(), updateHeader.GetHeight())
		}, true, false},
		{"misbehaviour detection: conflicting header", func() {
			clientID := path.EndpointA.ClientID

			height1 := clienttypes.NewHeight(1, 1)
			// store previous consensus state
			prevConsState := &ibctm.ConsensusState{
				Timestamp:          suite.past,
				NextValidatorsHash: suite.chainB.Vals.Hash(),
			}
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientID, height1, prevConsState)

			height5 := clienttypes.NewHeight(1, 5)
			// store next consensus state to check that trustedHeight does not need to be highest consensus state before header height
			nextConsState := &ibctm.ConsensusState{
				Timestamp:          suite.past.Add(time.Minute),
				NextValidatorsHash: suite.chainB.Vals.Hash(),
			}
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientID, height5, nextConsState)

			height3 := clienttypes.NewHeight(1, 3)
			// updateHeader will fill in consensus state between prevConsState and suite.consState
			// clientState should not be updated
			updateHeader = createPastUpdateFn(height3, height1)
			// set conflicting consensus state in store to create misbehaviour scenario
			conflictConsState := updateHeader.ConsensusState()
			conflictConsState.Root = commitmenttypes.NewMerkleRoot([]byte("conflicting apphash"))
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientID, updateHeader.GetHeight(), conflictConsState)
		}, true, true},
		{"misbehaviour detection: monotonic time violation", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientID := path.EndpointA.ClientID
			trustedHeight := clientState.LatestHeight

			// store intermediate consensus state at a time greater than updateHeader time
			// this will break time monotonicity
			incrementedClientHeight, ok := clientState.LatestHeight.Increment().(clienttypes.Height)
			suite.Require().True(ok)
			intermediateConsState := &ibctm.ConsensusState{
				Timestamp:          suite.coordinator.CurrentTime.Add(2 * time.Hour),
				NextValidatorsHash: suite.chainB.Vals.Hash(),
			}
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientID, incrementedClientHeight, intermediateConsState)
			// set iteration key
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)
			ibctm.SetIterationKey(clientStore, incrementedClientHeight)

			clientState.LatestHeight = incrementedClientHeight
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

			updateHeader = createFutureUpdateFn(trustedHeight)
		}, true, true},
		{"client state not found", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			updateHeader = createFutureUpdateFn(clientState.LatestHeight)

			path.EndpointA.ClientID = ibctesting.InvalidID
		}, false, false},
		{"consensus state not found", func() {
			clientState := path.EndpointA.GetClientState()
			tmClient, ok := clientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			tmClient.LatestHeight, ok = tmClient.LatestHeight.Increment().(clienttypes.Height)
			suite.Require().True(ok)

			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, clientState)
			updateHeader = createFutureUpdateFn(tmClient.LatestHeight)
		}, false, false},
		{"client is not active", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(1, 1)
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, clientState)
			updateHeader = createFutureUpdateFn(clientState.LatestHeight)
		}, false, false},
		{"invalid header", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			updateHeader = createFutureUpdateFn(clientState.LatestHeight)
			updateHeader.TrustedHeight, ok = updateHeader.TrustedHeight.Increment().(clienttypes.Height)
			suite.Require().True(ok)
		}, false, false},
	}

	for _, tc := range cases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			tc.malleate()

			var clientState *ibctm.ClientState
			var ok bool
			if tc.expPass {
				clientState, ok = path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
			}

			err := suite.chainA.App.GetIBCKeeper().ClientKeeper.UpdateClient(suite.chainA.GetContext(), path.EndpointA.ClientID, updateHeader)

			if tc.expPass {
				suite.Require().NoError(err, err)

				newClientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)

				if tc.expFreeze {
					suite.Require().True(!newClientState.FrozenHeight.IsZero(), "client did not freeze after conflicting header was submitted to UpdateClient")
				} else {
					expConsensusState := &ibctm.ConsensusState{
						Timestamp:          updateHeader.GetTime(),
						Root:               commitmenttypes.NewMerkleRoot(updateHeader.Header.GetAppHash()),
						NextValidatorsHash: updateHeader.Header.NextValidatorsHash,
					}

					consensusState, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(suite.chainA.GetContext(), path.EndpointA.ClientID, updateHeader.GetHeight())
					suite.Require().True(found)

					// Determine if clientState should be updated or not
					if updateHeader.GetHeight().GT(clientState.LatestHeight) {
						// Header Height is greater than clientState latest Height, clientState should be updated with header.GetHeight()
						suite.Require().Equal(updateHeader.GetHeight(), newClientState.LatestHeight, "clientstate height did not update")
					} else {
						// Update will add past consensus state, clientState should not be updated at all
						suite.Require().Equal(clientState.LatestHeight, newClientState.LatestHeight, "client state height updated for past header")
					}

					suite.Require().NoError(err)
					suite.Require().Equal(expConsensusState, consensusState, "consensus state should have been updated on case %s", tc.name)
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpgradeClient() {
	var (
		path                                             *ibctesting.Path
		upgradedClient                                   *ibctm.ClientState
		upgradedConsState                                exported.ConsensusState
		upgradeHeight                                    exported.Height
		upgradedClientAny, upgradedConsStateAny          *codectypes.Any
		upgradedClientProof, upgradedConsensusStateProof []byte
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
				upgradeHeight = clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedClientAny))
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedConsStateAny))
				suite.Require().NoError(err)

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expPass: true,
		},
		{
			name: "client state not found",
			setup: func() {
				// upgrade height is at next block
				upgradeHeight = clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedClientAny))
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedConsStateAny))
				suite.Require().NoError(err)

				// commit upgrade store changes and update clients

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				path.EndpointA.ClientID = "wrongclientid"
			},
			expPass: false,
		},
		{
			name: "client state is not active",
			setup: func() {
				// client is frozen

				// upgrade height is at next block
				upgradeHeight = clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedClientAny))
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedConsStateAny))
				suite.Require().NoError(err)

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				// set frozen client in store
				tmClient, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)
				tmClient.FrozenHeight = clienttypes.NewHeight(1, 1)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, tmClient)
			},
			expPass: false,
		},
		{
			name: "light client module VerifyUpgradeAndUpdateState fails",
			setup: func() {
				// upgrade height is at next block
				upgradeHeight = clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedClientAny))
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedConsStateAny))
				suite.Require().NoError(err)

				// change upgradedClient client-specified parameters
				upgradedClient.ChainId = "wrongchainID"
				upgradedClientAny, err = codectypes.NewAnyWithValue(upgradedClient)
				suite.Require().NoError(err)

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				suite.Require().True(ok)

				upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

			newChainID, err := clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
			suite.Require().NoError(err)

			upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			upgradedClient = upgradedClient.ZeroCustomFields()

			upgradedClientAny, err = codectypes.NewAnyWithValue(upgradedClient)
			suite.Require().NoError(err)

			upgradedConsState = &ibctm.ConsensusState{NextValidatorsHash: []byte("nextValsHash")}

			upgradedConsStateAny, err = codectypes.NewAnyWithValue(upgradedConsState)
			suite.Require().NoError(err)

			tc.setup()

			err = suite.chainA.App.GetIBCKeeper().ClientKeeper.UpgradeClient(suite.chainA.GetContext(), path.EndpointA.ClientID, upgradedClientAny.Value, upgradedConsStateAny.Value, upgradedClientProof, upgradedConsensusStateProof)

			if tc.expPass {
				suite.Require().NoError(err, "verify upgrade failed on valid case: %s", tc.name)
			} else {
				suite.Require().Error(err, "verify upgrade passed on invalid case: %s", tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateClientEventEmission() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupClients()

	tmClientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
	suite.Require().True(ok)
	trustedHeight := tmClientState.LatestHeight
	header, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
	suite.Require().NoError(err)

	msg, err := clienttypes.NewMsgUpdateClient(
		path.EndpointA.ClientID, header,
		suite.chainA.SenderAccount.GetAddress().String(),
	)
	suite.Require().NoError(err)

	result, err := suite.chainA.SendMsgs(msg)

	// check that update client event was emitted
	suite.Require().NoError(err)
	var event abci.Event
	for _, e := range result.Events {
		if e.Type == clienttypes.EventTypeUpdateClient {
			event = e
		}
	}
	suite.Require().NotNil(event)
}

func (suite *KeeperTestSuite) TestRecoverClient() {
	var (
		subject, substitute                       string
		subjectClientState, substituteClientState exported.ClientState
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"success, subject and substitute use different revision number",
			func() {
				tmClientState, ok := substituteClientState.(*ibctm.ClientState)
				suite.Require().True(ok)
				consState, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(suite.chainA.GetContext(), substitute, tmClientState.LatestHeight)
				suite.Require().True(found)
				newRevisionNumber := tmClientState.LatestHeight.GetRevisionNumber() + 1

				tmClientState.LatestHeight = clienttypes.NewHeight(newRevisionNumber, tmClientState.LatestHeight.GetRevisionHeight())

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), substitute, tmClientState.LatestHeight, consState)
				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substitute)
				ibctm.SetProcessedTime(clientStore, tmClientState.LatestHeight, 100)
				ibctm.SetProcessedHeight(clientStore, tmClientState.LatestHeight, clienttypes.NewHeight(0, 1))
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitute, tmClientState)
			},
			nil,
		},
		{
			"subject client does not exist",
			func() {
				subject = ibctesting.InvalidID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"subject is Active",
			func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				suite.Require().True(ok)
				// Set FrozenHeight to zero to ensure client is reported as Active
				tmClientState.FrozenHeight = clienttypes.ZeroHeight()
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)
			},
			clienttypes.ErrInvalidRecoveryClient,
		},
		{
			"substitute client does not exist",
			func() {
				substitute = ibctesting.InvalidID
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"subject and substitute have equal latest height",
			func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				suite.Require().True(ok)
				tmClientState.LatestHeight = substituteClientState.(*ibctm.ClientState).LatestHeight
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)
			},
			clienttypes.ErrInvalidHeight,
		},
		{
			"subject height is greater than substitute height",
			func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				suite.Require().True(ok)
				tmClientState.LatestHeight, ok = substituteClientState.(*ibctm.ClientState).LatestHeight.Increment().(clienttypes.Height)
				suite.Require().True(ok)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)
			},
			clienttypes.ErrInvalidHeight,
		},
		{
			"substitute is frozen",
			func() {
				tmClientState, ok := substituteClientState.(*ibctm.ClientState)
				suite.Require().True(ok)
				tmClientState.FrozenHeight = clienttypes.NewHeight(0, 1)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitute, tmClientState)
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"light client module RecoverClient fails, substitute client trust level doesn't match subject client trust level",
			func() {
				tmClientState, ok := substituteClientState.(*ibctm.ClientState)
				suite.Require().True(ok)
				tmClientState.UnbondingPeriod += time.Minute
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), substitute, tmClientState)
			},
			clienttypes.ErrInvalidSubstitute,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			subjectPath.SetupClients()
			subject = subjectPath.EndpointA.ClientID
			subjectClientState = suite.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath.SetupClients()
			substitute = substitutePath.EndpointA.ClientID

			// update substitute twice
			err := substitutePath.EndpointA.UpdateClient()
			suite.Require().NoError(err)
			err = substitutePath.EndpointA.UpdateClient()
			suite.Require().NoError(err)
			substituteClientState = suite.chainA.GetClientState(substitute)

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)

			tc.malleate()

			ctx := suite.chainA.GetContext()
			err = suite.chainA.App.GetIBCKeeper().ClientKeeper.RecoverClient(ctx, subject, substitute)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				expectedEvents := sdk.Events{
					sdk.NewEvent(
						clienttypes.EventTypeRecoverClient,
						sdk.NewAttribute(clienttypes.AttributeKeySubjectClientID, subjectPath.EndpointA.ClientID),
						sdk.NewAttribute(clienttypes.AttributeKeyClientType, subjectPath.EndpointA.GetClientState().ClientType()),
					),
				}.ToABCIEvents()

				expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
				ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())

				// Assert that client status is now Active
				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
				tmClientState, ok := subjectPath.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().Equal(tmClientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.App.AppCodec()), exported.Active)

			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
