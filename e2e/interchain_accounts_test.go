package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	ibctest "github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v5/modules/apps/transfer/types"
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

	_ = chainB
	_ = relayer
	_ = channelA

	connectionId := "connection-0"
	controllerWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, 0)
	var hostAccount string

	t.Run("register interchain account", func(t *testing.T) {
		_, err := s.RegisterICA(ctx, chainA, controllerWallet.KeyName, connectionId)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		hostAccount, err := s.QueryICA(ctx, chainA, connectionId, controllerWallet.Bech32Address(chainA.Config().Bech32Prefix))
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))
	})

	// TODO: RegisterICA should return account addr
	// TODO: change bech32 prefix so both are not the same

	t.Run("send successful bank transfer from controller account to host account", func(t *testing.T) {

		// fund the host account wallet so it has some $$ to send
		err := chainB.SendFunds(ctx, ibctest.FaucetAccountKeyName, ibc.WalletAmount{
			Address: hostAccount,
			Amount:  testvalues.StartingTokenAmount,
			Denom:   chainB.Config().Denom,
		})
		s.Require().NoError(err)

		resp, err := s.BroadcastMessages(ctx, chainA, controllerWallet, &banktypes.MsgSend{
			FromAddress: controllerWallet.Bech32Address(chainA.Config().Bech32Prefix),
			ToAddress:   chainBWallet.Bech32Address(chainB.Config().Bech32Prefix),
			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainA.Config().Denom)),
		})

		// s.AssertValidTxResponse(resp)
		// s.Require().NoError(err)

		// balance, err := chainB.GetBalance(ctx, chainBWallet.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
		// s.Require().NoError(err)

		// expected := testvalues.IBCTransferAmount
		// s.Require().Equal(expected, balance)
	})
}

// TODO: replace the below methods with transaction broadcasts.
// TODO: utility function wrapping Get&FundTestUsers

type IBCTransferTx struct {
	TxHash string `json:"txhash"`
}

// getIBCToken returns the denomination of the full token denom sent to the receiving channel
func getIBCToken(fullTokenDenom string, portID, channelID string) transfertypes.DenomTrace {
	return transfertypes.ParseDenomTrace(fmt.Sprintf("%s/%s/%s", portID, channelID, fullTokenDenom))
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
func (*InterchainAccountsTestSuite) SendICABankTransfer(ctx context.Context, chain *cosmos.CosmosChain, connectionID string, amount ibc.WalletAmount, toAddr string) error {
	config := chain.Config()
	node := chain.ChainNodes[0]

	msg, err := json.Marshal(map[string]any{
		"@type":        "/cosmos.bank.v1beta1.MsgSend",
		"from_address": amount.Address,
		"to_address":   toAddr,
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

	command := []string{
		config.Bin, "tx", "intertx", "submit", string(msg),
		"--connection-id", connectionID,
		"--from", amount.Address,
		"--chain-id", node.Chain.Config().ChainID,
		"--home", node.HomeDir(),
		"--node", fmt.Sprintf("tcp://%s:26657", node.HostName()),
		"--keyring-backend", keyring.BackendTest,
		"-y",
	}

	_, _, err = node.Exec(ctx, command, nil)
	return err
}
