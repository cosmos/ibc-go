package interchain_accounts

import (
	"context"
	"testing"
	"time"

	ibctest "github.com/strangelove-ventures/ibctest/v6"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"
	"golang.org/x/mod/semver"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"

	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	simappparams "github.com/cosmos/ibc-go/v6/testing/simapp/params"
)

func TestInterchainAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsTestSuite))
}

type InterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
}

// getICAVersion returns the version which should be used in the MsgRegisterAccount broadcast from the
// controller chain.
func getICAVersion(chainAVersion, chainBVersion string) string {
	chainBIsGreaterThanOrEqualToChainA := semver.Compare(chainAVersion, chainBVersion) <= 0
	if chainBIsGreaterThanOrEqualToChainA {
		// allow version to be specified by the controller chain
		return ""
	}
	// explicitly set the version string because the host chain might not yet support incentivized channels.
	return icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
}

func (s *InterchainAccountsTestSuite) TestMsgSendTx_SuccessfulTransfer() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		version := getICAVersion(testconfig.GetChainATag(), testconfig.GetChainBTag())
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), version)

		txResp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("interchain account executes a bank transfer on behalf of the corresponding owner account", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account account so it has some $$ to send
			err := chainB.SendFunds(ctx, ibctest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: hostAccount,
				Amount:  testvalues.StartingTokenAmount,
				Denom:   chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("broadcast MsgSendTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.Bech32Address(chainB.Config().Bech32Prefix),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			cfg := simappparams.MakeTestEncodingConfig()
			banktypes.RegisterInterfaces(cfg.InterfaceRegistry)
			cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)

			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend})
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			resp, err := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSendTx,
			)

			s.AssertValidTxResponse(resp)
			s.Require().NoError(err)

			s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
		})

		t.Run("verify tokens transferred", func(t *testing.T) {
			balance, err := chainB.GetBalance(ctx, chainBAccount.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = chainB.GetBalance(ctx, hostAccount, chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance)
		})
	})
}

func (s *InterchainAccountsTestSuite) TestMsgSendTx_FailedTransfer_InsufficientFunds() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		version := getICAVersion(testconfig.GetChainATag(), testconfig.GetChainBTag())
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), version)

		txResp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("fail to execute bank transfer over ICA", func(t *testing.T) {
		t.Run("verify empty host wallet", func(t *testing.T) {
			hostAccountBalance, err := chainB.GetBalance(ctx, hostAccount, chainB.Config().Denom)
			s.Require().NoError(err)
			s.Require().Zero(hostAccountBalance)
		})

		t.Run("broadcast MsgSendTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.Bech32Address(chainB.Config().Bech32Prefix),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			cfg := simappparams.MakeTestEncodingConfig()
			banktypes.RegisterInterfaces(cfg.InterfaceRegistry)
			cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)

			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend})
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			txResp, err := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSendTx,
			)

			s.AssertValidTxResponse(txResp)
			s.Require().NoError(err)

			s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
		})

		t.Run("verify balance is the same", func(t *testing.T) {
			balance, err := chainB.GetBalance(ctx, chainBAccount.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance)
		})
	})
}
