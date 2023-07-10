package keeper_test

import (
	"encoding/hex"
	"fmt"
	"time"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	localhost "github.com/cosmos/ibc-go/v7/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestCreateClient() {
	var (
		clientState    exported.ClientState
		consensusState exported.ConsensusState
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success: 07-tendermint client type supported",
			func() {
				clientState = ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				consensusState = s.consensusState
			},
			true,
		},
		{
			"failure: 07-tendermint client status is not active",
			func() {
				clientState = ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				tmcs, ok := clientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmcs.FrozenHeight = ibctm.FrozenHeight
				consensusState = s.consensusState
			},
			false,
		},
		{
			"success: 06-solomachine client type supported",
			func() {
				clientState = solomachine.NewClientState(0, &solomachine.ConsensusState{PublicKey: s.solomachine.ConsensusState().PublicKey, Diversifier: s.solomachine.Diversifier, Timestamp: s.solomachine.Time})
				consensusState = &solomachine.ConsensusState{PublicKey: s.solomachine.ConsensusState().PublicKey, Diversifier: s.solomachine.Diversifier, Timestamp: s.solomachine.Time}
			},
			true,
		},
		{
			"failure: 09-localhost client type not supported",
			func() {
				clientState = localhost.NewClientState(clienttypes.GetSelfHeight(s.chainA.GetContext()))
				consensusState = nil
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset
			tc.malleate()

			clientID, err := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.CreateClient(s.chainA.GetContext(), clientState, consensusState)

			// assert correct behaviour based on expected error
			clientState, found := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetClientState(s.chainA.GetContext(), clientID)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotEmpty(clientID)
				s.Require().True(found)
				s.Require().NotEmpty(clientState)
			} else {
				s.Require().Error(err)
				s.Require().Empty(clientID)
				s.Require().False(found)
				s.Require().Empty(clientState)
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpdateClientTendermint() {
	var (
		path         *ibctesting.Path
		updateHeader *ibctm.Header
	)

	// Must create header creation functions since s.header gets recreated on each test case
	createFutureUpdateFn := func(trustedHeight clienttypes.Height) *ibctm.Header {
		header, err := s.chainA.ConstructUpdateTMClientHeaderWithTrustedHeight(path.EndpointB.Chain, path.EndpointA.ClientID, trustedHeight)
		s.Require().NoError(err)
		return header
	}
	createPastUpdateFn := func(fillHeight, trustedHeight clienttypes.Height) *ibctm.Header {
		consState, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID, trustedHeight)
		s.Require().True(found)

		return s.chainB.CreateTMClientHeader(s.chainB.ChainID, int64(fillHeight.RevisionHeight), trustedHeight, consState.(*ibctm.ConsensusState).Timestamp.Add(time.Second*5),
			s.chainB.Vals, s.chainB.Vals, s.chainB.Vals, s.chainB.Signers)
	}

	cases := []struct {
		name      string
		malleate  func()
		expPass   bool
		expFreeze bool
	}{
		{"valid update", func() {
			clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
			trustHeight := clientState.GetLatestHeight().(clienttypes.Height)

			// store intermediate consensus state to check that trustedHeight does not need to be highest consensus state before header height
			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			updateHeader = createFutureUpdateFn(trustHeight)
		}, true, false},
		{"valid past update", func() {
			clientState := path.EndpointA.GetClientState()
			trustedHeight := clientState.GetLatestHeight().(clienttypes.Height)

			currHeight := s.chainB.CurrentHeader.Height
			fillHeight := clienttypes.NewHeight(clientState.GetLatestHeight().GetRevisionNumber(), uint64(currHeight))

			// commit a couple blocks to allow client to fill in gaps
			s.coordinator.CommitBlock(s.chainB) // this height is not filled in yet
			s.coordinator.CommitBlock(s.chainB) // this height is filled in by the update below

			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			// ensure fill height not set
			_, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID, fillHeight)
			s.Require().False(found)

			// updateHeader will fill in consensus state between prevConsState and s.consState
			// clientState should not be updated
			updateHeader = createPastUpdateFn(fillHeight, trustedHeight)
		}, true, false},
		{"valid duplicate update", func() {
			height1 := clienttypes.NewHeight(1, 1)

			// store previous consensus state
			prevConsState := &ibctm.ConsensusState{
				Timestamp:          s.past,
				NextValidatorsHash: s.chainB.Vals.Hash(),
			}
			path.EndpointA.SetConsensusState(prevConsState, height1)

			height5 := clienttypes.NewHeight(1, 5)
			// store next consensus state to check that trustedHeight does not need to be highest consensus state before header height
			nextConsState := &ibctm.ConsensusState{
				Timestamp:          s.past.Add(time.Minute),
				NextValidatorsHash: s.chainB.Vals.Hash(),
			}
			path.EndpointA.SetConsensusState(nextConsState, height5)

			// update client state latest height
			clientState := path.EndpointA.GetClientState()
			clientState.(*ibctm.ClientState).LatestHeight = height5
			path.EndpointA.SetClientState(clientState)

			height3 := clienttypes.NewHeight(1, 3)
			// updateHeader will fill in consensus state between prevConsState and s.consState
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
				Timestamp:          s.past,
				NextValidatorsHash: s.chainB.Vals.Hash(),
			}
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientID, height1, prevConsState)

			height5 := clienttypes.NewHeight(1, 5)
			// store next consensus state to check that trustedHeight does not need to be hightest consensus state before header height
			nextConsState := &ibctm.ConsensusState{
				Timestamp:          s.past.Add(time.Minute),
				NextValidatorsHash: s.chainB.Vals.Hash(),
			}
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientID, height5, nextConsState)

			height3 := clienttypes.NewHeight(1, 3)
			// updateHeader will fill in consensus state between prevConsState and s.consState
			// clientState should not be updated
			updateHeader = createPastUpdateFn(height3, height1)
			// set conflicting consensus state in store to create misbehaviour scenario
			conflictConsState := updateHeader.ConsensusState()
			conflictConsState.Root = commitmenttypes.NewMerkleRoot([]byte("conflicting apphash"))
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientID, updateHeader.GetHeight(), conflictConsState)
		}, true, true},
		{"misbehaviour detection: monotonic time violation", func() {
			clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
			clientID := path.EndpointA.ClientID
			trustedHeight := clientState.GetLatestHeight().(clienttypes.Height)

			// store intermediate consensus state at a time greater than updateHeader time
			// this will break time monotonicity
			incrementedClientHeight := clientState.GetLatestHeight().Increment().(clienttypes.Height)
			intermediateConsState := &ibctm.ConsensusState{
				Timestamp:          s.coordinator.CurrentTime.Add(2 * time.Hour),
				NextValidatorsHash: s.chainB.Vals.Hash(),
			}
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientID, incrementedClientHeight, intermediateConsState)
			// set iteration key
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), clientID)
			ibctm.SetIterationKey(clientStore, incrementedClientHeight)

			clientState.LatestHeight = incrementedClientHeight
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)

			updateHeader = createFutureUpdateFn(trustedHeight)
		}, true, true},
		{"client state not found", func() {
			updateHeader = createFutureUpdateFn(path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height))

			path.EndpointA.ClientID = ibctesting.InvalidID
		}, false, false},
		{"consensus state not found", func() {
			clientState := path.EndpointA.GetClientState()
			tmClient, ok := clientState.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClient.LatestHeight = tmClient.LatestHeight.Increment().(clienttypes.Height)

			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, clientState)
			updateHeader = createFutureUpdateFn(clientState.GetLatestHeight().(clienttypes.Height))
		}, false, false},
		{"client is not active", func() {
			clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
			clientState.FrozenHeight = clienttypes.NewHeight(1, 1)
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, clientState)
			updateHeader = createFutureUpdateFn(clientState.GetLatestHeight().(clienttypes.Height))
		}, false, false},
		{"invalid header", func() {
			updateHeader = createFutureUpdateFn(path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height))
			updateHeader.TrustedHeight = updateHeader.TrustedHeight.Increment().(clienttypes.Height)
		}, false, false},
	}

	for _, tc := range cases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupClients(path)

			tc.malleate()

			var clientState exported.ClientState
			if tc.expPass {
				clientState = path.EndpointA.GetClientState()
			}

			err := s.chainA.App.GetIBCKeeper().ClientKeeper.UpdateClient(s.chainA.GetContext(), path.EndpointA.ClientID, updateHeader)

			if tc.expPass {
				s.Require().NoError(err, err)

				newClientState := path.EndpointA.GetClientState()

				if tc.expFreeze {
					s.Require().True(!newClientState.(*ibctm.ClientState).FrozenHeight.IsZero(), "client did not freeze after conflicting header was submitted to UpdateClient")
				} else {
					expConsensusState := &ibctm.ConsensusState{
						Timestamp:          updateHeader.GetTime(),
						Root:               commitmenttypes.NewMerkleRoot(updateHeader.Header.GetAppHash()),
						NextValidatorsHash: updateHeader.Header.NextValidatorsHash,
					}

					consensusState, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID, updateHeader.GetHeight())
					s.Require().True(found)

					// Determine if clientState should be updated or not
					if updateHeader.GetHeight().GT(clientState.GetLatestHeight()) {
						// Header Height is greater than clientState latest Height, clientState should be updated with header.GetHeight()
						s.Require().Equal(updateHeader.GetHeight(), newClientState.GetLatestHeight(), "clientstate height did not update")
					} else {
						// Update will add past consensus state, clientState should not be updated at all
						s.Require().Equal(clientState.GetLatestHeight(), newClientState.GetLatestHeight(), "client state height updated for past header")
					}

					s.Require().NoError(err)
					s.Require().Equal(expConsensusState, consensusState, "consensus state should have been updated on case %s", tc.name)
				}
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpgradeClient() {
	var (
		path                                        *ibctesting.Path
		upgradedClient                              exported.ClientState
		upgradedConsState                           exported.ConsensusState
		lastHeight                                  exported.Height
		proofUpgradedClient, proofUpgradedConsState []byte
		upgradedClientBz, upgradedConsStateBz       []byte
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			name: "successful upgrade",
			setup: func() {
				// last Height is at next block
				lastHeight = clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
				s.Require().NoError(err)

				// commit upgrade store changes and update clients
				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
			},
			expPass: true,
		},
		{
			name: "client state not found",
			setup: func() {
				// last Height is at next block
				lastHeight = clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
				s.Require().NoError(err)

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				path.EndpointA.ClientID = "wrongclientid"
			},
			expPass: false,
		},
		{
			name: "client state is not active",
			setup: func() {
				// client is frozen

				// last Height is at next block
				lastHeight = clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
				s.Require().NoError(err)

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)

				proofUpgradedClient, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
				proofUpgradedConsState, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

				// set frozen client in store
				tmClient, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClient.FrozenHeight = clienttypes.NewHeight(1, 1)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, tmClient)
			},
			expPass: false,
		},
		{
			name: "tendermint client VerifyUpgrade fails",
			setup: func() {
				// last Height is at next block
				lastHeight = clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
				s.Require().NoError(err)

				// change upgradedClient client-specified parameters
				tmClient := upgradedClient.(*ibctm.ClientState)
				tmClient.ChainId = "wrongchainID"
				upgradedClient = tmClient

				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
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
		path = ibctesting.NewPath(s.chainA, s.chainB)
		s.coordinator.SetupClients(path)

		clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
		revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

		newChainID, err := clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
		s.Require().NoError(err)

		upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
		upgradedClient = upgradedClient.ZeroCustomFields()
		upgradedClientBz, err = clienttypes.MarshalClientState(s.chainA.App.AppCodec(), upgradedClient)
		s.Require().NoError(err)

		upgradedConsState = &ibctm.ConsensusState{
			NextValidatorsHash: []byte("nextValsHash"),
		}
		upgradedConsStateBz, err = clienttypes.MarshalConsensusState(s.chainA.App.AppCodec(), upgradedConsState)
		s.Require().NoError(err)

		tc.setup()

		err = s.chainA.App.GetIBCKeeper().ClientKeeper.UpgradeClient(s.chainA.GetContext(), path.EndpointA.ClientID, upgradedClient, upgradedConsState, proofUpgradedClient, proofUpgradedConsState)

		if tc.expPass {
			s.Require().NoError(err, "verify upgrade failed on valid case: %s", tc.name)
		} else {
			s.Require().Error(err, "verify upgrade passed on invalid case: %s", tc.name)
		}
	}
}

func (s *KeeperTestSuite) TestUpdateClientEventEmission() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(path)

	header, err := s.chainA.ConstructUpdateTMClientHeader(s.chainB, path.EndpointA.ClientID)
	s.Require().NoError(err)

	msg, err := clienttypes.NewMsgUpdateClient(
		path.EndpointA.ClientID, header,
		s.chainA.SenderAccount.GetAddress().String(),
	)
	s.Require().NoError(err)

	result, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err)
	// first event type is "message", followed by 3 "tx" events in ante
	updateEvent := result.Events[4]
	s.Require().Equal(clienttypes.EventTypeUpdateClient, updateEvent.Type)

	// use a boolean to ensure the update event contains the header
	contains := false
	for _, attr := range updateEvent.Attributes {
		if attr.Key == clienttypes.AttributeKeyHeader {
			contains = true

			bz, err := hex.DecodeString(attr.Value)
			s.Require().NoError(err)

			emittedHeader, err := clienttypes.UnmarshalClientMessage(s.chainA.App.AppCodec(), bz)
			s.Require().NoError(err)
			s.Require().Equal(header, emittedHeader)
		}
	}
	s.Require().True(contains)
}
