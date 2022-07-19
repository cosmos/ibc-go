package e2e

import (
	"context"
	"e2e/testsuite"
	"testing"

	"github.com/strangelove-ventures/ibctest/broadcast"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	feetypes "github.com/cosmos/ibc-go/v5/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
)

func TestFeeMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
	testsuite.E2ETestSuite
}

// RegisterCounterPartyPayee broadcasts a MsgRegisterCounterpartyPayee message.
func (s *FeeMiddlewareTestSuite) RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain,
	user broadcast.User, portID, channelID, relayerAddr, counterpartyPayeeAddr string,
) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// QueryCounterPartyPayee queries the counterparty payee of the given chain and relayer address on the specified channel.
func (s *FeeMiddlewareTestSuite) QueryCounterPartyPayee(ctx context.Context, chain ibc.Chain, relayerAddress, channelID string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).FeeQueryClient
	res, err := queryClient.CounterpartyPayee(ctx, &feetypes.QueryCounterpartyPayeeRequest{
		ChannelId: channelID,
		Relayer:   relayerAddress,
	})
	if err != nil {
		return "", err
	}
	return res.CounterpartyPayee, nil
}

// PayPacketFeeAsync broadcasts a MsgPayPacketFeeAsync message.
func (s *FeeMiddlewareTestSuite) PayPacketFeeAsync(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	user broadcast.User,
	packetID channeltypes.PacketId,
	packetFee feetypes.PacketFee,
) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgPayPacketFeeAsync(packetID, packetFee)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// QueryIncentivizedPacketsForChannel queries the incentivized packets on the specified channel.
func (s *FeeMiddlewareTestSuite) QueryIncentivizedPacketsForChannel(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	portId,
	channelId string,
) ([]*feetypes.IdentifiedPacketFees, error) {
	queryClient := s.GetChainGRCPClients(chain).FeeQueryClient
	res, err := queryClient.IncentivizedPacketsForChannel(ctx, &feetypes.QueryIncentivizedPacketsForChannelRequest{
		PortId:    portId,
		ChannelId: channelId,
	})
	if err != nil {
		return nil, err
	}
	return res.IncentivizedPackets, err
}

func (s *FeeMiddlewareTestSuite) TestPlaceholder() {
	ctx := context.Background()
	r := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	s.T().Run("start relayer", func(t *testing.T) {
		s.StartRelayer(r)
	})
}

// feeMiddlewareChannelOptions configures both of the chains to have fee middleware enabled.
func feeMiddlewareChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
		opts.DestPortName = "transfer"
		opts.SourcePortName = "transfer"
	}
}
