package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

func TestInterchainAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsTestSuite))
}

type InterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *InterchainAccountsTestSuite) TestInterchainAccounts() {
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	_ = chainB
	_ = relayer
	_ = channelA

	connectionId := "connection-0"
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	s.Run("register interchain account", func() {
		account, err := chainA.RegisterInterchainAccount(ctx, chainAWallet.KeyName, connectionId)
		s.Require().NoError(err)
		s.Require().NotEmpty(account)
		s.T().Logf("account created: %s", account)
	})

	s.Run("start relayer", func() {
		s.StartRelayer(relayer)
	})

	var (
		interchainAccountAddress string
	)

	s.Run("verify interchain account", func() {
		var err error
		interchainAccountAddress, err = chainA.QueryInterchainAccount(ctx, connectionId, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix))
		s.Require().NoError(err)
		s.Require().NotEmpty(interchainAccountAddress)
	})
}
