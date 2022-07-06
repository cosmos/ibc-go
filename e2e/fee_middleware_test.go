package e2e

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v4/e2e/testsuite"
	feetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
)

func TestFeeMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *FeeMiddlewareTestSuite) RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain /*user  broadcast.User*/, portID, channelID, relayerAddr, counterpartyPayeeAddr string) error {
	_ = feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	//s.AssertValidTxResponse(tx)
	return nil
}

func (s *FeeMiddlewareTestSuite) QueryCounterPartyPayee(ctx context.Context, chain ibc.Chain, relayerAddress, channelID string) (string, error) {
	queryClient := s.GetChainClientSet(chain).FeeQueryClient
	res, err := queryClient.CounterpartyPayee(ctx, &feetypes.QueryCounterpartyPayeeRequest{
		ChannelId: channelID,
		Relayer:   relayerAddress,
	})

	if err != nil {
		return "", err
	}
	return res.CounterpartyPayee, nil
}

func (s *FeeMiddlewareTestSuite) PayPacketFeeAsync(ctx context.Context, chain ibc.Chain, packetID channeltypes.PacketId, packetFee feetypes.PacketFee) error {
	_ = feetypes.NewMsgPayPacketFeeAsync(packetID, packetFee)
	return nil
}

// QueryIncentivizedPacketsForChannel queries the incentivized packets on the specified channel.
func (s *FeeMiddlewareTestSuite) QueryIncentivizedPacketsForChannel(ctx context.Context, chain *cosmos.CosmosChain, portId, channelId string) ([]*feetypes.IdentifiedPacketFees, error) {
	queryClient := s.GetChainClientSet(chain).FeeQueryClient
	res, err := queryClient.IncentivizedPacketsForChannel(ctx, &feetypes.QueryIncentivizedPacketsForChannelRequest{
		PortId:    portId,
		ChannelId: channelId,
	},
	)
	if err != nil {
		return nil, err
	}
	return res.IncentivizedPackets, err
}

func (s *FeeMiddlewareTestSuite) TestAsyncSingleSender() {
	t := s.T()
	ctx := context.TODO()

	relayer := s.CreateChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	srcChain, dstChain := s.GetChains()
	srcChannel := s.GetChannel(ctx, srcChain, relayer)

	startingTokenAmount := int64(10_000_000)

	srcChainWallet := s.CreateUserOnSourceChain(ctx, startingTokenAmount)
	dstChainWallet := s.CreateUserOnDestinationChain(ctx, startingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, relayer)
		s.Require().NoError(err)
	})

	srcRelayerWallet, dstRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, srcChain, dstChain), "failed to wait for blocks")

	t.Run("register counter party payee", func(t *testing.T) {
		err := s.RegisterCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, srcRelayerWallet.Address, srcChannel.Counterparty.PortID, srcChannel.Counterparty.ChannelID)
		s.Require().NoError(err)

		// give some time for update
		time.Sleep(time.Second * 5)
	})

	t.Run("verify counter party payee", func(t *testing.T) {
		address, err := s.QueryCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, "channel-0")
		s.Require().NoError(err)
		s.Require().Equal(srcRelayerWallet.Address, address)
	})

	chain1WalletToChain2WalletAmount := ibc.WalletAmount{
		Address: dstChainWallet.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
		Denom:   srcChain.Config().Denom,
		Amount:  10000,
	}

	var srcTx ibc.Tx
	t.Run("send IBC transfer", func(t *testing.T) {
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, srcChannel.ChannelID, srcChainWallet.KeyName, chain1WalletToChain2WalletAmount, nil)
		s.Require().NoError(err)
		s.Require().NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetSourceChainNativeBalance(ctx, srcChainWallet)
		s.Require().NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		s.Require().Equal(expected, actualBalance)
	})

	srcDenom := srcChain.Config().Denom
	fee := feetypes.Fee{
		RecvFee:    sdk.NewCoins(sdk.NewCoin(srcDenom, sdk.NewInt(50))),
		AckFee:     sdk.NewCoins(sdk.NewCoin(srcDenom, sdk.NewInt(25))),
		TimeoutFee: sdk.NewCoins(sdk.NewCoin(srcDenom, sdk.NewInt(10))),
	}

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, srcChain, srcChannel.PortID, srcChannel.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetId := channeltypes.NewPacketId(srcChannel.PortID, srcChannel.ChannelID, 1)
		packetFee := feetypes.NewPacketFee(fee, string(srcChainWallet.Address), nil)

		t.Run("should succeed", func(t *testing.T) {

			err := s.PayPacketFeeAsync(ctx, srcChain, packetId, packetFee)
			s.Require().NoError(err)

			// wait so that incentivised packets will show up
			time.Sleep(5 * time.Second)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, srcChain, srcChannel.PortID, srcChannel.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.IsEqual(fee.RecvFee))
			s.Require().True(actualFee.AckFee.IsEqual(fee.AckFee))
			s.Require().True(actualFee.TimeoutFee.IsEqual(fee.TimeoutFee))
		})
	})

	t.Run("balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := s.GetSourceChainNativeBalance(ctx, srcChainWallet)
		s.Require().NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - fee.Total().AmountOf(srcDenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, srcChain, srcChannel.PortID, srcChannel.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("timeout fee is refunded", func(t *testing.T) {
		actualBalance, err := s.GetSourceChainNativeBalance(ctx, srcChainWallet)
		s.Require().NoError(err)

		gasFee := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFee - fee.AckFee.AmountOf(srcDenom).Int64() - fee.RecvFee.AmountOf(srcDenom).Int64()
		s.Require().Equal(expected, actualBalance)
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
