package e2e

import (
	"context"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

func init() {
	// TODO: remove these, they should be set external to the tests.
	os.Setenv("CHAIN_A_SIMD_IMAGE", "ghcr.io/cosmos/ibc-go-icad")
	os.Setenv("CHAIN_A_SIMD_TAG", "v0.3.0")
	os.Setenv("CHAIN_B_SIMD_IMAGE", "ghcr.io/cosmos/ibc-go-icad")
	os.Setenv("CHAIN_B_SIMD_TAG", "v0.2.0")
	os.Setenv("CHAIN_A_BINARY", "icad")
	os.Setenv("CHAIN_B_BINARY", "icad")
}

func TestInterchainAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsTestSuite))
}

type InterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
}

// cd e2e
// make e2e-test test=TestInterchainAccounts suite=InterchainAccountsTestSuite
func (s *InterchainAccountsTestSuite) TestInterchainAccounts() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	_ = channelA

	connectionId := "connection-0"
	controllerWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	t.Run("register interchain account", func(t *testing.T) {
		err := s.RegisterICA(ctx, chainA, controllerWallet, controllerWallet.Bech32Address(chainA.Config().Bech32Prefix), connectionId)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	var (
		interchainAccountAddress string
	)

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		interchainAccountAddress, err = s.QueryInterchainAccount(ctx, chainA, controllerWallet.Bech32Address(chainA.Config().Bech32Prefix), connectionId)
		s.Require().NoError(err)
		s.Require().NotEmpty(interchainAccountAddress)
	})

	// TODO: change bech32 prefix so both are not the same

	t.Run("send successful bank transfer from controller account to host account", func(t *testing.T) {
		resp, err := s.BroadcastMessages(ctx, chainA, controllerWallet, &banktypes.MsgSend{
			FromAddress: controllerWallet.Bech32Address(chainA.Config().Bech32Prefix),
			ToAddress:   chainBWallet.Bech32Address(chainB.Config().Bech32Prefix),
			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainA.Config().Denom)),
		})

		s.AssertValidTxResponse(resp)
		s.Require().NoError(err)
	})
}

// RegisterICA will attempt to register an interchain account on the counterparty chain.
func (s *InterchainAccountsTestSuite) RegisterICA(ctx context.Context, chain *cosmos.CosmosChain, user *ibctest.User, fromAddress, connectionID string) error {
	msg := intertxtypes.NewMsgRegisterAccount(fromAddress, connectionID, "")
	txResp, err := s.BroadcastMessages(ctx, chain, user, msg)
	s.AssertValidTxResponse(txResp)
	return err
}
