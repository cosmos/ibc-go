package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/ante"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type AnteTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	path *ibctesting.Path
}

// SetupTest creates a coordinator with 2 test chains.
func (s *AnteTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	s.coordinator.CommitNBlocks(s.chainA, 2)
	s.coordinator.CommitNBlocks(s.chainB, 2)
	s.path = ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.Setup(s.path)
}

// TestAnteTestSuite runs all the tests within this package.
func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

// createRecvPacketMessage creates a RecvPacket message for a packet sent from chain A to chain B.
func (s *AnteTestSuite) createRecvPacketMessage(isRedundant bool) sdk.Msg {
	sequence, err := s.path.EndpointA.SendPacket(clienttypes.NewHeight(2, 0), 0, ibctesting.MockPacketData)
	s.Require().NoError(err)

	packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence,
		s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID,
		clienttypes.NewHeight(2, 0), 0)

	if isRedundant {
		err = s.path.EndpointB.RecvPacket(packet)
		s.Require().NoError(err)
	}

	err = s.path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := s.chainA.QueryProof(packetKey)

	return channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createAcknowledgementMessage creates an Acknowledgement message for a packet sent from chain B to chain A.
func (s *AnteTestSuite) createAcknowledgementMessage(isRedundant bool) sdk.Msg {
	sequence, err := s.path.EndpointB.SendPacket(clienttypes.NewHeight(2, 0), 0, ibctesting.MockPacketData)
	s.Require().NoError(err)

	packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence,
		s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID,
		s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
		clienttypes.NewHeight(2, 0), 0)
	err = s.path.EndpointA.RecvPacket(packet)
	s.Require().NoError(err)

	if isRedundant {
		err = s.path.EndpointB.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
		s.Require().NoError(err)
	}

	packetKey := host.PacketAcknowledgementKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := s.chainA.QueryProof(packetKey)

	return channeltypes.NewMsgAcknowledgement(packet, ibctesting.MockAcknowledgement, proof, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createTimeoutMessage creates an Timeout message for a packet sent from chain B to chain A.
func (s *AnteTestSuite) createTimeoutMessage(isRedundant bool) sdk.Msg {
	height := s.chainA.LastHeader.GetHeight()
	timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)

	sequence, err := s.path.EndpointB.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
	s.Require().NoError(err)

	s.coordinator.CommitNBlocks(s.chainA, 3)

	err = s.path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence,
		s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID,
		s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
		timeoutHeight, 0)

	if isRedundant {
		err = s.path.EndpointB.TimeoutPacket(packet)
		s.Require().NoError(err)
	}

	packetKey := host.PacketReceiptKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := s.chainA.QueryProof(packetKey)

	return channeltypes.NewMsgTimeout(packet, sequence, proof, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createTimeoutOnCloseMessage creates an TimeoutOnClose message for a packet sent from chain B to chain A.
func (s *AnteTestSuite) createTimeoutOnCloseMessage(isRedundant bool) sdk.Msg {
	height := s.chainA.LastHeader.GetHeight()
	timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)

	sequence, err := s.path.EndpointB.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
	s.Require().NoError(err)
	err = s.path.EndpointA.SetChannelState(channeltypes.CLOSED)
	s.Require().NoError(err)

	packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence,
		s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID,
		s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
		timeoutHeight, 0)

	if isRedundant {
		err = s.path.EndpointB.TimeoutOnClose(packet)
		s.Require().NoError(err)
	}

	packetKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := s.chainA.QueryProof(packetKey)

	channelKey := host.ChannelKey(packet.GetDestPort(), packet.GetDestChannel())
	proofClosed, _ := s.chainA.QueryProof(channelKey)

	return channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, proofClosed, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

func (s *AnteTestSuite) createUpdateClientMessage() sdk.Msg {
	endpoint := s.path.EndpointB

	// ensure counterparty has committed state
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Counterparty.Chain)

	var header exported.ClientMessage

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		header, _ = endpoint.Chain.ConstructUpdateTMClientHeader(endpoint.Counterparty.Chain, endpoint.ClientID)

	default:
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		endpoint.ClientID, header,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.TB, err)

	return msg
}

func (s *AnteTestSuite) TestAnteDecorator() {
	testCases := []struct {
		name     string
		malleate func(antesuite *AnteTestSuite) []sdk.Msg
		expPass  bool
	}{
		{
			"success on one new RecvPacket message",
			func(anteantesuite *AnteTestSuite) []sdk.Msg {
				// the RecvPacket message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createRecvPacketMessage(false)}
			},
			true,
		},
		{
			"success on one new Acknowledgement message",
			func(anteantesuite *AnteTestSuite) []sdk.Msg {
				// the Acknowledgement message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createAcknowledgementMessage(false)}
			},
			true,
		},
		{
			"success on one new Timeout message",
			func(anteantesuite *AnteTestSuite) []sdk.Msg {
				// the Timeout message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createTimeoutMessage(false)}
			},
			true,
		},
		{
			"success on one new TimeoutOnClose message",
			func(anteantesuite *AnteTestSuite) []sdk.Msg {
				// the TimeoutOnClose message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createTimeoutOnCloseMessage(false)}
			},
			true,
		},
		{
			"success on three new messages of each type",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				// none of the messages of each type has been submitted to the chain yet,
				// the first message is succeed and the next two of each type will be rejected
				// because they are redundant.

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(false))
				}

				// from B to A
				for i := 1; i <= 9; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, s.createAcknowledgementMessage(false))
					case i >= 4 && i <= 6:
						msgs = append(msgs, s.createTimeoutMessage(false))
					case i >= 7 && i <= 9:
						msgs = append(msgs, s.createTimeoutOnCloseMessage(false))
					}
				}
				return msgs
			},
			true,
		},
		{
			"success on three redundant messages of RecvPacket, Acknowledgement and TimeoutOnClose, and one new Timeout message",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				// we pass three messages of RecvPacket, Acknowledgement and TimeoutOnClose that
				// are all redundant (i.e. those messages have already been submitted and
				// processed by the chain). But these messages will not be rejected because the
				// Timeout message is new.

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(true))
				}

				// from B to A
				for i := 1; i <= 7; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, s.createAcknowledgementMessage(true))
					case i == 4:
						msgs = append(msgs, s.createTimeoutMessage(false))
					case i >= 5 && i <= 7:
						msgs = append(msgs, s.createTimeoutOnCloseMessage(true))
					}
				}
				return msgs
			},
			true,
		},
		{
			"success on one new message and two redundant messages of each type",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				// For each type there is a new message and two messages that are redundant
				// (i.e. they have been already submitted and processed by the chain). But all
				// the redundant messages will not be rejected because there is a new message
				// of each type.

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(i != 2))
				}

				// from B to A
				for i := 1; i <= 9; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, s.createAcknowledgementMessage(i != 2))
					case i >= 4 && i <= 6:
						msgs = append(msgs, s.createTimeoutMessage(i != 5))
					case i >= 7 && i <= 9:
						msgs = append(msgs, s.createTimeoutOnCloseMessage(i != 8))
					}
				}
				return msgs
			},
			true,
		},
		{
			"success on one new UpdateClient message",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{s.createUpdateClientMessage()}
			},
			true,
		},
		{
			"success on three new UpdateClient messages",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{s.createUpdateClientMessage(), s.createUpdateClientMessage(), s.createUpdateClientMessage()}
			},
			true,
		},
		{
			"success on three new Updateclient messages and one new RecvPacket message",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{
					s.createUpdateClientMessage(),
					s.createUpdateClientMessage(),
					s.createUpdateClientMessage(),
					s.createRecvPacketMessage(false),
				}
			},
			true,
		},
		{
			"success on three redundant RecvPacket messages and one SubmitMisbehaviour message",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{s.createUpdateClientMessage()}

				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(true))
				}

				// append non packet and update message to msgs to ensure multimsg tx should pass
				msgs = append(msgs, &clienttypes.MsgSubmitMisbehaviour{})
				return msgs
			},
			true,
		},
		{
			"no success on one redundant RecvPacket message",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{s.createRecvPacketMessage(true)}
			},
			false,
		},
		{
			"no success on three redundant messages of each type",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(true))
				}

				// from B to A
				for i := 1; i <= 9; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, s.createAcknowledgementMessage(true))
					case i >= 4 && i <= 6:
						msgs = append(msgs, s.createTimeoutMessage(true))
					case i >= 7 && i <= 9:
						msgs = append(msgs, s.createTimeoutOnCloseMessage(true))
					}
				}
				return msgs
			},
			false,
		},
		{
			"no success on one new UpdateClient message and three redundant RecvPacket messages",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{&clienttypes.MsgUpdateClient{}}

				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(true))
				}

				return msgs
			},
			false,
		},
		{
			"no success on three new UpdateClient messages and three redundant messages of each type",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{s.createUpdateClientMessage(), s.createUpdateClientMessage(), s.createUpdateClientMessage()}

				// from A to B
				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(true))
				}

				// from B to A
				for i := 1; i <= 9; i++ {
					switch {
					case i >= 1 && i <= 3:
						msgs = append(msgs, s.createAcknowledgementMessage(true))
					case i >= 4 && i <= 6:
						msgs = append(msgs, s.createTimeoutMessage(true))
					case i >= 7 && i <= 9:
						msgs = append(msgs, s.createTimeoutOnCloseMessage(true))
					}
				}
				return msgs
			},
			false,
		},
		{
			"no success on one new message and one invalid message",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, 2,
					s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
					s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID,
					clienttypes.NewHeight(2, 0), 0)

				return []sdk.Msg{
					s.createRecvPacketMessage(false),
					channeltypes.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(1, 1), "signer"),
				}
			},
			false,
		},
		{
			"no success on one new message and one redundant message in the same block",
			func(antesuite *AnteTestSuite) []sdk.Msg {
				msg := s.createRecvPacketMessage(false)

				// We want to be able to run check tx with the non-redundant message without
				// committing it to a block, so that the when check tx runs with the redundant
				// message they are both in the same block
				k := s.chainB.App.GetIBCKeeper()
				decorator := ante.NewRedundantRelayDecorator(k)
				checkCtx := s.chainB.GetContext().WithIsCheckTx(true)
				next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) { return ctx, nil }
				txBuilder := s.chainB.TxConfig.NewTxBuilder()
				err := txBuilder.SetMsgs([]sdk.Msg{msg}...)
				s.Require().NoError(err)
				tx := txBuilder.GetTx()

				_, err = decorator.AnteHandle(checkCtx, tx, false, next)
				s.Require().NoError(err)

				return []sdk.Msg{msg}
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			// reset suite
			s.SetupTest()

			k := s.chainB.App.GetIBCKeeper()
			decorator := ante.NewRedundantRelayDecorator(k)

			msgs := tc.malleate(s)

			deliverCtx := s.chainB.GetContext().WithIsCheckTx(false)
			checkCtx := s.chainB.GetContext().WithIsCheckTx(true)

			// create multimsg tx
			txBuilder := s.chainB.TxConfig.NewTxBuilder()
			err := txBuilder.SetMsgs(msgs...)
			s.Require().NoError(err)
			tx := txBuilder.GetTx()

			next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) { return ctx, nil }

			_, err = decorator.AnteHandle(deliverCtx, tx, false, next)
			s.Require().NoError(err, "antedecorator should not error on DeliverTx")

			_, err = decorator.AnteHandle(checkCtx, tx, false, next)
			if tc.expPass {
				s.Require().NoError(err, "non-strict decorator did not pass as expected")
			} else {
				s.Require().Error(err, "non-strict antehandler did not return error as expected")
			}
		})
	}
}
