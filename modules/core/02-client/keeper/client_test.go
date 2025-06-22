package keeper_test

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestCreateClient() {
	var (
		clientState    []byte
		consensusState []byte
	)

	testCases := []struct {
		msg        string
		malleate   func()
		clientType string
		expErr     error
	}{
		{
			"success: 07-tendermint client type supported",
			func() {
				tmClientState := ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				clientState = s.chainA.App.AppCodec().MustMarshal(tmClientState)
				consensusState = s.chainA.App.AppCodec().MustMarshal(s.consensusState)
			},
			exported.Tendermint,
			nil,
		},
		{
			"failure: 07-tendermint client status is not active",
			func() {
				tmClientState := ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				tmClientState.FrozenHeight = ibctm.FrozenHeight
				clientState = s.chainA.App.AppCodec().MustMarshal(tmClientState)
				consensusState = s.chainA.App.AppCodec().MustMarshal(s.consensusState)
			},
			exported.Tendermint,
			errorsmod.Wrapf(clienttypes.ErrClientNotActive, "cannot create client (07-tendermint-0) with status Frozen"),
		},
		{
			"success: 06-solomachine client type supported",
			func() {
				smClientState := solomachine.NewClientState(1, &solomachine.ConsensusState{PublicKey: s.solomachine.ConsensusState().PublicKey, Diversifier: s.solomachine.Diversifier, Timestamp: s.solomachine.Time})
				smConsensusState := &solomachine.ConsensusState{PublicKey: s.solomachine.ConsensusState().PublicKey, Diversifier: s.solomachine.Diversifier, Timestamp: s.solomachine.Time}
				clientState = s.chainA.App.AppCodec().MustMarshal(smClientState)
				consensusState = s.chainA.App.AppCodec().MustMarshal(smConsensusState)
			},
			exported.Solomachine,
			nil,
		},
		{
			"failure: 09-localhost client type not supported",
			func() {},
			exported.Localhost,
			errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "cannot create client of type: 09-localhost"),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset
			clientState, consensusState = []byte{}, []byte{}

			tc.malleate()

			clientID, err := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.CreateClient(s.chainA.GetContext(), tc.clientType, clientState, consensusState)

			// assert correct behaviour based on expected error
			clientState, found := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetClientState(s.chainA.GetContext(), clientID)
			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotEmpty(clientID)
				s.Require().True(found)
				s.Require().NotEmpty(clientState)
			} else {
				s.Require().Error(err)
				s.Require().Empty(clientID)
				s.Require().False(found)
				s.Require().Empty(clientState)
				s.Require().ErrorIs(err, tc.expErr)
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
		header, err := path.EndpointB.Chain.IBCClientHeader(path.EndpointB.Chain.LatestCommittedHeader, trustedHeight)
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
		expErr    error
		expFreeze bool
	}{
		{"valid update", func() {
			trustedHeight := path.EndpointA.GetClientLatestHeight()

			// store intermediate consensus state to check that trustedHeight does not need to be highest consensus state before header height
			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			updateHeader = createFutureUpdateFn(trustedHeight.(clienttypes.Height))
		}, nil, false},
		{"valid past update", func() {
			trustedHeight := path.EndpointA.GetClientLatestHeight()

			currHeight := s.chainB.ProposedHeader.Height
			fillHeight := clienttypes.NewHeight(trustedHeight.GetRevisionNumber(), uint64(currHeight))

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
			updateHeader = createPastUpdateFn(fillHeight, trustedHeight.(clienttypes.Height))
		}, nil, false},
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
		}, nil, false},
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
			// store next consensus state to check that trustedHeight does not need to be highest consensus state before header height
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
		}, nil, true},
		{"misbehaviour detection: monotonic time violation", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientID := path.EndpointA.ClientID
			trustedHeight := clientState.LatestHeight

			// store intermediate consensus state at a time greater than updateHeader time
			// this will break time monotonicity
			incrementedClientHeight, ok := clientState.LatestHeight.Increment().(clienttypes.Height)
			s.Require().True(ok)
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
		}, nil, true},
		{"client state not found", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			updateHeader = createFutureUpdateFn(clientState.LatestHeight)

			path.EndpointA.ClientID = ibctesting.InvalidID
		}, errorsmod.Wrapf(host.ErrInvalidID, "invalid client identifier IDisInvalid is not in format: `{client-type}-{N}`"), false},
		{"consensus state not found", func() {
			clientState := path.EndpointA.GetClientState()
			tmClient, ok := clientState.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClient.LatestHeight, ok = tmClient.LatestHeight.Increment().(clienttypes.Height)
			s.Require().True(ok)

			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, clientState)
			updateHeader = createFutureUpdateFn(tmClient.LatestHeight)
		}, errorsmod.Wrapf(clienttypes.ErrClientNotActive, "cannot update client (07-tendermint-0) with status Expired"), false},
		{"client is not active", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(1, 1)
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, clientState)
			updateHeader = createFutureUpdateFn(clientState.LatestHeight)
		}, errorsmod.Wrapf(clienttypes.ErrClientNotActive, "cannot update client (07-tendermint-0) with status Frozen"), false},
		{"invalid header", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			updateHeader = createFutureUpdateFn(clientState.LatestHeight)
			updateHeader.TrustedHeight, ok = updateHeader.TrustedHeight.Increment().(clienttypes.Height)
			s.Require().True(ok)
		}, errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header at TrustedHeight: 1-3"), false},
	}

	for _, tc := range cases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest()
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			tc.malleate()

			var clientState *ibctm.ClientState
			var ok bool
			if tc.expErr == nil {
				clientState, ok = path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
			}

			err := s.chainA.App.GetIBCKeeper().ClientKeeper.UpdateClient(s.chainA.GetContext(), path.EndpointA.ClientID, updateHeader)

			if tc.expErr == nil {
				s.Require().NoError(err, err)

				newClientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)

				if tc.expFreeze {
					s.Require().True(!newClientState.FrozenHeight.IsZero(), "client did not freeze after conflicting header was submitted to UpdateClient")
				} else {
					expConsensusState := &ibctm.ConsensusState{
						Timestamp:          updateHeader.GetTime(),
						Root:               commitmenttypes.NewMerkleRoot(updateHeader.Header.GetAppHash()),
						NextValidatorsHash: updateHeader.Header.NextValidatorsHash,
					}

					consensusState, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID, updateHeader.GetHeight())
					s.Require().True(found)

					// Determine if clientState should be updated or not
					if updateHeader.GetHeight().GT(clientState.LatestHeight) {
						// Header Height is greater than clientState latest Height, clientState should be updated with header.GetHeight()
						s.Require().Equal(updateHeader.GetHeight(), newClientState.LatestHeight, "clientstate height did not update")
					} else {
						// Update will add past consensus state, clientState should not be updated at all
						s.Require().Equal(clientState.LatestHeight, newClientState.LatestHeight, "client state height updated for past header")
					}

					s.Require().NoError(err)
					s.Require().Equal(expConsensusState, consensusState, "consensus state should have been updated on case %s", tc.name)
				}
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpgradeClient() {
	var (
		path                                             *ibctesting.Path
		upgradedClient                                   *ibctm.ClientState
		upgradedConsState                                exported.ConsensusState
		upgradeHeight                                    exported.Height
		upgradedClientAny, upgradedConsStateAny          *codectypes.Any
		upgradedClientProof, upgradedConsensusStateProof []byte
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
				upgradeHeight = clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedClientAny))
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedConsStateAny))
				s.Require().NoError(err)

				// commit upgrade store changes and update clients
				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: nil,
		},
		{
			name: "client state not found",
			setup: func() {
				// upgrade height is at next block
				upgradeHeight = clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedClientAny))
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedConsStateAny))
				s.Require().NoError(err)

				// commit upgrade store changes and update clients

				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				path.EndpointA.ClientID = "wrongclientid"
			},
			expErr: errorsmod.Wrap(host.ErrInvalidID, "unable to parse client identifier wrongclientid: invalid client identifier wrongclientid is not in format: `{client-type}-{N}"),
		},
		{
			name: "client state is not active",
			setup: func() {
				// client is frozen

				// upgrade height is at next block
				upgradeHeight = clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedClientAny))
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedConsStateAny))
				s.Require().NoError(err)

				// commit upgrade store changes and update clients
				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())

				// set frozen client in store
				tmClient, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClient.FrozenHeight = clienttypes.NewHeight(1, 1)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, tmClient)
			},
			expErr: errorsmod.Wrap(clienttypes.ErrClientNotActive, "cannot upgrade client (07-tendermint-2) with status Frozen"),
		},
		{
			name: "light client module VerifyUpgradeAndUpdateState fails",
			setup: func() {
				// upgrade height is at next block
				upgradeHeight = clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				err := s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedClientAny))
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedConsStateAny))
				s.Require().NoError(err)

				// change upgradedClient client-specified parameters
				upgradedClient.ChainId = "wrongchainID"
				upgradedClientAny, err = codectypes.NewAnyWithValue(upgradedClient)
				s.Require().NoError(err)

				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), path.EndpointA.ClientID)
				s.Require().True(found)
				tmCs, ok := cs.(*ibctm.ClientState)
				s.Require().True(ok)

				upgradedClientProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), tmCs.LatestHeight.GetRevisionHeight())
			},
			expErr: errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed to verify membership proof at index 0: provided value doesn't match proof"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

			newChainID, err := clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
			s.Require().NoError(err)

			upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
			upgradedClient = upgradedClient.ZeroCustomFields()

			upgradedClientAny, err = codectypes.NewAnyWithValue(upgradedClient)
			s.Require().NoError(err)

			upgradedConsState = &ibctm.ConsensusState{NextValidatorsHash: []byte("nextValsHash")}

			upgradedConsStateAny, err = codectypes.NewAnyWithValue(upgradedConsState)
			s.Require().NoError(err)

			tc.setup()

			err = s.chainA.App.GetIBCKeeper().ClientKeeper.UpgradeClient(s.chainA.GetContext(), path.EndpointA.ClientID, upgradedClientAny.Value, upgradedConsStateAny.Value, upgradedClientProof, upgradedConsensusStateProof)

			if tc.expErr == nil {
				s.Require().NoError(err, "verify upgrade failed on valid case: %s", tc.name)
			} else {
				s.Require().Error(err, "verify upgrade passed on invalid case: %s", tc.name)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpdateClientEventEmission() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	tmClientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
	s.Require().True(ok)
	trustedHeight := tmClientState.LatestHeight
	header, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
	s.Require().NoError(err)

	msg, err := clienttypes.NewMsgUpdateClient(
		path.EndpointA.ClientID, header,
		s.chainA.SenderAccount.GetAddress().String(),
	)
	s.Require().NoError(err)

	result, err := s.chainA.SendMsgs(msg)

	// check that update client event was emitted
	s.Require().NoError(err)
	var event abci.Event
	for _, e := range result.Events {
		if e.Type == clienttypes.EventTypeUpdateClient {
			event = e
		}
	}
	s.Require().NotNil(event)
}

func (s *KeeperTestSuite) TestRecoverClient() {
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
				s.Require().True(ok)
				consState, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(s.chainA.GetContext(), substitute, tmClientState.LatestHeight)
				s.Require().True(found)
				newRevisionNumber := tmClientState.LatestHeight.GetRevisionNumber() + 1

				tmClientState.LatestHeight = clienttypes.NewHeight(newRevisionNumber, tmClientState.LatestHeight.GetRevisionHeight())

				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), substitute, tmClientState.LatestHeight, consState)
				clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), substitute)
				ibctm.SetProcessedTime(clientStore, tmClientState.LatestHeight, 100)
				ibctm.SetProcessedHeight(clientStore, tmClientState.LatestHeight, clienttypes.NewHeight(0, 1))
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitute, tmClientState)
			},
			nil,
		},
		{
			"subject client does not exist",
			func() {
				subject = ibctesting.InvalidID
			},
			clienttypes.ErrRouteNotFound,
		},
		{
			"subject is Active",
			func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				// Set FrozenHeight to zero to ensure client is reported as Active
				tmClientState.FrozenHeight = clienttypes.ZeroHeight()
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subject, tmClientState)
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
				s.Require().True(ok)
				tmClientState.LatestHeight = substituteClientState.(*ibctm.ClientState).LatestHeight
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subject, tmClientState)
			},
			clienttypes.ErrInvalidHeight,
		},
		{
			"subject height is greater than substitute height",
			func() {
				tmClientState, ok := subjectClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClientState.LatestHeight, ok = substituteClientState.(*ibctm.ClientState).LatestHeight.Increment().(clienttypes.Height)
				s.Require().True(ok)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subject, tmClientState)
			},
			clienttypes.ErrInvalidHeight,
		},
		{
			"substitute is frozen",
			func() {
				tmClientState, ok := substituteClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClientState.FrozenHeight = clienttypes.NewHeight(0, 1)
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitute, tmClientState)
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"light client module RecoverClient fails, substitute client trust level doesn't match subject client trust level",
			func() {
				tmClientState, ok := substituteClientState.(*ibctm.ClientState)
				s.Require().True(ok)
				tmClientState.UnbondingPeriod += time.Minute
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), substitute, tmClientState)
			},
			clienttypes.ErrInvalidSubstitute,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest() // reset

			subjectPath := ibctesting.NewPath(s.chainA, s.chainB)
			subjectPath.SetupClients()
			subject = subjectPath.EndpointA.ClientID
			subjectClientState = s.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(s.chainA, s.chainB)
			substitutePath.SetupClients()
			substitute = substitutePath.EndpointA.ClientID

			// update substitute twice
			err := substitutePath.EndpointA.UpdateClient()
			s.Require().NoError(err)
			err = substitutePath.EndpointA.UpdateClient()
			s.Require().NoError(err)
			substituteClientState = s.chainA.GetClientState(substitute)

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), subject, tmClientState)

			tc.malleate()

			ctx := s.chainA.GetContext()
			err = s.chainA.App.GetIBCKeeper().ClientKeeper.RecoverClient(ctx, subject, substitute)

			if tc.expErr == nil {
				s.Require().NoError(err)

				expectedEvents := sdk.Events{
					sdk.NewEvent(
						clienttypes.EventTypeRecoverClient,
						sdk.NewAttribute(clienttypes.AttributeKeySubjectClientID, subjectPath.EndpointA.ClientID),
						sdk.NewAttribute(clienttypes.AttributeKeyClientType, subjectPath.EndpointA.GetClientState().ClientType()),
					),
				}.ToABCIEvents()

				expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
				ibctesting.AssertEvents(&s.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())

				// Assert that client status is now Active
				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), subjectPath.EndpointA.ClientID)
				s.Require().NoError(err)
				s.Require().Equal(lightClientModule.Status(s.chainA.GetContext(), subjectPath.EndpointA.ClientID), exported.Active)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
