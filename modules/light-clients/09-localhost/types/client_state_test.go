package types_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v5/modules/core/03-connection/types"

	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v5/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
	"github.com/cosmos/ibc-go/v5/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v5/modules/light-clients/07-tendermint/types"
	"github.com/cosmos/ibc-go/v5/modules/light-clients/09-localhost/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
	ibcmock "github.com/cosmos/ibc-go/v5/testing/mock"
)

const (
	testConnectionID = "connectionid"
	testPortID       = "testportid"
	testChannelID    = "testchannelid"
	testSequence     = 1
)

func (suite *LocalhostTestSuite) TestStatus() {
	ctx := suite.chain.GetContext()
	clientState := types.NewClientState("chainID", clienttypes.NewHeight(3, 10))

	// localhost should always return active
	status := clientState.Status(ctx, nil, nil)
	suite.Require().Equal(exported.Active, status)
}

func (suite *LocalhostTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState *types.ClientState
		expPass     bool
	}{
		{
			name:        "valid client",
			clientState: types.NewClientState("chainID", clienttypes.NewHeight(3, 10)),
			expPass:     true,
		},
		{
			name:        "invalid chain id",
			clientState: types.NewClientState(" ", clienttypes.NewHeight(3, 10)),
			expPass:     false,
		},
		{
			name:        "invalid height",
			clientState: types.NewClientState("chainID", clienttypes.ZeroHeight()),
			expPass:     false,
		},
	}

	for _, tc := range testCases {
		err := tc.clientState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}

func (suite *LocalhostTestSuite) TestInitialize() {
	testCases := []struct {
		name      string
		consState exported.ConsensusState
		expPass   bool
	}{
		{
			"valid initialization",
			nil,
			true,
		},
		{
			"invalid consenus state",
			&ibctmtypes.ConsensusState{},
			false,
		},
	}

	clientState := types.NewClientState("chainID", clienttypes.NewHeight(3, 10))

	for _, tc := range testCases {
		err := clientState.Initialize(suite.chain.GetContext(), suite.chain.Codec, nil, tc.consState)

		if tc.expPass {
			suite.Require().NoError(err, "valid testcase: %s failed", tc.name)
		} else {
			suite.Require().Error(err, "invalid testcase: %s passed", tc.name)
		}
	}
}

func (suite *LocalhostTestSuite) TestVerifyClientState() {
	clientState := types.NewClientState("chainID", clienttypes.Height{})
	invalidClient := types.NewClientState("chainID", clienttypes.NewHeight(0, 12))
	testCases := []struct {
		name         string
		clientState  *types.ClientState
		malleate     func(codec.BinaryCodec, sdk.KVStore)
		counterparty *types.ClientState
		expPass      bool
	}{
		{
			name:        "proof verification success",
			clientState: clientState,
			malleate: func(cdc codec.BinaryCodec, store sdk.KVStore) {
				bz := clienttypes.MustMarshalClientState(cdc, clientState)
				store.Set(host.ClientStateKey(), bz)
			},
			counterparty: clientState,
			expPass:      true,
		},
		{
			name:        "proof verification failed: invalid client",
			clientState: clientState,
			malleate: func(cdc codec.BinaryCodec, store sdk.KVStore) {
				bz := clienttypes.MustMarshalClientState(cdc, clientState)
				store.Set(host.ClientStateKey(), bz)
			},
			counterparty: invalidClient,
			expPass:      false,
		},
		{
			name:         "proof verification failed: client not stored",
			clientState:  clientState,
			malleate:     func(cdc codec.BinaryCodec, store sdk.KVStore) {},
			counterparty: clientState,
			expPass:      false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			cdc := suite.chain.Codec
			store := suite.chain.GetContext().KVStore(suite.chain.App.GetKey(host.StoreKey))
			tc.malleate(cdc, store)

			err := tc.clientState.VerifyClientState(
				store, cdc, clienttypes.NewHeight(0, 10), nil, "", []byte{}, tc.counterparty,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestVerifyClientConsensusState() {
	clientState := types.NewClientState("chainID", clienttypes.Height{})
	err := clientState.VerifyClientConsensusState(
		nil, nil, nil, "", nil, nil, nil, nil,
	)
	suite.Require().NoError(err)
}

func (suite *LocalhostTestSuite) TestCheckHeaderAndUpdateState() {
	ctx := suite.chain.GetContext()
	clientState := types.NewClientState("chainID", clienttypes.Height{})
	cs, _, err := clientState.CheckHeaderAndUpdateState(ctx, nil, nil, nil)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(0), cs.GetLatestHeight().GetRevisionNumber())
	suite.Require().Equal(ctx.BlockHeight(), int64(cs.GetLatestHeight().GetRevisionHeight()))
	suite.Require().Equal(ctx.BlockHeader().ChainID, clientState.ChainId)
}

func (suite *LocalhostTestSuite) TestMisbehaviourAndUpdateState() {
	ctx := suite.chain.GetContext()
	clientState := types.NewClientState("chainID", clienttypes.Height{})
	cs, err := clientState.CheckMisbehaviourAndUpdateState(ctx, nil, nil, nil)
	suite.Require().Error(err)
	suite.Require().Nil(cs)
}

func (suite *LocalhostTestSuite) TestProposedHeaderAndUpdateState() {
	ctx := suite.chain.GetContext()
	clientState := types.NewClientState("chainID", clienttypes.Height{})
	cs, err := clientState.CheckSubstituteAndUpdateState(ctx, nil, nil, nil, nil)
	suite.Require().Error(err)
	suite.Require().Nil(cs)
}

func (suite *LocalhostTestSuite) TestVerifyConnectionState() {
	var (
		path   *ibctesting.Path
		connID string
		conn   connectiontypes.ConnectionEnd
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "proof verification success",
			malleate: func() {
				conn = path.EndpointB.GetConnection()
				connID = path.EndpointB.ConnectionID
			},
			expPass: true,
		},
		{
			name: "proof verification failed: connection not stored",
			malleate: func() {
				connID = testConnectionID
			},
			expPass: false,
		},
		{
			name: "proof verification failed: unmarshal failed",
			malleate: func() {
				connID = testConnectionID
				store := suite.chain.GetContext().KVStore(suite.chain.App.GetKey(host.StoreKey))
				store.Set(host.ConnectionKey(connID), []byte("connection"))
			},
			expPass: false,
		},
		{
			name: "proof verification failed: different connection stored",
			malleate: func() {
				counterparty := connectiontypes.NewCounterparty(path.EndpointB.ClientID, path.EndpointB.ConnectionID, commitmenttypes.NewMerklePrefix([]byte("ibc")))
				conn = connectiontypes.NewConnectionEnd(connectiontypes.OPEN, path.EndpointA.ClientID, counterparty, []*connectiontypes.Version{connectiontypes.NewVersion("2", nil)}, 0)
				connID = conn.Counterparty.ConnectionId
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewLocalPath(suite.chain)
			suite.coordinator.Setup(path)
			tc.malleate()

			clientStateI := suite.chain.GetClientState(path.EndpointA.ClientID)
			clientState, ok := clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			store := suite.chain.GetContext().KVStore(suite.chain.App.GetKey(host.StoreKey))
			err := clientState.VerifyConnectionState(
				store, suite.chain.Codec, clienttypes.Height{}, nil, []byte{}, connID, conn,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestVerifyChannelState() {

	var (
		path      *ibctesting.Path
		channelID string
		portID    string
		channel   channeltypes.Channel
	)
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "proof verification success",
			malleate: func() {
				channelB := path.EndpointB.GetChannel()
				channelID = channelB.Counterparty.ChannelId
				portID = channelB.Counterparty.PortId
				channel = path.EndpointA.GetChannel()
			},
			expPass: true,
		},
		{
			name: "proof verification failed: channel not stored",
			malleate: func() {
				channelID = testChannelID
				portID = testPortID
			},
			expPass: false,
		},
		{
			name: "proof verification failed: unmarshal failed",
			malleate: func() {
				channelID = testChannelID
				portID = testPortID
				store := suite.chain.GetContext().KVStore(suite.chain.App.GetKey(host.StoreKey))
				store.Set(host.ChannelKey(testPortID, testChannelID), []byte("channel"))
			},
			expPass: false,
		},
		{
			name: "proof verification failed: different channel stored",
			malleate: func() {
				activeChannel := path.EndpointB.GetChannel()
				channelID = activeChannel.Counterparty.ChannelId
				portID = activeChannel.Counterparty.PortId

				counterparty := channeltypes.NewCounterparty(testPortID, testChannelID)
				channel = channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, counterparty, []string{testConnectionID}, "1.0.0")
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewLocalPath(suite.chain)
			suite.coordinator.Setup(path)
			tc.malleate()

			clientStateI := suite.chain.GetClientState(path.EndpointA.ClientID)
			clientState, ok := clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			store := suite.chain.GetContext().KVStore(suite.chain.App.GetKey(host.StoreKey))
			err := clientState.VerifyChannelState(
				store, suite.chain.Codec, clienttypes.Height{}, nil, []byte{}, portID, channelID, channel,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestVerifyPacketCommitment() {
	var (
		packet     channeltypes.Packet
		portID     string
		channelID  string
		sequence   uint64
		commitment []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "proof verification success",
			malleate: func() {
				portID = packet.GetSourcePort()
				channelID = packet.GetSourceChannel()
				sequence = packet.GetSequence()
				commitment = channeltypes.CommitPacket(suite.chain.Codec, packet)
			},
			expPass: true,
		},
		{
			name: "proof verification failed: different commitment stored",
			malleate: func() {
				portID = packet.GetSourcePort()
				channelID = packet.GetSourceChannel()
				sequence = packet.GetSequence()
				commitment = []byte("commitment")
			},
			expPass: false,
		},
		{
			name: "proof verification failed: no commitment stored",
			malleate: func() {
				portID = testPortID
				channelID = testChannelID
				sequence = testSequence
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewLocalPath(suite.chain)
			suite.coordinator.Setup(path)

			// send packet
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, clienttypes.NewHeight(0, 100), 0)
			err := path.EndpointB.SendPacket(packet)
			suite.Require().NoError(err)

			tc.malleate()

			clientStateI := suite.chain.GetClientState(path.EndpointA.ClientID)
			clientState, ok := clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			ctx := suite.chain.GetContext()
			store := ctx.KVStore(suite.chain.App.GetKey(host.StoreKey))
			err = clientState.VerifyPacketCommitment(
				ctx, store, suite.chain.Codec, clienttypes.Height{}, 0, 0, nil, []byte{}, portID, channelID, sequence, commitment,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestVerifyPacketAcknowledgement() {
	var (
		packet    channeltypes.Packet
		portID    string
		channelID string
		sequence  uint64
		ack       []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "proof verification success",
			malleate: func() {
				portID = packet.GetDestPort()
				channelID = packet.GetDestChannel()
				sequence = packet.GetSequence()
				ack = ibcmock.MockAcknowledgement.Acknowledgement()
			},
			expPass: true,
		},
		{
			name: "proof verification failed: different ack stored",
			malleate: func() {
				portID = packet.GetDestPort()
				channelID = packet.GetDestChannel()
				sequence = packet.GetSequence()
				ack = channeltypes.NewResultAcknowledgement([]byte("different acknowledgement")).Acknowledgement()
			},
			expPass: false,
		},
		{
			name: "proof verification failed: no ack stored",
			malleate: func() {
				portID = testPortID
				channelID = testChannelID
				sequence = testSequence
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewLocalPath(suite.chain)
			suite.coordinator.Setup(path)

			// send packet
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)
			err := path.EndpointA.SendPacket(packet)
			suite.Require().NoError(err)

			// write receipt and ack
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			tc.malleate()

			clientStateI := suite.chain.GetClientState(path.EndpointA.ClientID)
			clientState, ok := clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			ctx := suite.chain.GetContext()
			store := ctx.KVStore(suite.chain.App.GetKey(host.StoreKey))
			err = clientState.VerifyPacketAcknowledgement(
				ctx, store, suite.chain.Codec, clienttypes.Height{}, 0, 0, nil, []byte{}, portID, channelID, sequence, ack,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestVerifyPacketReceiptAbsence() {
	suite.SetupTest()
	path := ibctesting.NewLocalPath(suite.chain)
	suite.coordinator.Setup(path)

	// send packet
	packet := channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)
	err := path.EndpointA.SendPacket(packet)
	suite.Require().NoError(err)

	clientStateI := suite.chain.GetClientState(path.EndpointA.ClientID)
	clientState, ok := clientStateI.(*types.ClientState)
	suite.Require().True(ok)

	ctx := suite.chain.GetContext()
	store := ctx.KVStore(suite.chain.App.GetKey(host.StoreKey))
	portID := packet.GetDestPort()
	channelID := packet.GetDestChannel()
	sequence := packet.GetSequence()
	err = clientState.VerifyPacketReceiptAbsence(
		ctx, store, suite.chain.Codec, clienttypes.Height{}, 0, 0, nil, nil, portID, channelID, sequence,
	)
	suite.Require().NoError(err, "receipt absence failed")

	// write receipt and ack
	err = path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	err = clientState.VerifyPacketReceiptAbsence(
		ctx, store, suite.chain.Codec, clienttypes.Height{}, 0, 0, nil, nil, portID, channelID, sequence,
	)
	suite.Require().Error(err, "receipt exists in store")
}

func (suite *LocalhostTestSuite) TestVerifyNextSeqRecv() {
	var (
		path         *ibctesting.Path
		packet       channeltypes.Packet
		portID       string
		channelID    string
		nextSequence uint64
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "proof verification success",
			malleate: func() {
				// send packet
				packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)
				err := path.EndpointA.SendPacket(packet)
				suite.Require().NoError(err)

				// write receipt and ack
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				portID = packet.GetDestPort()
				channelID = packet.GetDestChannel()
				nextSequence = packet.GetSequence() + 1
			},
			expPass: true,
		},
		{
			name: "proof verification failed: different nextSequence stored",
			malleate: func() {
				// send packet
				packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)
				err := path.EndpointA.SendPacket(packet)
				suite.Require().NoError(err)

				// write receipt and ack
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				portID = packet.GetDestPort()
				channelID = packet.GetDestChannel()
				nextSequence = packet.GetSequence()
			},
			expPass: false,
		},
		{
			name: "proof verification failed: no nextSequence stored",
			malleate: func() {
				portID = testPortID
				channelID = testChannelID
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewLocalPath(suite.chain)
			suite.coordinator.Setup(path)

			tc.malleate()

			clientStateI := suite.chain.GetClientState(path.EndpointA.ClientID)
			clientState, ok := clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			ctx := suite.chain.GetContext()
			store := ctx.KVStore(suite.chain.App.GetKey(host.StoreKey))
			err := clientState.VerifyNextSequenceRecv(
				ctx, store, suite.chain.Codec, clienttypes.Height{}, 0, 0, nil, nil, portID, channelID, nextSequence,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
