package keeper_test

import (
	"time"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// TestConnOpenInit - chainA initializes (INIT state) a connection with
// chainB which is yet UNINITIALIZED
func (s *KeeperTestSuite) TestConnOpenInit() {
	var (
		path                 *ibctesting.Path
		version              *types.Version
		delayPeriod          uint64
		emptyConnBID         bool
		expErrorMsgSubstring string
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{"success", func() {
		}, true},
		{"success with empty counterparty identifier", func() {
			emptyConnBID = true
		}, true},
		{"success with non empty version", func() {
			version = types.ExportedVersionsToProto(types.GetCompatibleVersions())[0]
		}, true},
		{"success with non zero delayPeriod", func() {
			delayPeriod = uint64(time.Hour.Nanoseconds())
		}, true},

		{"invalid version", func() {
			version = &types.Version{}
		}, false},
		{"couldn't add connection to client", func() {
			// set path.EndpointA.ClientID to invalid client identifier
			path.EndpointA.ClientID = "clientidentifier"
		}, false},
		{
			msg:     "unauthorized client",
			expPass: false,
			malleate: func() {
				expErrorMsgSubstring = "status is Unauthorized"
				// remove client from allowed list
				params := s.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(s.chainA.GetContext())
				params.AllowedClients = []string{}
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetParams(s.chainA.GetContext(), params)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.msg, func() {
			s.SetupTest()        // reset
			emptyConnBID = false // must be explicitly changed
			version = nil        // must be explicitly changed
			expErrorMsgSubstring = ""
			path = ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupClients(path)

			tc.malleate()

			if emptyConnBID {
				path.EndpointB.ConnectionID = ""
			}
			counterparty := types.NewCounterparty(path.EndpointB.ClientID, path.EndpointB.ConnectionID, s.chainB.GetPrefix())

			connectionID, err := s.chainA.App.GetIBCKeeper().ConnectionKeeper.ConnOpenInit(s.chainA.GetContext(), path.EndpointA.ClientID, counterparty, version, delayPeriod)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(types.FormatConnectionIdentifier(0), connectionID)
			} else {
				s.Require().Error(err)
				s.Contains(err.Error(), expErrorMsgSubstring)
				s.Require().Equal("", connectionID)
			}
		})
	}
}

// TestConnOpenTry - chainB calls ConnOpenTry to verify the state of
// connection on chainA is INIT
func (s *KeeperTestSuite) TestConnOpenTry() {
	var (
		path               *ibctesting.Path
		delayPeriod        uint64
		versions           []exported.Version
		consensusHeight    exported.Height
		counterpartyClient exported.ClientState
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{"success", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)
		}, true},
		{"success with delay period", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			delayPeriod = uint64(time.Hour.Nanoseconds())

			// set delay period on counterparty to non-zero value
			conn := path.EndpointA.GetConnection()
			conn.DelayPeriod = delayPeriod
			s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID, conn)

			// commit in order for proof to return correct value
			s.coordinator.CommitBlock(s.chainA)
			err = path.EndpointB.UpdateClient()
			s.Require().NoError(err)

			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)
		}, true},
		{"invalid counterparty client", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)

			// Set an invalid client of chainA on chainB
			tmClient, ok := counterpartyClient.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClient.ChainId = "wrongchainid"

			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, tmClient)
		}, false},
		{"consensus height >= latest height", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)

			consensusHeight = clienttypes.GetSelfHeight(s.chainB.GetContext())
		}, false},
		{"self consensus state not found", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)

			consensusHeight = clienttypes.NewHeight(0, 1)
		}, false},
		{"counterparty versions is empty", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)

			versions = nil
		}, false},
		{"counterparty versions don't have a match", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)

			version := types.NewVersion("0.0", nil)
			versions = []exported.Version{version}
		}, false},
		{"connection state verification failed", func() {
			// chainA connection not created

			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)
		}, false},
		{"client state verification failed", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)

			// modify counterparty client without setting in store so it still passes validate but fails proof verification
			tmClient, ok := counterpartyClient.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClient.LatestHeight = tmClient.LatestHeight.Increment().(clienttypes.Height)
		}, false},
		{"consensus state verification failed", func() {
			// retrieve client state of chainA to pass as counterpartyClient
			counterpartyClient = s.chainA.GetClientState(path.EndpointA.ClientID)

			// give chainA wrong consensus state for chainB
			consState, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetLatestClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().True(found)

			tmConsState, ok := consState.(*ibctm.ConsensusState)
			s.Require().True(ok)

			tmConsState.Timestamp = time.Now()
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), path.EndpointA.ClientID, counterpartyClient.GetLatestHeight(), tmConsState)

			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.msg, func() {
			s.SetupTest()                              // reset
			consensusHeight = clienttypes.ZeroHeight() // may be changed in malleate
			versions = types.GetCompatibleVersions()   // may be changed in malleate
			delayPeriod = 0                            // may be changed in malleate
			path = ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupClients(path)

			tc.malleate()

			counterparty := types.NewCounterparty(path.EndpointA.ClientID, path.EndpointA.ConnectionID, s.chainA.GetPrefix())

			// ensure client is up to date to receive proof
			err := path.EndpointB.UpdateClient()
			s.Require().NoError(err)

			connectionKey := host.ConnectionKey(path.EndpointA.ConnectionID)
			proofInit, proofHeight := s.chainA.QueryProof(connectionKey)

			if consensusHeight.IsZero() {
				// retrieve consensus state height to provide proof for
				consensusHeight = counterpartyClient.GetLatestHeight()
			}
			consensusKey := host.FullConsensusStateKey(path.EndpointA.ClientID, consensusHeight)
			proofConsensus, _ := s.chainA.QueryProof(consensusKey)

			// retrieve proof of counterparty clientstate on chainA
			clientKey := host.FullClientStateKey(path.EndpointA.ClientID)
			proofClient, _ := s.chainA.QueryProof(clientKey)

			connectionID, err := s.chainB.App.GetIBCKeeper().ConnectionKeeper.ConnOpenTry(
				s.chainB.GetContext(), counterparty, delayPeriod, path.EndpointB.ClientID, counterpartyClient,
				versions, proofInit, proofClient, proofConsensus,
				proofHeight, consensusHeight,
			)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(types.FormatConnectionIdentifier(0), connectionID)
			} else {
				s.Require().Error(err)
				s.Require().Equal("", connectionID)
			}
		})
	}
}

// TestConnOpenAck - Chain A (ID #1) calls TestConnOpenAck to acknowledge (ACK state)
// the initialization (TRYINIT) of the connection on  Chain B (ID #2).
func (s *KeeperTestSuite) TestConnOpenAck() {
	var (
		path               *ibctesting.Path
		consensusHeight    exported.Height
		version            *types.Version
		counterpartyClient exported.ClientState
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{"success", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)
		}, true},
		{"invalid counterparty client", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			// Set an invalid client of chainA on chainB
			tmClient, ok := counterpartyClient.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClient.ChainId = "wrongchainid"

			s.chainB.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainB.GetContext(), path.EndpointB.ClientID, tmClient)
		}, false},
		{"consensus height >= latest height", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			consensusHeight = clienttypes.GetSelfHeight(s.chainA.GetContext())
		}, false},
		{"connection not found", func() {
			// connections are never created

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)
		}, false},
		{"invalid counterparty connection ID", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			// modify connB to set counterparty connection identifier to wrong identifier
			connection, found := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID)
			s.Require().True(found)

			connection.Counterparty.ConnectionId = "badconnectionid"

			s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID, connection)

			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			err = path.EndpointB.UpdateClient()
			s.Require().NoError(err)
		}, false},
		{"connection state is not INIT", func() {
			// connection state is already OPEN on chainA
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ConnOpenAck()
			s.Require().NoError(err)
		}, false},
		{"connection is in INIT but the proposed version is invalid", func() {
			// chainA is in INIT, chainB is in TRYOPEN
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			version = types.NewVersion("2.0", nil)
		}, false},
		{"incompatible IBC versions", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			// set version to a non-compatible version
			version = types.NewVersion("2.0", nil)
		}, false},
		{"empty version", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			version = &types.Version{}
		}, false},
		{"feature set verification failed - unsupported feature", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			version = types.NewVersion(types.DefaultIBCVersionIdentifier, []string{"ORDER_ORDERED", "ORDER_UNORDERED", "ORDER_DAG"})
		}, false},
		{"self consensus state not found", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			consensusHeight = clienttypes.NewHeight(0, 1)
		}, false},
		{"connection state verification failed", func() {
			// chainB connection is not in INIT
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)
		}, false},
		{"client state verification failed", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			// modify counterparty client without setting in store so it still passes validate but fails proof verification
			tmClient, ok := counterpartyClient.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClient.LatestHeight = tmClient.LatestHeight.Increment().(clienttypes.Height)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)
		}, false},
		{"consensus state verification failed", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// retrieve client state of chainB to pass as counterpartyClient
			counterpartyClient = s.chainB.GetClientState(path.EndpointB.ClientID)

			// give chainB wrong consensus state for chainA
			consState, found := s.chainB.App.GetIBCKeeper().ClientKeeper.GetLatestClientConsensusState(s.chainB.GetContext(), path.EndpointB.ClientID)
			s.Require().True(found)

			tmConsState, ok := consState.(*ibctm.ConsensusState)
			s.Require().True(ok)

			tmConsState.Timestamp = tmConsState.Timestamp.Add(time.Second)
			s.chainB.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainB.GetContext(), path.EndpointB.ClientID, counterpartyClient.GetLatestHeight(), tmConsState)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.msg, func() {
			s.SetupTest()                                                             // reset
			version = types.ExportedVersionsToProto(types.GetCompatibleVersions())[0] // must be explicitly changed in malleate
			consensusHeight = clienttypes.ZeroHeight()                                // must be explicitly changed in malleate
			path = ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupClients(path)

			tc.malleate()

			// ensure client is up to date to receive proof
			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			connectionKey := host.ConnectionKey(path.EndpointB.ConnectionID)
			proofTry, proofHeight := s.chainB.QueryProof(connectionKey)

			if consensusHeight.IsZero() {
				// retrieve consensus state height to provide proof for
				clientState := s.chainB.GetClientState(path.EndpointB.ClientID)
				consensusHeight = clientState.GetLatestHeight()
			}
			consensusKey := host.FullConsensusStateKey(path.EndpointB.ClientID, consensusHeight)
			proofConsensus, _ := s.chainB.QueryProof(consensusKey)

			// retrieve proof of counterparty clientstate on chainA
			clientKey := host.FullClientStateKey(path.EndpointB.ClientID)
			proofClient, _ := s.chainB.QueryProof(clientKey)

			err = s.chainA.App.GetIBCKeeper().ConnectionKeeper.ConnOpenAck(
				s.chainA.GetContext(), path.EndpointA.ConnectionID, counterpartyClient, version, path.EndpointB.ConnectionID,
				proofTry, proofClient, proofConsensus, proofHeight, consensusHeight,
			)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestConnOpenConfirm - chainB calls ConnOpenConfirm to confirm that
// chainA state is now OPEN.
func (s *KeeperTestSuite) TestConnOpenConfirm() {
	var path *ibctesting.Path
	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{"success", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ConnOpenAck()
			s.Require().NoError(err)
		}, true},
		{"connection not found", func() {
			// connections are never created
		}, false},
		{"chain B's connection state is not TRYOPEN", func() {
			// connections are OPEN
			s.coordinator.CreateConnections(path)
		}, false},
		{"connection state verification failed", func() {
			// chainA is in INIT
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)
		}, false},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.msg, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupClients(path)

			tc.malleate()

			// ensure client is up to date to receive proof
			err := path.EndpointB.UpdateClient()
			s.Require().NoError(err)

			connectionKey := host.ConnectionKey(path.EndpointA.ConnectionID)
			proofAck, proofHeight := s.chainA.QueryProof(connectionKey)

			err = s.chainB.App.GetIBCKeeper().ConnectionKeeper.ConnOpenConfirm(
				s.chainB.GetContext(), path.EndpointB.ConnectionID, proofAck, proofHeight,
			)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
