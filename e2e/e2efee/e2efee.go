package e2efee

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
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

	_, _, err := tn.Exec(ctx, cmd, nil)
	return err
}

func QueryPackets(ctx context.Context, chain *cosmos.CosmosChain, portId, channelId string) (QueryIncentivizedPacketsResponse, error) {
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

	stdout, _, err := tn.Exec(ctx, cmd, nil)
	if err != nil {
		return QueryIncentivizedPacketsResponse{}, err
	}

	response := QueryIncentivizedPacketsResponse{}
	//if err := types.ModuleCdc.Unmarshal(respBytes, &response); err != nil {
	if err := json.Unmarshal(stdout, &response); err != nil {
		return QueryIncentivizedPacketsResponse{}, err
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

	stdout, _, err := tn.Exec(ctx, cmd, nil)
	if err != nil {
		return "", err
	}

	type QueryOutput struct {
		CounterPartyPayee string `json:"counterparty_payee"`
	}

	res := &QueryOutput{}
	if err := json.Unmarshal(stdout, res); err != nil {
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

	_, _, err := tn.Exec(ctx, cmd, nil)
	return err
}

// FeeMiddlewareChannelOptions configures both of the chains to have fee middleware enabled.
func FeeMiddlewareChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
		opts.DestPortName = "transfer"
		opts.SourcePortName = "transfer"
	}
}

// TODO: remove all of these types and import the protobuff generated type.

type QueryIncentivizedPacketsResponse struct {
	// list of identified fees for incentivized packets
	IncentivizedPackets []IdentifiedPacketFees `json:"incentivized_packets"`
}

// IdentifiedPacketFees contains a list of type PacketFee and associated PacketId
type IdentifiedPacketFees struct {
	// unique packet identifier comprised of the channel ID, port ID and sequence
	PacketId PacketId `json:"packet_id"`
	// list of packet fees
	PacketFees []PacketFee `json:"packet_fees"`
}

type PacketId struct {
	// channel port identifier
	PortId string `json:"port_id,omitempty"`
	// channel unique identifier
	ChannelId string `json:"channel_id,omitempty"`
	// packet sequence
	Sequence string `json:"sequence,omitempty"`
}

type PacketFee struct {
	// fee encapsulates the recv, ack and timeout fees associated with an IBC packet
	Fee Fee `json:"fee"`
	// the refund address for unspent fees
	RefundAddress string `json:"refund_address,omitempty"`
	// optional list of relayers permitted to receive fees
	Relayers []string `json:"relayers,omitempty"`
}

type Fee struct {
	// the packet receive fee
	RecvFee sdktypes.Coins `json:"recv_fee"`
	// the packet acknowledgement fee
	AckFee sdktypes.Coins `json:"ack_fee"`
	// the packet timeout fee
	TimeoutFee sdktypes.Coins `json:"timeout_fee"`
}
