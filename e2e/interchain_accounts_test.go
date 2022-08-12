package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

func init() {
	// TODO: remove these, they should be set external to the tests.
	os.Setenv("CHAIN_A_SIMD_IMAGE", "ghcr.io/cosmos/ibc-go-icad")
	os.Setenv("CHAIN_A_SIMD_TAG", "v0.3.0")
	os.Setenv("CHAIN_B_SIMD_IMAGE", "ghcr.io/cosmos/ibc-go-icad")
	os.Setenv("CHAIN_B_SIMD_TAG", "v0.3.0")
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
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	_ = chainB
	_ = relayer
	_ = channelA

	connectionId := "connection-0"
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	s.Run("register interchain account", func() {
		account, err := s.RegisterICA(ctx, chainA, chainAWallet.KeyName, connectionId)
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
		interchainAccountAddress, err = s.QueryICA(ctx, chainA, connectionId, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix))
		s.Require().NoError(err)
		s.Require().NotEmpty(interchainAccountAddress)
	})
}

// TODO: replace the below methods with transaction broadcasts.

type IBCTransferTx struct {
	TxHash string `json:"txhash"`
}

// RegisterICA will attempt to register an interchain account on the counterparty chain.
func (*InterchainAccountsTestSuite) RegisterICA(ctx context.Context, chain *cosmos.CosmosChain, address, connectionID string) (string, error) {
	config := chain.Config()
	node := chain.ChainNodes[0]
	command := []string{config.Bin, "tx", "intertx", "register",
		"--from", address,
		"--connection-id", connectionID,
		"--chain-id", config.ChainID,
		"--home", node.HomeDir(),
		"--node", fmt.Sprintf("tcp://%s:26657", node.HostName()),
		"--keyring-backend", keyring.BackendTest,
		"-y",
	}

	stdout, _, err := node.Exec(ctx, command, nil)
	if err != nil {
		return "", err
	}
	output := IBCTransferTx{}
	err = yaml.Unmarshal(stdout, &output)
	if err != nil {
		return "", err
	}
	return output.TxHash, nil
}

// QueryICA will query for an interchain account controlled by the specified address on the counterparty chain.
func (*InterchainAccountsTestSuite) QueryICA(ctx context.Context, chain *cosmos.CosmosChain, connectionID, address string) (string, error) {
	config := chain.Config()
	node := chain.ChainNodes[0]
	command := []string{config.Bin, "query", "intertx", "interchainaccounts", connectionID, address,
		"--chain-id", node.Chain.Config().ChainID,
		"--home", node.HomeDir(),
		"--node", fmt.Sprintf("tcp://%s:26657", node.HostName())}

	stdout, _, err := node.Exec(ctx, command, nil)
	if err != nil {
		return "", err
	}

	// at this point stdout should look like this:
	// interchain_account_address: cosmos1p76n3mnanllea4d3av0v0e42tjj03cae06xq8fwn9at587rqp23qvxsv0j
	// we split the string at the : and then just grab the address before returning.
	parts := strings.SplitN(string(stdout), ":", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("malformed stdout from command: %s", stdout)
	}
	return strings.TrimSpace(parts[1]), nil
}

// SendICABankTransfer builds a bank transfer message for a specified address and sends it to the specified
// interchain account.
func (*InterchainAccountsTestSuite) SendICABankTransfer(ctx context.Context, chain *cosmos.CosmosChain, connectionID, fromAddr string, amount ibc.WalletAmount) error {
	config := chain.Config()
	node := chain.ChainNodes[0]
	msg, err := json.Marshal(map[string]any{
		"@type":        "/cosmos.bank.v1beta1.MsgSend",
		"from_address": fromAddr,
		"to_address":   amount.Address,
		"amount": []map[string]any{
			{
				"denom":  amount.Denom,
				"amount": amount.Amount,
			},
		},
	})
	if err != nil {
		return err
	}

	command := []string{config.Bin, "tx", "intertx", "submit", string(msg),
		"--connection-id", connectionID,
		"--from", fromAddr,
		"--chain-id", node.Chain.Config().ChainID,
		"--home", node.HomeDir(),
		"--node", fmt.Sprintf("tcp://%s:26657", node.HostName()),
		"--keyring-backend", keyring.BackendTest,
		"-y",
	}

	_, _, err = node.Exec(ctx, command, nil)
	return err
}
