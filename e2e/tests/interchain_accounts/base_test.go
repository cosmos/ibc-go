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
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestInterchainAccountsTestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsTestSuite))
}

type InterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
}

// RegisterInterchainAccount will attempt to register an interchain account on the counterparty chain.
func (s *InterchainAccountsTestSuite) RegisterInterchainAccount(ctx context.Context, chain ibc.Chain, user ibc.Wallet, msgRegisterAccount *controllertypes.MsgRegisterInterchainAccount) {
	txResp := s.BroadcastMessages(ctx, chain, user, msgRegisterAccount)
	s.AssertTxSuccess(txResp)
}

func (s *InterchainAccountsTestSuite) TestMsgSendTx_SuccessfulTransfer() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx, nil)
	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.FormattedAddress()
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)

		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.AssertTxSuccess(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("interchain account executes a bank transfer on behalf of the corresponding owner account", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account so it has some $$ to send
			err := chainB.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: hostAccount,
				Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
				Denom:   chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("broadcast MsgSendTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.FormattedAddress(),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAccount.FormattedAddress(), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			resp := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSendTx,
			)

			s.AssertTxSuccess(resp)

			s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
		})

		t.Run("verify tokens transferred", func(t *testing.T) {
			balance, err := s.QueryBalance(ctx, chainB, chainBAccount.FormattedAddress(), chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = s.QueryBalance(ctx, chainB, hostAccount, chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance.Int64())
		})
	})
}

func (s *InterchainAccountsTestSuite) TestMsgSendTx_FailedTransfer_InsufficientFunds() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx, nil)
	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.FormattedAddress()
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)

		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.AssertTxSuccess(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("fail to execute bank transfer over ICA", func(t *testing.T) {
		t.Run("verify empty host wallet", func(t *testing.T) {
			hostAccountBalance, err := s.QueryBalance(ctx, chainB, hostAccount, chainB.Config().Denom)

			s.Require().NoError(err)
			s.Require().Zero(hostAccountBalance.Int64())
		})

		t.Run("broadcast MsgSendTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.FormattedAddress(),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			txResp := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSendTx,
			)

			s.AssertTxSuccess(txResp)

			s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
		})

		t.Run("verify balance is the same", func(t *testing.T) {
			balance, err := s.QueryBalance(ctx, chainB, chainBAccount.FormattedAddress(), chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance.Int64())
		})
	})
}

func (s *InterchainAccountsTestSuite) TestMsgSendTx_SuccessfulTransfer_AfterReopeningICA() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx, nil)
	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.FormattedAddress()
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	var (
		portID      string
		hostAccount string

		initialChannelID        = "channel-1"
		channelIDAfterReopening = "channel-2"
	)

	t.Run("register interchain account", func(t *testing.T) {
		var err error
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterInterchainAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)
		s.RegisterInterchainAccount(ctx, chainA, controllerAccount, msgRegisterInterchainAccount)
		portID, err = icatypes.NewControllerPortID(controllerAddress)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		_, err = s.QueryChannel(ctx, chainA, portID, initialChannelID)
		s.Require().NoError(err)
	})

	// stop the relayer to let the submit tx message time out
	t.Run("stop relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
	})

	t.Run("submit tx message with bank transfer message times out", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account account so it has some $$ to send
			err := chainB.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: hostAccount,
				Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
				Denom:   chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("broadcast MsgSendTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.FormattedAddress(),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			cdc := testsuite.Codec()

			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, uint64(1), packetData)

			resp := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSendTx,
			)

			s.AssertTxSuccess(resp)

			// this sleep is to allow the packet to timeout
			time.Sleep(1 * time.Second)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify channel is closed due to timeout on ordered channel", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainA, portID, initialChannelID)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.CLOSED, channel.State, "the channel was not in an expected state")
	})

	t.Run("verify tokens not transferred", func(t *testing.T) {
		balance, err := s.QueryBalance(ctx, chainB, chainBAccount.FormattedAddress(), chainB.Config().Denom)
		s.Require().NoError(err)

		_, err = s.QueryBalance(ctx, chainB, hostAccount, chainB.Config().Denom)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance.Int64())
	})

	// re-register interchain account to reopen the channel now that it has been closed due to timeout
	// on an ordered channel
	t.Run("register interchain account", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterInterchainAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)
		s.RegisterInterchainAccount(ctx, chainA, controllerAccount, msgRegisterInterchainAccount)

		s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
	})

	t.Run("verify new channel is now open and interchain account has been reregistered with the same portID", func(t *testing.T) {
		channel, err := s.QueryChannel(ctx, chainA, portID, channelIDAfterReopening)
		s.Require().NoError(err)

		s.Require().Equal(channeltypes.OPEN, channel.State, "the channel was not in an expected state")
	})

	t.Run("broadcast MsgSendTx", func(t *testing.T) {
		// assemble bank transfer message from host account to user account on host chain
		msgSend := &banktypes.MsgSend{
			FromAddress: hostAccount,
			ToAddress:   chainBAccount.FormattedAddress(),
			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
		}

		cdc := testsuite.Codec()

		bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
		s.Require().NoError(err)

		packetData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: bz,
			Memo: "e2e",
		}

		msgSendTx := controllertypes.NewMsgSendTx(controllerAddress, ibctesting.FirstConnectionID, uint64(5*time.Minute), packetData)

		resp := s.BroadcastMessages(
			ctx,
			chainA,
			controllerAccount,
			msgSendTx,
		)

		s.AssertTxSuccess(resp)

		// time for the packet to be relayed
		s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB))
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		balance, err := s.QueryBalance(ctx, chainB, chainBAccount.FormattedAddress(), chainB.Config().Denom)
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance.Int64())
	})
}
