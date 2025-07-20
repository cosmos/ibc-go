package ante_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/tmhash"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtprotoversion "github.com/cometbft/cometbft/proto/tendermint/version"
	cmttypes "github.com/cometbft/cometbft/types"
	cmtversion "github.com/cometbft/cometbft/version"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/ante"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

type AnteTestSuite struct {
	testifysuite.Suite

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
	s.path.Setup()
}

// TestAnteTestSuite runs all the tests within this package.
func TestAnteTestSuite(t *testing.T) {
	testifysuite.Run(t, new(AnteTestSuite))
}

// createRecvPacketMessage creates a RecvPacket message for a packet sent from chain A to chain B.
func (s *AnteTestSuite) createRecvPacketMessage(isRedundant bool) *channeltypes.MsgRecvPacket {
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

// createRecvPacketMessageV2 creates a V2 RecvPacket message for a packet sent from chain A to chain B.
func (s *AnteTestSuite) createRecvPacketMessageV2(isRedundant bool) *channeltypesv2.MsgRecvPacket {
	packet, err := s.path.EndpointA.MsgSendPacket(s.chainA.GetTimeoutTimestampSecs(), mock.NewMockPayload(mock.ModuleNameA, mock.ModuleNameB))
	s.Require().NoError(err)

	if isRedundant {
		err = s.path.EndpointB.MsgRecvPacket(packet)
		s.Require().NoError(err)
	}

	err = s.path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	packetKey := hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence)
	proof, proofHeight := s.chainA.QueryProof(packetKey)

	return channeltypesv2.NewMsgRecvPacket(packet, proof, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
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

	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := s.chainA.QueryProof(packetKey)

	return channeltypes.NewMsgAcknowledgement(packet, ibctesting.MockAcknowledgement, proof, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createAcknowledgementMessageV2 creates a V2 Acknowledgement message for a packet sent from chain B to chain A.
func (s *AnteTestSuite) createAcknowledgementMessageV2(isRedundant bool) *channeltypesv2.MsgAcknowledgement {
	packet, err := s.path.EndpointB.MsgSendPacket(s.chainB.GetTimeoutTimestampSecs(), mock.NewMockPayload(mock.ModuleNameA, mock.ModuleNameB))
	s.Require().NoError(err)

	err = s.path.EndpointA.MsgRecvPacket(packet)
	s.Require().NoError(err)

	ack := channeltypesv2.Acknowledgement{AppAcknowledgements: [][]byte{mock.MockRecvPacketResult.Acknowledgement}}
	if isRedundant {
		err = s.path.EndpointB.MsgAcknowledgePacket(packet, ack)
		s.Require().NoError(err)
	}

	packetKey := hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence)
	proof, proofHeight := s.chainA.QueryProof(packetKey)

	return channeltypesv2.NewMsgAcknowledgement(packet, ack, proof, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createTimeoutMessage creates an Timeout message for a packet sent from chain B to chain A.
func (s *AnteTestSuite) createTimeoutMessage(isRedundant bool) sdk.Msg {
	height := s.chainA.LatestCommittedHeader.GetHeight()
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

// createTimeoutMessageV2 creates a V2 Timeout message for a packet sent from chain B to chain A.
func (s *AnteTestSuite) createTimeoutMessageV2(isRedundant bool) *channeltypesv2.MsgTimeout {
	timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Second).Unix())
	packet, err := s.path.EndpointB.MsgSendPacket(timeoutTimestamp, mock.NewMockPayload(mock.ModuleNameA, mock.ModuleNameB))
	s.Require().NoError(err)

	s.coordinator.IncrementTimeBy(time.Hour)
	err = s.path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	if isRedundant {
		err = s.path.EndpointB.MsgTimeoutPacket(packet)
		s.Require().NoError(err)
	}

	packetKey := hostv2.PacketReceiptKey(packet.SourceClient, packet.Sequence)
	proof, proofHeight := s.chainA.QueryProof(packetKey)

	return channeltypesv2.NewMsgTimeout(packet, proof, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

// createTimeoutOnCloseMessage creates an TimeoutOnClose message for a packet sent from chain B to chain A.
func (s *AnteTestSuite) createTimeoutOnCloseMessage(isRedundant bool) sdk.Msg {
	height := s.chainA.LatestCommittedHeader.GetHeight()
	timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)

	sequence, err := s.path.EndpointB.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
	s.Require().NoError(err)
	s.path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })

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
	closedProof, _ := s.chainA.QueryProof(channelKey)

	return channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, closedProof, proofHeight, s.path.EndpointA.Chain.SenderAccount.GetAddress().String())
}

func (s *AnteTestSuite) createUpdateClientMessage() sdk.Msg {
	endpoint := s.path.EndpointB

	// ensure counterparty has committed state
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Counterparty.Chain)

	var header exported.ClientMessage

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		trustedHeight := endpoint.GetClientLatestHeight()
		header, _ = endpoint.Counterparty.Chain.IBCClientHeader(endpoint.Counterparty.Chain.LatestCommittedHeader, trustedHeight.(clienttypes.Height))

	default:
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		endpoint.ClientID, header,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.TB, err)

	return msg
}

func (s *AnteTestSuite) createMaliciousUpdateClientMessage() sdk.Msg {
	endpoint := s.path.EndpointB

	// ensure counterparty has committed state
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Counterparty.Chain)

	trustedHeight, ok := endpoint.GetClientLatestHeight().(clienttypes.Height)
	if !ok {
		require.True(endpoint.Chain.TB, ok, "bad height conversion")
	}

	currentHeader := endpoint.Counterparty.Chain.LatestCommittedHeader.Header

	validators := endpoint.Counterparty.Chain.Vals.Validators
	signers := endpoint.Counterparty.Chain.Signers

	// Signers must be in the same order as
	// the validators when signing.
	signerArr := make([]cmttypes.PrivValidator, len(validators))
	for i, v := range validators {
		signerArr[i] = signers[v.Address.String()]
	}
	cmtTrustedVals, ok := endpoint.Counterparty.Chain.TrustedValidators[trustedHeight.RevisionHeight]
	if !ok {
		require.True(endpoint.Chain.TB, ok, "no validators")
	}

	maliciousHeader, err := createMaliciousTMHeader(endpoint.Counterparty.Chain.ChainID, int64(trustedHeight.RevisionHeight+1), trustedHeight, currentHeader.Time, endpoint.Counterparty.Chain.Vals, cmtTrustedVals, signerArr, currentHeader)
	require.NoError(endpoint.Chain.TB, err, "invalid header update")

	msg, err := clienttypes.NewMsgUpdateClient(
		endpoint.ClientID, maliciousHeader,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.TB, err, "msg update")

	return msg
}

func (s *AnteTestSuite) TestAnteDecoratorCheckTx() {
	testCases := []struct {
		name     string
		malleate func(s *AnteTestSuite) []sdk.Msg
		expError error
	}{
		{
			"success on one new RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				// the RecvPacket message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createRecvPacketMessage(false)}
			},
			nil,
		},
		{
			"success on one new V2 RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				s.path.SetupV2()
				// the RecvPacket message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createRecvPacketMessageV2(false)}
			},
			nil,
		},
		{
			"success on one new Acknowledgement message",
			func(s *AnteTestSuite) []sdk.Msg {
				// the Acknowledgement message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createAcknowledgementMessage(false)}
			},
			nil,
		},
		{
			"success on one new V2 Acknowledgement message",
			func(s *AnteTestSuite) []sdk.Msg {
				s.path.SetupV2()
				// the Acknowledgement message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createAcknowledgementMessageV2(false)}
			},
			nil,
		},
		{
			"success on one new Timeout message",
			func(s *AnteTestSuite) []sdk.Msg {
				// the Timeout message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createTimeoutMessage(false)}
			},
			nil,
		},
		{
			"success on one new Timeout V2 message",
			func(s *AnteTestSuite) []sdk.Msg {
				s.path.SetupV2()
				// the Timeout message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createTimeoutMessageV2(false)}
			},
			nil,
		},
		{
			"success on one new TimeoutOnClose message",
			func(s *AnteTestSuite) []sdk.Msg {
				// the TimeoutOnClose message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createTimeoutOnCloseMessage(false)}
			},
			nil,
		},
		{
			"success on three new messages of each type",
			func(s *AnteTestSuite) []sdk.Msg {
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
			nil,
		},
		{
			"success on three redundant messages of RecvPacket, Acknowledgement and TimeoutOnClose, and one new Timeout message",
			func(s *AnteTestSuite) []sdk.Msg {
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
			nil,
		},
		{
			"success on one new message and two redundant messages of each type",
			func(s *AnteTestSuite) []sdk.Msg {
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
			nil,
		},
		{
			"success on one new UpdateClient message",
			func(s *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{s.createUpdateClientMessage()}
			},
			nil,
		},
		{
			"success on three new UpdateClient messages",
			func(s *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{s.createUpdateClientMessage(), s.createUpdateClientMessage(), s.createUpdateClientMessage()}
			},
			nil,
		},
		{
			"success on three new Updateclient messages and one new RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{
					s.createUpdateClientMessage(),
					s.createUpdateClientMessage(),
					s.createUpdateClientMessage(),
					s.createRecvPacketMessage(false),
				}
			},
			nil,
		},
		{
			"success on app callback error, app callbacks are skipped for performance",
			func(s *AnteTestSuite) []sdk.Msg {
				s.chainB.GetSimApp().IBCMockModule.IBCApp.OnRecvPacket = func(
					ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress,
				) exported.Acknowledgement {
					panic(errors.New("failed OnRecvPacket mock callback"))
				}

				// the RecvPacket message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createRecvPacketMessage(false)}
			},
			nil,
		},
		{
			"no success on one redundant RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{s.createRecvPacketMessage(true)}
			},
			channeltypes.ErrRedundantTx,
		},
		{
			"no success on one redundant V2 RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				s.path.SetupV2()
				return []sdk.Msg{s.createRecvPacketMessageV2(true)}
			},
			channeltypes.ErrRedundantTx,
		},
		{
			"no success on three redundant messages of each type",
			func(s *AnteTestSuite) []sdk.Msg {
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
			channeltypes.ErrRedundantTx,
		},
		{
			"no success on one new UpdateClient message and three redundant RecvPacket messages",
			func(s *AnteTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{s.createUpdateClientMessage()}

				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(true))
				}

				return msgs
			},
			channeltypes.ErrRedundantTx,
		},
		{
			"no success on one new malicious UpdateClient message and three redundant RecvPacket messages",
			func(s *AnteTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{s.createMaliciousUpdateClientMessage()}

				for i := 1; i <= 3; i++ {
					msgs = append(msgs, s.createRecvPacketMessage(true))
				}

				return msgs
			},
			channeltypes.ErrRedundantTx,
		},
		{
			"no success on one new UpdateClient message: invalid client identifier",
			func(s *AnteTestSuite) []sdk.Msg {
				clientMsg, err := codectypes.NewAnyWithValue(&ibctm.Header{})
				s.Require().NoError(err)

				msgs := []sdk.Msg{&clienttypes.MsgUpdateClient{ClientId: ibctesting.InvalidID, ClientMessage: clientMsg}}
				return msgs
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"no success on one new UpdateClient message: client module not found",
			func(s *AnteTestSuite) []sdk.Msg {
				clientMsg, err := codectypes.NewAnyWithValue(&ibctm.Header{})
				s.Require().NoError(err)

				msgs := []sdk.Msg{&clienttypes.MsgUpdateClient{ClientId: clienttypes.FormatClientIdentifier("08-wasm", 1), ClientMessage: clientMsg}}
				return msgs
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"no success on one new UpdateClient message: no consensus state for trusted height",
			func(s *AnteTestSuite) []sdk.Msg {
				clientMsg, err := codectypes.NewAnyWithValue(&ibctm.Header{TrustedHeight: clienttypes.NewHeight(1, 10000)})
				s.Require().NoError(err)

				msgs := []sdk.Msg{&clienttypes.MsgUpdateClient{ClientId: s.path.EndpointA.ClientID, ClientMessage: clientMsg}}
				return msgs
			},
			clienttypes.ErrConsensusStateNotFound,
		},
		{
			"no success on three new UpdateClient messages and three redundant messages of each type",
			func(s *AnteTestSuite) []sdk.Msg {
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
			channeltypes.ErrRedundantTx,
		},
		{
			"no success on one new message and one invalid message",
			func(s *AnteTestSuite) []sdk.Msg {
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, 2,
					s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
					s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID,
					clienttypes.NewHeight(2, 0), 0)

				return []sdk.Msg{
					s.createRecvPacketMessage(false),
					channeltypes.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(1, 1), "signer"),
				}
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"no success on one new message and one redundant message in the same block",
			func(s *AnteTestSuite) []sdk.Msg {
				msg := s.createRecvPacketMessage(false)

				// We want to be able to run check tx with the non-redundant message without
				// committing it to a block, so that the when check tx runs with the redundant
				// message they are both in the same block
				k := s.chainB.App.GetIBCKeeper()
				decorator := ante.NewRedundantRelayDecorator(k)
				checkCtx := s.chainB.GetContext().WithIsCheckTx(true)
				next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) { return ctx, nil }
				txBuilder := s.chainB.TxConfig.NewTxBuilder()
				err := txBuilder.SetMsgs([]sdk.Msg{msg}...)
				s.Require().NoError(err)
				tx := txBuilder.GetTx()

				_, err = decorator.AnteHandle(checkCtx, tx, false, next)
				s.Require().NoError(err)

				return []sdk.Msg{msg}
			},
			channeltypes.ErrRedundantTx,
		},
	}

	for _, tc := range testCases {
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

			next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) { return ctx, nil }

			_, err = decorator.AnteHandle(deliverCtx, tx, false, next)
			s.Require().NoError(err, "antedecorator should not error on DeliverTx")

			_, err = decorator.AnteHandle(checkCtx, tx, false, next)
			if tc.expError == nil {
				s.Require().NoError(err, "non-strict decorator did not pass as expected")
			} else {
				s.Require().ErrorIs(err, tc.expError, "non-strict antehandler did not return error as expected")
			}
		})
	}
}

func (s *AnteTestSuite) TestAnteDecoratorReCheckTx() {
	testCases := []struct {
		name     string
		malleate func(s *AnteTestSuite) []sdk.Msg
		expError error
	}{
		{
			"success on one new RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				// the RecvPacket message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createRecvPacketMessage(false)}
			},
			nil,
		},
		{
			"success on one new V2 RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				s.path.SetupV2()
				// the RecvPacket message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createRecvPacketMessageV2(false)}
			},
			nil,
		},
		{
			"success on one redundant and one new RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{
					s.createRecvPacketMessage(true),
					s.createRecvPacketMessage(false),
				}
			},
			nil,
		},
		{
			"success on invalid proof (proof checks occur in checkTx)",
			func(s *AnteTestSuite) []sdk.Msg {
				msg := s.createRecvPacketMessage(false)
				msg.ProofCommitment = []byte("invalid-proof")
				return []sdk.Msg{msg}
			},
			nil,
		},
		{
			"success on app callback error, app callbacks are skipped for performance",
			func(s *AnteTestSuite) []sdk.Msg {
				s.chainB.GetSimApp().IBCMockModule.IBCApp.OnRecvPacket = func(
					ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress,
				) exported.Acknowledgement {
					panic(errors.New("failed OnRecvPacket mock callback"))
				}

				// the RecvPacket message has not been submitted to the chain yet, so it will succeed
				return []sdk.Msg{s.createRecvPacketMessage(false)}
			},
			nil,
		},
		{
			"no success on one redundant RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				return []sdk.Msg{s.createRecvPacketMessage(true)}
			},
			channeltypes.ErrRedundantTx,
		},
		{
			"no success on one redundant V2 RecvPacket message",
			func(s *AnteTestSuite) []sdk.Msg {
				s.path.SetupV2()
				return []sdk.Msg{s.createRecvPacketMessageV2(true)}
			},
			channeltypes.ErrRedundantTx,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// reset suite
			s.SetupTest()

			k := s.chainB.App.GetIBCKeeper()
			decorator := ante.NewRedundantRelayDecorator(k)

			msgs := tc.malleate(s)

			deliverCtx := s.chainB.GetContext().WithIsCheckTx(false)
			reCheckCtx := s.chainB.GetContext().WithIsReCheckTx(true)

			// create multimsg tx
			txBuilder := s.chainB.TxConfig.NewTxBuilder()
			err := txBuilder.SetMsgs(msgs...)
			s.Require().NoError(err)
			tx := txBuilder.GetTx()

			next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) { return ctx, nil }
			_, err = decorator.AnteHandle(deliverCtx, tx, false, next)
			s.Require().NoError(err, "antedecorator should not error on DeliverTx")

			_, err = decorator.AnteHandle(reCheckCtx, tx, false, next)
			if tc.expError == nil {
				s.Require().NoError(err, "non-strict decorator did not pass as expected")
			} else {
				s.Require().ErrorIs(err, tc.expError, "non-strict antehandler did not return error as expected")
			}
		})
	}
}

// createMaliciousTMHeader creates a header with the provided trusted height with an invalid app hash.
func createMaliciousTMHeader(chainID string, blockHeight int64, trustedHeight clienttypes.Height, timestamp time.Time, tmValSet, tmTrustedVals *cmttypes.ValidatorSet, signers []cmttypes.PrivValidator, oldHeader *cmtproto.Header) (*ibctm.Header, error) {
	const (
		invalidHashValue = "invalid_hash"
	)

	tmHeader := cmttypes.Header{
		Version:            cmtprotoversion.Consensus{Block: cmtversion.BlockProtocol, App: 2},
		ChainID:            chainID,
		Height:             blockHeight,
		Time:               timestamp,
		LastBlockID:        ibctesting.MakeBlockID(make([]byte, tmhash.Size), 10_000, make([]byte, tmhash.Size)),
		LastCommitHash:     oldHeader.GetLastCommitHash(),
		ValidatorsHash:     tmValSet.Hash(),
		NextValidatorsHash: tmValSet.Hash(),
		DataHash:           tmhash.Sum([]byte(invalidHashValue)),
		ConsensusHash:      tmhash.Sum([]byte(invalidHashValue)),
		AppHash:            tmhash.Sum([]byte(invalidHashValue)),
		LastResultsHash:    tmhash.Sum([]byte(invalidHashValue)),
		EvidenceHash:       tmhash.Sum([]byte(invalidHashValue)),
		ProposerAddress:    tmValSet.Proposer.Address, //nolint:staticcheck
	}

	hhash := tmHeader.Hash()
	blockID := ibctesting.MakeBlockID(hhash, 3, tmhash.Sum([]byte(invalidHashValue)))
	voteSet := cmttypes.NewVoteSet(chainID, blockHeight, 1, cmtproto.PrecommitType, tmValSet)

	extCommit, err := cmttypes.MakeExtCommit(blockID, blockHeight, 1, voteSet, signers, timestamp, false)
	if err != nil {
		return nil, err
	}

	signedHeader := &cmtproto.SignedHeader{
		Header: tmHeader.ToProto(),
		Commit: extCommit.ToCommit().ToProto(),
	}

	valSet, err := tmValSet.ToProto()
	if err != nil {
		return nil, err
	}

	trustedVals, err := tmTrustedVals.ToProto()
	if err != nil {
		return nil, err
	}

	return &ibctm.Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedVals,
	}, nil
}
