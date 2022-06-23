package e2efee

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/ibc-go/v3/e2e/dockerutil"
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
)

func RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain, relayerAddress, counterPartyPayee, portId, channelId string) error {
	tn := chain.ChainNodes[0]
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
		"--chain-id", chain.Config().ChainID,
		"--yes",
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}

	return nil

}

func QueryPackets(ctx context.Context, chain *cosmos.CosmosChain, portId, channelId string) (types.QueryIncentivizedPacketsResponse, error) {
	tn := chain.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"q",
		"ibc-fee",
		"packets-for-channel",
		portId,
		channelId,
		"--home", tn.NodeHome(),
		"--node", fmt.Sprintf("tcp://%s:26657", tn.HostName()),
		"--output", "json",
		"--chain-id", chain.Config().ChainID,
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

func QueryCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain, relayerAddress, channelID string) (string, error) {
	tn := chain.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"q",
		"ibc-fee",
		"counterparty-payee",
		channelID,
		relayerAddress,
		"--home", tn.NodeHome(),
		"--node", fmt.Sprintf("tcp://%s:26657", tn.HostName()),
		"--output", "json",
		"--chain-id", chain.Config().ChainID,
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

func PayPacketFee(ctx context.Context, chain *cosmos.CosmosChain, fromAddress, portId, channelId string, sequenceNumber, recvFee, ackFee, timeoutFee int64) error {
	tn := chain.ChainNodes[0]
	cmd := []string{tn.Chain.Config().Bin,
		"tx",
		"ibc-fee",
		"pay-packet-fee",
		portId,
		channelId,
		fmt.Sprintf("%d", sequenceNumber),
		"--from", fromAddress,
		"--recv-fee", fmt.Sprintf("%d%s", recvFee, chain.Config().Denom),
		"--ack-fee", fmt.Sprintf("%d%s", ackFee, chain.Config().Denom),
		"--timeout-fee", fmt.Sprintf("%d%s", timeoutFee, chain.Config().Denom),
		"--keyring-backend", keyring.BackendTest,
		"--home", tn.NodeHome(),
		"--node", fmt.Sprintf("tcp://%s:26657", tn.HostName()),
		"--output", "json",
		"--chain-id", chain.Config().ChainID,
		"--yes",
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}

	return nil
}

// FeeMiddlewareChannelOptions configures both of the chains to have fee middleware enabled.
func FeeMiddlewareChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
		opts.DestPortName = "transfer"
		opts.SourcePortName = "transfer"
	}
}
