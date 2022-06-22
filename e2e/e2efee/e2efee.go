package e2efee

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/ibc-go/v3/e2e/dockerutil"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"strings"
)

type FeeMiddlewareChain struct {
	*cosmos.CosmosChain
}

func (fc *FeeMiddlewareChain) RegisterCounterPartyPayee(ctx context.Context, chain1Address, chain2Address string) error {
	tn := fc.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"tx",
		"ibc-fee",
		"register-counterparty-payee",
		"transfer",
		"channel-0",
		strings.TrimSpace(chain2Address),
		strings.TrimSpace(chain1Address),
		"--from", strings.TrimSpace(chain2Address),
		"--keyring-backend", keyring.BackendTest,
		"--home", tn.NodeHome(),
		"--node", fmt.Sprintf("tcp://%s:26657", tn.HostName()),
		"--output", "json",
		"--chain-id", fc.Config().ChainID,
		"--yes",
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}

	return nil

}

func (fc *FeeMiddlewareChain) QueryPackets(ctx context.Context) error {
	tn := fc.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"q",
		"ibc-fee",
		"packets-for-channel",
		"transfer",
		"channel-0",
		"--home", tn.NodeHome(),
		"--node", fmt.Sprintf("tcp://%s:26657", tn.HostName()),
		"--output", "json",
		"--chain-id", fc.Config().ChainID,
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}

	return nil

}

func (fc *FeeMiddlewareChain) QueryCounterPartyPayee(ctx context.Context, chain2Address string) (string, error) {
	tn := fc.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"q",
		"ibc-fee",
		"counterparty-payee",
		"channel-0",
		chain2Address,
		"--home", tn.NodeHome(),
		"--node", fmt.Sprintf("tcp://%s:26657", tn.HostName()),
		"--output", "json",
		"--chain-id", fc.Config().ChainID,
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return "", dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}

	type QueryOutput struct {
		CounterPartyPayee string `json:"counterparty_payee"`
	}

	stdOutBytes := []byte(stdout)
	res := &QueryOutput{}
	if err := json.Unmarshal(stdOutBytes, res); err != nil {
		return "", err
	}

	return res.CounterPartyPayee, nil
}

func (fc *FeeMiddlewareChain) PayPacketFee(ctx context.Context, fromAddress string, recvFee, ackFee, timeoutFee int64) error {
	tn := fc.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"tx",
		"ibc-fee",
		"pay-packet-fee",
		"transfer",
		"channel-0",
		"1",
		"--from", fromAddress,
		"--recv-fee", fmt.Sprintf("%d%s", recvFee, fc.Config().Denom),
		"--ack-fee", fmt.Sprintf("%d%s", ackFee, fc.Config().Denom),
		"--timeout-fee", fmt.Sprintf("%d%s", timeoutFee, fc.Config().Denom),
		"--keyring-backend", keyring.BackendTest,
		"--home", tn.NodeHome(),
		"--node", fmt.Sprintf("tcp://%s:26657", tn.HostName()),
		"--output", "json",
		"--chain-id", fc.Config().ChainID,
		"--yes",
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}

	return nil
}
