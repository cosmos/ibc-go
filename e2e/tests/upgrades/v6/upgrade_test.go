package v6

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"
	ibctest "github.com/strangelove-ventures/ibctest/v6"
	"github.com/strangelove-ventures/ibctest/v6/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"
	"golang.org/x/mod/semver"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

const (
	blocksAfterUpgrade = uint64(10)
	haltHeight         = uint64(100)

	prevVersion    = "v0.3.5"
	upgradeVersion = "v0.4.0"
)

func TestUpgradeV6TestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeV6TestSuite))
}

type UpgradeV6TestSuite struct {
	testsuite.E2ETestSuite
}

// UpgradeChain upgrades a chain to a specific version using the planName provided.
// The software upgrade proposal is broadcast by the provided wallet.
// TODO: Consider taking prevVersion as arg or the upgradePlan
func (s *UpgradeV6TestSuite) UpgradeChain(ctx context.Context, chain *cosmos.CosmosChain, wallet *ibc.Wallet, planName, prevVersion, upgradeVersion string) {
	plan := upgradetypes.Plan{
		Name:   planName,
		Height: int64(haltHeight),
		Info:   fmt.Sprintf("upgrade version test from %s to %s", prevVersion, upgradeVersion),
	}

	upgradeProposal := upgradetypes.NewSoftwareUpgradeProposal(fmt.Sprintf("upgrade from %s to %s", prevVersion, upgradeVersion), "upgrade chain E2E test", plan)
	s.ExecuteGovProposal(ctx, chain, wallet, upgradeProposal)

	height, err := chain.Height(ctx)
	s.Require().NoError(err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chain)
	s.Require().Error(err, "chain did not halt at halt height")

	err = chain.StopAllNodes(ctx)
	s.Require().NoError(err, "error stopping node(s)")

	chain.UpgradeVersion(ctx, s.DockerClient, upgradeVersion)

	err = chain.StartAllNodes(ctx)
	s.Require().NoError(err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	s.Require().NoError(err, "chain did not produce blocks after upgrade")

	height, err = chain.Height(ctx)
	s.Require().NoError(err, "error fetching height after upgrade")

	s.Require().GreaterOrEqual(height, haltHeight+blocksAfterUpgrade, "height did not increment enough after upgrade")
}

func (s *UpgradeV6TestSuite) TestV5ToV6ChainUpgrade() {
	t := s.T()

	s.Require().NoError(os.Setenv(testconfig.ChainATagEnv, prevVersion))
	s.Require().NoError(os.Setenv(testconfig.ChainBTagEnv, prevVersion))

	ctx := context.Background()
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	// create separate user specifically for the upgrade proposal to more easily verify starting
	// and end balances of the chainA users.
	chainAUpgradeProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	// chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	// chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	// chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	// chainBAddress := chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("register interchain account", func(t *testing.T) {
		version := getICAVersion(testconfig.GetChainATag(), testconfig.GetChainBTag())
		msgRegisterAccount := intertxtypes.NewMsgRegisterAccount(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, version)
		err := s.RegisterInterchainAccount(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.Require().NoError(err)
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

		t.Run("broadcast MsgSubmitTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.Bech32Address(chainB.Config().Bech32Prefix),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			// assemble submitMessage tx for intertx
			msgSubmitTx, err := intertxtypes.NewMsgSubmitTx(
				msgSend,
				ibctesting.FirstConnectionID,
				controllerAccount.Bech32Address(chainA.Config().Bech32Prefix),
			)
			s.Require().NoError(err)

			// broadcast submitMessage tx from controller account on chain A
			// this message should trigger the sending of an ICA packet over channel-1 (channel created between controller and host)
			// this ICA packet contains the assembled bank transfer message from above, which will be executed by the host account on the host chain.
			resp, err := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSubmitTx,
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

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("upgrade chainA", func(t *testing.T) {
		s.UpgradeChain(ctx, chainA, chainAUpgradeProposalWallet, "ibc-go/v6", prevVersion, upgradeVersion)
	})

	t.Run("restart relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
		s.StartRelayer(relayer)
	})

	t.Run("broadcast MsgSubmitTx", func(t *testing.T) {
		// assemble bank transfer message from host account to user account on host chain
		msgSend := &banktypes.MsgSend{
			FromAddress: hostAccount,
			ToAddress:   chainBAccount.Bech32Address(chainB.Config().Bech32Prefix),
			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
		}

		// assemble submitMessage tx for intertx
		msgSubmitTx, err := intertxtypes.NewMsgSubmitTx(
			msgSend,
			ibctesting.FirstConnectionID,
			controllerAccount.Bech32Address(chainA.Config().Bech32Prefix),
		)
		s.Require().NoError(err)

		// broadcast submitMessage tx from controller account on chain A
		// this message should trigger the sending of an ICA packet over channel-1 (channel created between controller and host)
		// this ICA packet contains the assembled bank transfer message from above, which will be executed by the host account on the host chain.
		resp, err := s.BroadcastMessages(
			ctx,
			chainA,
			controllerAccount,
			msgSubmitTx,
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

		expected := (testvalues.IBCTransferAmount * 2) + testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance)
	})
}

// RegisterInterchainAccount will attempt to register an interchain account on the counterparty chain.
func (s *UpgradeV6TestSuite) RegisterInterchainAccount(ctx context.Context, chain *cosmos.CosmosChain, user *ibc.Wallet, msgRegisterAccount *intertxtypes.MsgRegisterAccount) error {
	txResp, err := s.BroadcastMessages(ctx, chain, user, msgRegisterAccount)
	s.Require().NoError(err)
	s.AssertValidTxResponse(txResp)
	return err
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
