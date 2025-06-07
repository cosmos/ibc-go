package keeper_test

import (
	"time"

	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
		expErr   error
	}{
		{"success", func() {
		}, nil},
		{"success with empty counterparty identifier", func() {
			emptyConnBID = true
		}, nil},
		{"success with non empty version", func() {
			version = types.GetCompatibleVersions()[0]
		}, nil},
		{"success with non zero delayPeriod", func() {
			delayPeriod = uint64(time.Hour.Nanoseconds())
		}, nil},

		{"invalid version", func() {
			version = &types.Version{}
		}, errorsmod.Wrap(types.ErrInvalidVersion, "version is not supported")},
		{"couldn't add connection to client", func() {
			// set path.EndpointA.ClientID to invalid client identifier
			path.EndpointA.ClientID = "clientidentifier"
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (clientidentifier) status is Unauthorized")},
		{
			msg:    "unauthorized client",
			expErr: errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Unauthorized"),
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
		s.Run(tc.msg, func() {
			s.SetupTest()        // reset
			emptyConnBID = false // must be explicitly changed
			version = nil        // must be explicitly changed
			expErrorMsgSubstring = ""
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			tc.malleate()

			if emptyConnBID {
				path.EndpointB.ConnectionID = ""
			}
			counterparty := types.NewCounterparty(path.EndpointB.ClientID, path.EndpointB.ConnectionID, s.chainB.GetPrefix())

			connectionID, err := s.chainA.App.GetIBCKeeper().ConnectionKeeper.ConnOpenInit(s.chainA.GetContext(), path.EndpointA.ClientID, counterparty, version, delayPeriod)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(types.FormatConnectionIdentifier(0), connectionID)
			} else {
				s.Require().Error(err)
				s.Contains(err.Error(), expErrorMsgSubstring)
				s.Require().Equal("", connectionID)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestConnOpenTry - chainB calls ConnOpenTry to verify the state of
// connection on chainA is INIT
func (s *KeeperTestSuite) TestConnOpenTry() {
	var (
		path        *ibctesting.Path
		delayPeriod uint64
		versions    []*types.Version
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{"success", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)
		}, nil},
		{"success with delay period", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			delayPeriod = uint64(time.Hour.Nanoseconds())

			// set delay period on counterparty to non-zero value
			path.EndpointA.UpdateConnection(func(connection *types.ConnectionEnd) { connection.DelayPeriod = delayPeriod })

			// commit in order for proof to return correct value
			s.coordinator.CommitBlock(s.chainA)
			err = path.EndpointB.UpdateClient()
			s.Require().NoError(err)
		}, nil},
		{"counterparty versions is empty", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			versions = nil
		}, errorsmod.Wrap(types.ErrVersionNegotiationFailed, "failed to find a matching counterparty version ([]) from the supported version list ([identifier:\"1\" features:\"ORDER_ORDERED\" features:\"ORDER_UNORDERED\" ])")},
		{"counterparty versions don't have a match", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			version := types.NewVersion("0.0", nil)
			versions = []*types.Version{version}
		}, errorsmod.Wrap(types.ErrVersionNegotiationFailed, "failed to find a matching counterparty version ([identifier:\"0.0\" ]) from the supported version list ([identifier:\"1\" features:\"ORDER_ORDERED\" features:\"ORDER_UNORDERED\" ])")},
		{"connection state verification failed", func() {
			// chainA connection not created
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed connection state verification for client (07-tendermint-0): commitment proof must be existence proof. got: int at index &{1374402732384}")},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest()                            // reset
			versions = types.GetCompatibleVersions() // may be changed in malleate
			delayPeriod = 0                          // may be changed in malleate
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			tc.malleate()

			counterparty := types.NewCounterparty(path.EndpointA.ClientID, path.EndpointA.ConnectionID, s.chainA.GetPrefix())

			// ensure client is up to date to receive proof
			err := path.EndpointB.UpdateClient()
			s.Require().NoError(err)

			connectionKey := host.ConnectionKey(path.EndpointA.ConnectionID)
			initProof, proofHeight := s.chainA.QueryProof(connectionKey)

			connectionID, err := s.chainB.App.GetIBCKeeper().ConnectionKeeper.ConnOpenTry(
				s.chainB.GetContext(), counterparty, delayPeriod, path.EndpointB.ClientID,
				versions, initProof, proofHeight,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(types.FormatConnectionIdentifier(0), connectionID)
			} else {
				s.Require().Error(err)
				s.Require().Equal("", connectionID)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestConnOpenAck - Chain A (ID #1) calls TestConnOpenAck to acknowledge (ACK state)
// the initialization (TRYINIT) of the connection on  Chain B (ID #2).
func (s *KeeperTestSuite) TestConnOpenAck() {
	var (
		path    *ibctesting.Path
		version *types.Version
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{"success", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)
		}, nil},
		{"connection not found", func() {
			// connections are never created
		}, errorsmod.Wrap(types.ErrConnectionNotFound, "")},
		{"invalid counterparty connection ID", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			// modify connB to set counterparty connection identifier to wrong identifier
			path.EndpointA.UpdateConnection(func(c *types.ConnectionEnd) { c.Counterparty.ConnectionId = ibctesting.InvalidID })
			path.EndpointB.ConnectionID = ibctesting.InvalidID

			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			err = path.EndpointB.UpdateClient()
			s.Require().NoError(err)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed connection state verification for client (07-tendermint-0): commitment proof must be existence proof. got: int at index &{1374412614704}")},
		{"connection state is not INIT", func() {
			// connection state is already OPEN on chainA
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ConnOpenAck()
			s.Require().NoError(err)
		}, errorsmod.Wrap(types.ErrInvalidConnectionState, "connection state is not INIT (got STATE_OPEN)")},
		{"connection is in INIT but the proposed version is invalid", func() {
			// chainA is in INIT, chainB is in TRYOPEN
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			version = types.NewVersion("2.0", nil)
		}, errorsmod.Wrap(types.ErrInvalidConnectionState, "the counterparty selected version identifier:\"2.0\"  is not supported by versions selected on INIT")},
		{"incompatible IBC versions", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			// set version to a non-compatible version
			version = types.NewVersion("2.0", nil)
		}, errorsmod.Wrap(types.ErrInvalidConnectionState, "the counterparty selected version identifier:\"2.0\"  is not supported by versions selected on INIT")},
		{"empty version", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			version = &types.Version{}
		}, errorsmod.Wrap(types.ErrInvalidConnectionState, "the counterparty selected version  is not supported by versions selected on INIT")},
		{"feature set verification failed - unsupported feature", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			version = types.NewVersion(types.DefaultIBCVersionIdentifier, []string{"ORDER_ORDERED", "ORDER_UNORDERED", "ORDER_DAG"})
		}, errorsmod.Wrap(types.ErrInvalidConnectionState, "the counterparty selected version identifier:\"1\" features:\"ORDER_ORDERED\" features:\"ORDER_UNORDERED\" features:\"ORDER_DAG\"  is not supported by versions selected on INIT")},
		{"connection state verification failed", func() {
			// chainB connection is not in INIT
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed connection state verification for client (07-tendermint-0): commitment proof must be existence proof. got: int at index &{1374414228888}")},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest()                              // reset
			version = types.GetCompatibleVersions()[0] // must be explicitly changed in malleate
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			tc.malleate()

			// ensure client is up to date to receive proof
			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			connectionKey := host.ConnectionKey(path.EndpointB.ConnectionID)
			tryProof, proofHeight := s.chainB.QueryProof(connectionKey)

			err = s.chainA.App.GetIBCKeeper().ConnectionKeeper.ConnOpenAck(
				s.chainA.GetContext(), path.EndpointA.ConnectionID, version,
				path.EndpointB.ConnectionID, tryProof, proofHeight,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
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
		expErr   error
	}{
		{"success", func() {
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ConnOpenAck()
			s.Require().NoError(err)
		}, nil},
		{"connection not found", func() {
			// connections are never created
		}, errorsmod.Wrap(types.ErrConnectionNotFound, "")},
		{"chain B's connection state is not TRYOPEN", func() {
			// connections are OPEN
			path.CreateConnections()
		}, errorsmod.Wrap(types.ErrInvalidConnectionState, "connection state is not TRYOPEN (got STATE_OPEN)")},
		{"connection state verification failed", func() {
			// chainA is in INIT
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			err = path.EndpointB.ConnOpenTry()
			s.Require().NoError(err)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed connection state verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			tc.malleate()

			// ensure client is up to date to receive proof
			err := path.EndpointB.UpdateClient()
			s.Require().NoError(err)

			connectionKey := host.ConnectionKey(path.EndpointA.ConnectionID)
			ackProof, proofHeight := s.chainA.QueryProof(connectionKey)

			err = s.chainB.App.GetIBCKeeper().ConnectionKeeper.ConnOpenConfirm(
				s.chainB.GetContext(), path.EndpointB.ConnectionID, ackProof, proofHeight,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
