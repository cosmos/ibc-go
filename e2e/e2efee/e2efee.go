package e2efee

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/ibc-go/v3/e2e/dockerutil"
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
)

type FeeMiddlewareChain struct {
	*cosmos.CosmosChain
}

func (fc *FeeMiddlewareChain) RecoverKeyring(ctx context.Context, name, mnemonic string) error {
	tn := fc.ChainNodes[0]

	cmd := []string{
		"bash",
		"-c",
		fmt.Sprintf(`echo "%s" | %s keys add %s --recover --keyring-backend %s --home %s`, mnemonic, fc.Config().Bin, name, keyring.BackendTest, tn.NodeHome()),
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}

	return nil

}

func (fc *FeeMiddlewareChain) RegisterCounterPartyPayee(ctx context.Context, relayerAddress, counterPartyPayee, portId, channelId string) error {
	tn := fc.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"tx",
		"ibc-fee",
		"register-counterparty-payee",
		portId,
		channelId,
		relayerAddress,
		counterPartyPayee,
		"--from", relayerAddress,
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

func (fc *FeeMiddlewareChain) QueryPackets(ctx context.Context, portId, channelId string) (types.QueryIncentivizedPacketsResponse, error) {
	tn := fc.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"q",
		"ibc-fee",
		"packets-for-channel",
		portId,
		channelId,
		"--home", tn.NodeHome(),
		"--node", fmt.Sprintf("tcp://%s:26657", tn.HostName()),
		"--output", "json",
		"--chain-id", fc.Config().ChainID,
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return types.QueryIncentivizedPacketsResponse{}, dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}

	respBytes := []byte(stdout)
	response := types.QueryIncentivizedPacketsResponse{}
	//if err := types.ModuleCdc.Unmarshal(respBytes, &response); err != nil {
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return types.QueryIncentivizedPacketsResponse{}, err
	}

	return response, nil

}

func (fc *FeeMiddlewareChain) QueryCounterPartyPayee(ctx context.Context, relayerAddress, channelID string) (string, error) {
	tn := fc.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"q",
		"ibc-fee",
		"counterparty-payee",
		channelID,
		relayerAddress,
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

func (fc *FeeMiddlewareChain) PayPacketFee(ctx context.Context, fromAddress, portId, channelId string, sequenceNumber, recvFee, ackFee, timeoutFee int64) error {
	tn := fc.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"tx",
		"ibc-fee",
		"pay-packet-fee",
		portId,
		channelId,
		fmt.Sprintf("%d", sequenceNumber),
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
