package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/auth/middleware"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/cosmos/ibc-go/v3/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

type MiddlewareTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	path *ibctesting.Path
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *MiddlewareTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(suite.path)
}

// TestMiddlewareTestSuite runs all the tests within this package.
func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

// createRecvPacketMessage creates a RecvPacket message for a packet sent from chain A to chain B.
func (suite *MiddlewareTestSuite) createRecvPacketMessage(sequenceNumber uint64, isRedundant bool) sdk.Msg {
	packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequenceNumber,
		suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 0), 0)

	err := suite.path.EndpointA.SendPacket(packet)
	suite.Require().NoError(err)

	if isRedundant {
		err = suite.path.EndpointB.RecvPacket(packet)
		suite.Require().NoError(err)
	}

	err = suite.path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := suite.chainA.QueryProof(packetKey)

	return channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, suite.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createAcknowledgementMessage creates an Acknowledgement message for a packet sent from chain B to chain A.
func (suite *MiddlewareTestSuite) createAcknowledgementMessage(sequenceNumber uint64, isRedundant bool) sdk.Msg {
	packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequenceNumber,
		suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
		suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
		clienttypes.NewHeight(1, 0), 0)

	err := suite.path.EndpointB.SendPacket(packet)
	suite.Require().NoError(err)
	err = suite.path.EndpointA.RecvPacket(packet)
	suite.Require().NoError(err)

	if isRedundant {
		err = suite.path.EndpointB.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
		suite.Require().NoError(err)
	}

	packetKey := host.PacketAcknowledgementKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := suite.chainA.QueryProof(packetKey)

	return channeltypes.NewMsgAcknowledgement(packet, ibctesting.MockAcknowledgement, proof, proofHeight, suite.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createTimeoutMessage creates an Timeout message for a packet sent from chain B to chain A.
func (suite *MiddlewareTestSuite) createTimeoutMessage(sequenceNumber uint64, isRedundant bool) sdk.Msg {
	height := suite.chainA.LastHeader.GetHeight()
	timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)
	packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequenceNumber,
		suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
		suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
		timeoutHeight, 0)

	err := suite.path.EndpointB.SendPacket(packet)
	suite.Require().NoError(err)

	suite.coordinator.CommitNBlocks(suite.chainA, 3)

	err = suite.path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	if isRedundant {
		err = suite.path.EndpointB.TimeoutPacket(packet)
		suite.Require().NoError(err)
	}

	packetKey := host.PacketReceiptKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := suite.chainA.QueryProof(packetKey)

	return channeltypes.NewMsgTimeout(packet, sequenceNumber, proof, proofHeight, suite.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createTimeoutOnCloseMessage creates an TimeoutOnClose message for a packet sent from chain B to chain A.
func (suite *MiddlewareTestSuite) createTimeoutOnCloseMessage(sequenceNumber uint64, isRedundant bool) sdk.Msg {
	height := suite.chainA.LastHeader.GetHeight()
	timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)
	packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequenceNumber,
		suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
		suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
		timeoutHeight, 0)

	err := suite.path.EndpointB.SendPacket(packet)
	suite.Require().NoError(err)
	err = suite.path.EndpointA.SetChannelClosed()
	suite.Require().NoError(err)

	if isRedundant {
		err = suite.path.EndpointB.TimeoutOnClose(packet)
		suite.Require().NoError(err)
	}

	packetKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := suite.chainA.QueryProof(packetKey)

	channelKey := host.ChannelKey(packet.GetDestPort(), packet.GetDestChannel())
	proofClosed, _ := suite.chainA.QueryProof(channelKey)

	return channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, proofClosed, proofHeight, suite.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

func (suite *MiddlewareTestSuite) createUpdateClientMessage() sdk.Msg {
	endpoint := suite.path.EndpointB

	// ensure counterparty has committed state
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Counterparty.Chain)

	var header exported.Header

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		header, _ = endpoint.Chain.ConstructUpdateTMClientHeader(endpoint.Counterparty.Chain, endpoint.ClientID)

	default:
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		endpoint.ClientID, header,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.T, err)

	return msg
}

func (suite *MiddlewareTestSuite) TestMiddleware() {
	testCases := []struct {
		name     string
		malleate func(suite *MiddlewareTestSuite) []sdk.Msg
		expPass  bool
	}{
		{
			"success on one new RecvPacket message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				// the RecvPacket message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{suite.createRecvPacketMessage(1, false)}
			},
			true,
		},
		{
			"success on one new Acknowledgement message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				// the Acknowledgement message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{suite.createAcknowledgementMessage(1, false)}
			},
			true,
		},
		{
			"success on one new Timeout message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				// the Timeout message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{suite.createTimeoutMessage(1, false)}
			},
			true,
		},
		{
			"success on one new TimeoutOnClose message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				// the TimeoutOnClose message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{suite.createTimeoutOnCloseMessage(uint64(1), false)}
			},
			true,
		},
		{
			"success on three new messages of each type",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				// none of the messages of each type has been submitted to the chain yet,
				// the first message is succeed and the next two of each type will be rejected
				// because they are redundant.

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, suite.createRecvPacketMessage(uint64(i), false))
				}

				// from B to A
				for i := 1; i <= 9; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, suite.createAcknowledgementMessage(uint64(i), false))
					case i >= 4 && i <= 6:
						msgs = append(msgs, suite.createTimeoutMessage(uint64(i), false))
					case i >= 7 && i <= 9:
						msgs = append(msgs, suite.createTimeoutOnCloseMessage(uint64(i), false))
					}
				}
				return msgs
			},
			true,
		},
		{
			"success on three redundant messages of RecvPacket, Acknowledgement and TimeoutOnClose, and one new Timeout message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				// we pass three messages of RecvPacket, Acknowledgement and TimeoutOnClose that
				// are all redundant (i.e. those messages have already been submitted and
				// processed by the chain). But these messages will not be rejected because the
				// Timeout message is new.

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, suite.createRecvPacketMessage(uint64(i), true))
				}

				// from B to A
				for i := 1; i <= 7; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, suite.createAcknowledgementMessage(uint64(i), true))
					case i == 4:
						msgs = append(msgs, suite.createTimeoutMessage(uint64(i), false))
					case i >= 5 && i <= 7:
						msgs = append(msgs, suite.createTimeoutOnCloseMessage(uint64(i), true))
					}
				}
				return msgs
			},
			true,
		},
		{
			"success on one new message and two redundant messages of each type",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				// For each type there is a new message and two messages that are redundant
				// (i.e. they have been already submitted and processed by the chain). But all
				// the redundant messages will not be rejected because there is a new message
				// of each type.

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, suite.createRecvPacketMessage(uint64(i), i != 2))
				}

				// from B to A
				for i := 1; i <= 9; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, suite.createAcknowledgementMessage(uint64(i), i != 2))
					case i >= 4 && i <= 6:
						msgs = append(msgs, suite.createTimeoutMessage(uint64(i), i != 5))
					case i >= 7 && i <= 9:
						msgs = append(msgs, suite.createTimeoutOnCloseMessage(uint64(i), i != 8))
					}
				}
				return msgs
			},
			true,
		},
		{
			"success on one new UpdateClient message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				return []sdk.Msg{suite.createUpdateClientMessage()}
			},
			true,
		},
		{
			"success on three new UpdateClient messages",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				return []sdk.Msg{suite.createUpdateClientMessage(), suite.createUpdateClientMessage(), suite.createUpdateClientMessage()}
			},
			true,
		},
		{
			"success on three new Updateclient messages and one new RecvPacket message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				return []sdk.Msg{
					suite.createUpdateClientMessage(),
					suite.createUpdateClientMessage(),
					suite.createUpdateClientMessage(),
					suite.createRecvPacketMessage(uint64(1), false),
				}
			},
			true,
		},
		{
			"success on three redundant RecvPacket messages and one SubmitMisbehaviour message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{suite.createUpdateClientMessage()}

				for i := 1; i <= 3; i++ {
					msgs = append(msgs, suite.createRecvPacketMessage(uint64(i), true))
				}

				// append non packet and update message to msgs to ensure multimsg tx should pass
				msgs = append(msgs, &clienttypes.MsgSubmitMisbehaviour{})
				return msgs
			},
			true,
		},
		{
			"no success on one redundant RecvPacket message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				return []sdk.Msg{suite.createRecvPacketMessage(uint64(1), true)}
			},
			false,
		},
		{
			"no success on three redundant messages of each type",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, suite.createRecvPacketMessage(uint64(i), true))
				}

				// from B to A
				for i := 1; i <= 9; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, suite.createAcknowledgementMessage(uint64(i), true))
					case i >= 4 && i <= 6:
						msgs = append(msgs, suite.createTimeoutMessage(uint64(i), true))
					case i >= 7 && i <= 9:
						msgs = append(msgs, suite.createTimeoutOnCloseMessage(uint64(i), true))
					}
				}
				return msgs
			},
			false,
		},
		{
			"no success on one new UpdateClient message and three redundant RecvPacket messages",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{&clienttypes.MsgUpdateClient{}}

				for i := 1; i <= 3; i++ {
					msgs = append(msgs, suite.createRecvPacketMessage(uint64(i), true))
				}

				return msgs
			},
			false,
		},
		{
			"no success on three new UpdateClient messages and three redundant messages of each type",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{suite.createUpdateClientMessage(), suite.createUpdateClientMessage(), suite.createUpdateClientMessage()}

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, suite.createRecvPacketMessage(uint64(i), true))
				}

				// from B to A
				for i := 1; i <= 9; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, suite.createAcknowledgementMessage(uint64(i), true))
					case i >= 4 && i <= 6:
						msgs = append(msgs, suite.createTimeoutMessage(uint64(i), true))
					case i >= 7 && i <= 9:
						msgs = append(msgs, suite.createTimeoutOnCloseMessage(uint64(i), true))
					}
				}
				return msgs
			},
			false,
		},
		{
			"no success on one new message and one invalid message",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, 2,
					suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
					suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
					clienttypes.NewHeight(1, 0), 0)

				return []sdk.Msg{
					suite.createRecvPacketMessage(uint64(1), false),
					channeltypes.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"),
				}
			},
			false,
		},
		{
			"no success on one new message and one redundant message in the same block",
			func(suite *MiddlewareTestSuite) []sdk.Msg {
				msg := suite.createRecvPacketMessage(uint64(1), false)

				// We want to be able to run check tx with the non-redundant message without
				// commiting it to a block, so that the when check tx runs with the redundant
				// message they are both in the same block
				k := suite.chainB.App.GetIBCKeeper()
				mw := keeper.IBCTxMiddleware(k)
				checkCtx := suite.chainB.GetContext().WithIsCheckTx(true)
				txHandler := middleware.ComposeMiddlewares(noopTxHandler, mw)

				txBuilder := suite.chainB.TxConfig.NewTxBuilder()
				err := txBuilder.SetMsgs([]sdk.Msg{msg}...)
				suite.Require().NoError(err)
				tx := txBuilder.GetTx()

				_, _, err = txHandler.CheckTx(sdk.WrapSDKContext(checkCtx), txtypes.Request{Tx: tx}, txtypes.RequestCheckTx{})
				suite.Require().NoError(err)

				return []sdk.Msg{msg}
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()

			k := suite.chainB.App.GetIBCKeeper()
			mw := keeper.IBCTxMiddleware(k)

			msgs := tc.malleate(suite)

			deliverCtx := suite.chainB.GetContext().WithIsCheckTx(false)
			checkCtx := suite.chainB.GetContext().WithIsCheckTx(true)

			txHandler := middleware.ComposeMiddlewares(noopTxHandler, mw)

			// create multimsg tx
			txBuilder := suite.chainB.TxConfig.NewTxBuilder()
			err := txBuilder.SetMsgs(msgs...)
			suite.Require().NoError(err)
			tx := txBuilder.GetTx()

			_, err = txHandler.DeliverTx(sdk.WrapSDKContext(deliverCtx), txtypes.Request{Tx: tx})
			suite.Require().NoError(err, "middleware should not error on DeliverTx")

			_, _, err = txHandler.CheckTx(sdk.WrapSDKContext(checkCtx), txtypes.Request{Tx: tx}, txtypes.RequestCheckTx{})
			if tc.expPass {
				suite.Require().NoError(err, "non-strict middleware did not pass as expected")
			} else {
				suite.Require().Error(err, "non-strict middleware did not return error as expected")
			}
		})
	}
}

// customTxHandler is a test middleware that will run a custom function.
type customTxHandler struct {
	fn func(context.Context, txtypes.Request) (txtypes.Response, error)
}

var _ txtypes.Handler = customTxHandler{}

func (h customTxHandler) DeliverTx(ctx context.Context, req txtypes.Request) (txtypes.Response, error) {
	return h.fn(ctx, req)
}

func (h customTxHandler) CheckTx(ctx context.Context, req txtypes.Request, _ txtypes.RequestCheckTx) (txtypes.Response, txtypes.ResponseCheckTx, error) {
	res, err := h.fn(ctx, req)
	return res, txtypes.ResponseCheckTx{}, err
}

func (h customTxHandler) SimulateTx(ctx context.Context, req txtypes.Request) (txtypes.Response, error) {
	return h.fn(ctx, req)
}

// noopTxHandler is a test middleware that returns an empty response.
var noopTxHandler = customTxHandler{func(_ context.Context, _ txtypes.Request) (txtypes.Response, error) {
	return txtypes.Response{}, nil
}}
