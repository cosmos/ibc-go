//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	localhost "github.com/cosmos/ibc-go/v10/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestInterchainAccountsLocalhostTestSuite(t *testing.T) {
	testifysuite.Run(t, new(LocalhostInterchainAccountsTestSuite))
}

// compatibility:from_version: v7.10.0
type LocalhostInterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
}

// compatibility:TestInterchainAccounts_Localhost:from_versions: v7.10.0,v8.7.0,v10.0.0
func (s *LocalhostInterchainAccountsTestSuite) TestInterchainAccounts_Localhost() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	s.CreateDefaultPaths(testName)

	chainA, _ := s.GetChains()

	chainADenom := chainA.Config().Denom

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
		msgChanOpenTryRes  channeltypes.MsgChannelOpenTryResponse
		ack                []byte
		packet             channeltypes.Packet
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")

	version := icatypes.NewDefaultMetadataString(exported.LocalhostConnectionID, exported.LocalhostConnectionID)
	controllerPortID, err := icatypes.NewControllerPortID(userAWallet.FormattedAddress())
	s.Require().NoError(err)

	t.Run("channel open init localhost - broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(exported.LocalhostConnectionID, userAWallet.FormattedAddress(), version, channeltypes.ORDERED)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgRegisterAccount)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))
	})

	t.Run("channel open try localhost", func(t *testing.T) {
		msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
			icatypes.HostPortID, icatypes.Version,
			channeltypes.ORDERED, []string{exported.LocalhostConnectionID},
			controllerPortID, msgChanOpenInitRes.ChannelId,
			version, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenTry)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))
	})

	t.Run("channel open ack localhost", func(t *testing.T) {
		msgChanOpenAck := channeltypes.NewMsgChannelOpenAck(
			controllerPortID, msgChanOpenInitRes.ChannelId,
			msgChanOpenTryRes.ChannelId, msgChanOpenTryRes.Version,
			localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenAck)
		s.AssertTxSuccess(txResp)
	})

	t.Run("channel open confirm localhost", func(t *testing.T) {
		msgChanOpenConfirm := channeltypes.NewMsgChannelOpenConfirm(
			icatypes.HostPortID, msgChanOpenTryRes.ChannelId,
			localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenConfirm)
		s.AssertTxSuccess(txResp)
	})

	t.Run("query localhost interchain accounts channel ends", func(t *testing.T) {
		channelEndA, err := query.Channel(ctx, chainA, controllerPortID, msgChanOpenInitRes.ChannelId)
		s.Require().NoError(err)
		s.Require().NotNil(channelEndA)

		channelEndB, err := query.Channel(ctx, chainA, icatypes.HostPortID, msgChanOpenTryRes.ChannelId)
		s.Require().NoError(err)
		s.Require().NotNil(channelEndB)

		s.Require().Equal(channelEndA.ConnectionHops, channelEndB.ConnectionHops)
	})

	t.Run("verify interchain account registration and deposit funds", func(t *testing.T) {
		interchainAccAddress, err := query.InterchainAccount(ctx, chainA, userAWallet.FormattedAddress(), exported.LocalhostConnectionID)
		s.Require().NoError(err)
		s.Require().NotEmpty(interchainAccAddress)

		walletAmount := ibc.WalletAmount{
			Address: interchainAccAddress,
			Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
			Denom:   chainADenom,
		}

		s.Require().NoError(chainA.SendFunds(ctx, interchaintest.FaucetAccountKeyName, walletAmount))
	})

	t.Run("send packet localhost interchain accounts", func(t *testing.T) {
		interchainAccAddress, err := query.InterchainAccount(ctx, chainA, userAWallet.FormattedAddress(), exported.LocalhostConnectionID)
		s.Require().NoError(err)
		s.Require().NotEmpty(interchainAccAddress)

		msgSend := &banktypes.MsgSend{
			FromAddress: interchainAccAddress,
			ToAddress:   userBWallet.FormattedAddress(),
			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainADenom)),
		}

		cdc := testsuite.Codec()
		bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
		s.Require().NoError(err)

		packetData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: bz,
			Memo: "e2e",
		}

		msgSendTx := controllertypes.NewMsgSendTx(userAWallet.FormattedAddress(), exported.LocalhostConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSendTx)
		s.AssertTxSuccess(txResp)

		packet, err = ibctesting.ParsePacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
	})

	t.Run("recv packet localhost interchain accounts", func(t *testing.T) {
		msgRecvPacket := channeltypes.NewMsgRecvPacket(packet, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		ack, err = ibctesting.ParseAckFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ack)
	})

	t.Run("acknowledge packet localhost interchain accounts", func(t *testing.T) {
		msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
		s.AssertTxSuccess(txResp)
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		balance, err := query.Balance(ctx, chainA, userBWallet.FormattedAddress(), chainADenom)
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance.Int64())
	})
}

func (s *LocalhostInterchainAccountsTestSuite) TestInterchainAccounts_ReopenChannel_Localhost() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	s.CreateDefaultPaths(testName)

	chainA, _ := s.GetChains()
	chainADenom := chainA.Config().Denom

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	// Initial channel setup
	controllerPortID, msgChanOpenInitRes, msgChanOpenTryRes := s.setupInitialChannel(ctx, t, chainA, userAWallet, rlyWallet)

	// Verify initial channel state
	s.verifyInitialChannelState(ctx, t, chainA, controllerPortID, msgChanOpenInitRes, msgChanOpenTryRes)

	// Fund and verify interchain account
	interchainAccAddress := s.fundAndVerifyInterchainAccount(ctx, t, chainA, userAWallet, chainADenom)

	// Test packet timeout and channel closure
	s.testPacketTimeoutAndChannelClosure(ctx, t, chainA, userAWallet, userBWallet, rlyWallet, interchainAccAddress, chainADenom, controllerPortID, msgChanOpenInitRes, msgChanOpenTryRes)

	// Reopen channel and verify
	s.reopenChannelAndVerify(ctx, t, chainA, userAWallet, rlyWallet, controllerPortID, interchainAccAddress, chainADenom)

	// Test successful packet transfer
	s.testSuccessfulPacketTransfer(ctx, t, chainA, userAWallet, userBWallet, rlyWallet, interchainAccAddress, chainADenom, controllerPortID, msgChanOpenInitRes)
}

func (s *LocalhostInterchainAccountsTestSuite) setupInitialChannel(ctx context.Context, t *testing.T, chainA ibc.Chain, userAWallet, rlyWallet ibc.Wallet) (string, channeltypes.MsgChannelOpenInitResponse, channeltypes.MsgChannelOpenTryResponse) {
	t.Helper()
	var (
		msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
		msgChanOpenTryRes  channeltypes.MsgChannelOpenTryResponse
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")

	version := icatypes.NewDefaultMetadataString(exported.LocalhostConnectionID, exported.LocalhostConnectionID)
	controllerPortID, err := icatypes.NewControllerPortID(userAWallet.FormattedAddress())
	s.Require().NoError(err)

	// Channel open init
	msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(exported.LocalhostConnectionID, userAWallet.FormattedAddress(), version, channeltypes.ORDERED)
	txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgRegisterAccount)
	s.AssertTxSuccess(txResp)
	s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))

	// Channel open try
	msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
		icatypes.HostPortID, icatypes.Version,
		channeltypes.ORDERED, []string{exported.LocalhostConnectionID},
		controllerPortID, msgChanOpenInitRes.ChannelId,
		version, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
	)
	txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenTry)
	s.AssertTxSuccess(txResp)
	s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))

	return controllerPortID, msgChanOpenInitRes, msgChanOpenTryRes
}

func (s *LocalhostInterchainAccountsTestSuite) verifyInitialChannelState(ctx context.Context, t *testing.T, chainA ibc.Chain, controllerPortID string, msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse, msgChanOpenTryRes channeltypes.MsgChannelOpenTryResponse) {
	t.Helper()
	channelEndA, err := query.Channel(ctx, chainA, controllerPortID, msgChanOpenInitRes.ChannelId)
	s.Require().NoError(err)
	s.Require().NotNil(channelEndA)

	channelEndB, err := query.Channel(ctx, chainA, icatypes.HostPortID, msgChanOpenTryRes.ChannelId)
	s.Require().NoError(err)
	s.Require().NotNil(channelEndB)

	s.Require().Equal(channelEndA.ConnectionHops, channelEndB.ConnectionHops)
}

func (s *LocalhostInterchainAccountsTestSuite) fundAndVerifyInterchainAccount(ctx context.Context, t *testing.T, chainA ibc.Chain, userAWallet ibc.Wallet, chainADenom string) string {
	t.Helper()
	interchainAccAddress, err := query.InterchainAccount(ctx, chainA, userAWallet.FormattedAddress(), exported.LocalhostConnectionID)
	s.Require().NoError(err)
	s.Require().NotEmpty(interchainAccAddress)

	walletAmount := ibc.WalletAmount{
		Address: interchainAccAddress,
		Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
		Denom:   chainADenom,
	}

	s.Require().NoError(chainA.SendFunds(ctx, interchaintest.FaucetAccountKeyName, walletAmount))
	return interchainAccAddress
}

func (s *LocalhostInterchainAccountsTestSuite) testPacketTimeoutAndChannelClosure(ctx context.Context, t *testing.T, chainA ibc.Chain, userAWallet, userBWallet, rlyWallet ibc.Wallet, interchainAccAddress, chainADenom, controllerPortID string, msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse, msgChanOpenTryRes channeltypes.MsgChannelOpenTryResponse) {
	t.Helper()
	// Send packet with timeout
	msgSend := &banktypes.MsgSend{
		FromAddress: interchainAccAddress,
		ToAddress:   userBWallet.FormattedAddress(),
		Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainADenom)),
	}

	cdc := testsuite.Codec()
	bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
	s.Require().NoError(err)

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "e2e",
	}

	msgSendTx := controllertypes.NewMsgSendTx(userAWallet.FormattedAddress(), exported.LocalhostConnectionID, uint64(1), packetData)
	txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSendTx)
	s.AssertTxSuccess(txResp)

	packet, err := ibctesting.ParsePacketFromEvents(txResp.Events)
	s.Require().NoError(err)
	s.Require().NotNil(packet)

	// Timeout packet
	msgTimeout := channeltypes.NewMsgTimeout(packet, 1, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())
	txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgTimeout)
	s.AssertTxSuccess(txResp)

	// Close channel
	msgCloseConfirm := channeltypes.NewMsgChannelCloseConfirm(icatypes.HostPortID, msgChanOpenTryRes.ChannelId, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())
	txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgCloseConfirm)
	s.AssertTxSuccess(txResp)

	// Verify channel is closed
	channelEndA, err := query.Channel(ctx, chainA, controllerPortID, msgChanOpenInitRes.ChannelId)
	s.Require().NoError(err)
	s.Require().Equal(channeltypes.CLOSED, channelEndA.State)

	channelEndB, err := query.Channel(ctx, chainA, icatypes.HostPortID, msgChanOpenTryRes.ChannelId)
	s.Require().NoError(err)
	s.Require().Equal(channeltypes.CLOSED, channelEndB.State)
}

func (s *LocalhostInterchainAccountsTestSuite) reopenChannelAndVerify(ctx context.Context, t *testing.T, chainA ibc.Chain, userAWallet, rlyWallet ibc.Wallet, controllerPortID, interchainAccAddress, chainADenom string) {
	t.Helper()
	version := icatypes.NewDefaultMetadataString(exported.LocalhostConnectionID, exported.LocalhostConnectionID)

	// Register new interchain account
	msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(exported.LocalhostConnectionID, userAWallet.FormattedAddress(), version, channeltypes.ORDERED)
	txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgRegisterAccount)
	s.AssertTxSuccess(txResp)

	var msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
	s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))

	// Channel open try
	msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
		icatypes.HostPortID, icatypes.Version,
		channeltypes.ORDERED, []string{exported.LocalhostConnectionID},
		controllerPortID, msgChanOpenInitRes.ChannelId,
		version, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
	)
	txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenTry)
	s.AssertTxSuccess(txResp)

	var msgChanOpenTryRes channeltypes.MsgChannelOpenTryResponse
	s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))

	// Verify channel state
	channelEndA, err := query.Channel(ctx, chainA, controllerPortID, msgChanOpenInitRes.ChannelId)
	s.Require().NoError(err)
	s.Require().NotNil(channelEndA)

	channelEndB, err := query.Channel(ctx, chainA, icatypes.HostPortID, msgChanOpenTryRes.ChannelId)
	s.Require().NoError(err)
	s.Require().NotNil(channelEndB)

	s.Require().Equal(channelEndA.ConnectionHops, channelEndB.ConnectionHops)

	// Verify interchain account balance
	balance, err := query.Balance(ctx, chainA, interchainAccAddress, chainADenom)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.StartingTokenAmount, balance.Int64())
}

func (s *LocalhostInterchainAccountsTestSuite) testSuccessfulPacketTransfer(ctx context.Context, t *testing.T, chainA ibc.Chain, userAWallet, userBWallet, rlyWallet ibc.Wallet, interchainAccAddress, chainADenom, controllerPortID string, msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse) {
	t.Helper()
	msgSend := &banktypes.MsgSend{
		FromAddress: interchainAccAddress,
		ToAddress:   userBWallet.FormattedAddress(),
		Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainADenom)),
	}

	cdc := testsuite.Codec()
	bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
	s.Require().NoError(err)

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "e2e",
	}

	msgSendTx := controllertypes.NewMsgSendTx(userAWallet.FormattedAddress(), exported.LocalhostConnectionID, uint64(time.Hour.Nanoseconds()), packetData)
	txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSendTx)
	s.AssertTxSuccess(txResp)

	packet, err := ibctesting.ParsePacketFromEvents(txResp.Events)
	s.Require().NoError(err)
	s.Require().NotNil(packet)

	// Recv packet
	msgRecvPacket := channeltypes.NewMsgRecvPacket(packet, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())
	txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
	s.AssertTxSuccess(txResp)

	ack, err := ibctesting.ParseAckFromEvents(txResp.Events)
	s.Require().NoError(err)
	s.Require().NotNil(ack)

	// Acknowledge packet
	msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())
	txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
	s.AssertTxSuccess(txResp)

	// Verify tokens transferred
	s.AssertPacketRelayed(ctx, chainA, controllerPortID, msgChanOpenInitRes.ChannelId, 1)

	balance, err := query.Balance(ctx, chainA, userBWallet.FormattedAddress(), chainADenom)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.IBCTransferAmount+testvalues.StartingTokenAmount, balance.Int64())
}
