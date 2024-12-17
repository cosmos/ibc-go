//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v9"
	"github.com/strangelove-ventures/interchaintest/v9/ibc"
	test "github.com/strangelove-ventures/interchaintest/v9/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	banktypes "cosmossdk.io/x/bank/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	localhost "github.com/cosmos/ibc-go/v9/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func TestInterchainAccountsLocalhostTestSuite(t *testing.T) {
	testifysuite.Run(t, new(LocalhostInterchainAccountsTestSuite))
}

// compatibility:from_version: v7.4.0
type LocalhostInterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
}

// compatibility:TestInterchainAccounts_Localhost:from_versions: v7.4.0,v7.5.0,v7.6.0,v7.7.0,v7.8.0,v8.4.0,v8.5.0,v9.0.0
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

	t.Run("send localhost interchain accounts packet with timeout", func(t *testing.T) {
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

		msgSendTx := controllertypes.NewMsgSendTx(userAWallet.FormattedAddress(), exported.LocalhostConnectionID, uint64(1), packetData)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSendTx)
		s.AssertTxSuccess(txResp)

		packet, err = ibctesting.ParsePacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
	})

	t.Run("timeout localhost interchain accounts packet", func(t *testing.T) {
		msgTimeout := channeltypes.NewMsgTimeout(packet, 1, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgTimeout)
		s.AssertTxSuccess(txResp)
	})

	t.Run("close interchain accounts host channel end", func(t *testing.T) {
		// Pass in zero for counterpartyUpgradeSequence given that channel has not undergone any upgrades.
		msgCloseConfirm := channeltypes.NewMsgChannelCloseConfirm(icatypes.HostPortID, msgChanOpenTryRes.ChannelId, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(), 0)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgCloseConfirm)
		s.AssertTxSuccess(txResp)
	})

	t.Run("verify localhost interchain accounts channel is closed", func(t *testing.T) {
		channelEndA, err := query.Channel(ctx, chainA, controllerPortID, msgChanOpenInitRes.ChannelId)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.CLOSED, channelEndA.State, "the channel was not in an expected state")

		channelEndB, err := query.Channel(ctx, chainA, icatypes.HostPortID, msgChanOpenTryRes.ChannelId)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.CLOSED, channelEndB.State, "the channel was not in an expected state")
	})

	t.Run("channel open init localhost: create new channel for existing account", func(t *testing.T) {
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(exported.LocalhostConnectionID, userAWallet.FormattedAddress(), version, channeltypes.ORDERED)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgRegisterAccount)
		s.AssertTxSuccess(txResp)

		// note: response values are updated here in msgChanOpenInitRes
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

		// note: response values are updated here in msgChanOpenTryRes
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

	t.Run("verify interchain account and existing balance", func(t *testing.T) {
		interchainAccAddress, err := query.InterchainAccount(ctx, chainA, userAWallet.FormattedAddress(), exported.LocalhostConnectionID)
		s.Require().NoError(err)
		s.Require().NotEmpty(interchainAccAddress)

		balance, err := query.Balance(ctx, chainA, interchainAccAddress, chainADenom)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance.Int64())
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
		s.AssertPacketRelayed(ctx, chainA, controllerPortID, msgChanOpenInitRes.ChannelId, 1)

		balance, err := query.Balance(ctx, chainA, userBWallet.FormattedAddress(), chainADenom)
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance.Int64())
	})
}
