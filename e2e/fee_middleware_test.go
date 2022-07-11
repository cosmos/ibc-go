package e2e

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/ibctest/broadcast"
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

// RegisterCounterPartyPayee broadcasts a MsgRegisterCounterpartyPayee message.
func (s *FeeMiddlewareTestSuite) RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain, user broadcast.User, portID, channelID, relayerAddr, counterpartyPayeeAddr string) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// QueryCounterPartyPayee queries the counterparty payee of the given chain and relayer address on the specified channel.
func (s *FeeMiddlewareTestSuite) QueryCounterPartyPayee(ctx context.Context, chain ibc.Chain, relayerAddress, channelID string) (string, error) {
	queryClient := s.GetChainGRCPClientSet(chain).FeeQueryClient
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
func (s *FeeMiddlewareTestSuite) PayPacketFeeAsync(ctx context.Context, chain *cosmos.CosmosChain, user broadcast.User, packetID channeltypes.PacketId, packetFee feetypes.PacketFee) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgPayPacketFeeAsync(packetID, packetFee)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// QueryIncentivizedPacketsForChannel queries the incentivized packets on the specified channel.
func (s *FeeMiddlewareTestSuite) QueryIncentivizedPacketsForChannel(ctx context.Context, chain *cosmos.CosmosChain, portId, channelId string) ([]*feetypes.IdentifiedPacketFees, error) {
	queryClient := s.GetChainGRCPClientSet(chain).FeeQueryClient
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

	var (
		chainATx           ibc.Tx
		payPacketFeeTxResp sdk.TxResponse
	)

	relayer, channelA := s.CreateChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	startingTokenAmount := int64(10_000_000)

	chainAWallet := s.CreateUserOnChainA(ctx, startingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, relayer)
		s.Require().NoError(err)
	})

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	_, chainBRelayerUser := s.GetRelayerUsers(ctx)

	t.Run("register counter party payee", func(t *testing.T) {
		resp, err := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.Address, chainARelayerWallet.Address)
		s.Require().NoError(err)
		s.AssertValidTxResponse(resp)
		// give some time for update
		time.Sleep(time.Second * 5)
	})

	t.Run("verify counter party payee", func(t *testing.T) {
		address, err := s.QueryCounterPartyPayee(ctx, chainB, chainBRelayerWallet.Address, channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.Address, address)
	})

	chain1WalletToChain2WalletAmount := ibc.WalletAmount{
		Address: chainAWallet.Bech32Address(chainB.Config().Bech32Prefix), // destination address
		Denom:   chainADenom,
		Amount:  10000,
	}

	t.Run("send IBC transfer", func(t *testing.T) {
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName, chain1WalletToChain2WalletAmount, nil)
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "chain-a ibc transfer tx is invalid")
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - chainA.GetGasFeesInNativeDenom(chainATx.GasSpent)
		s.Require().Equal(expected, actualBalance)
	})

	fee := feetypes.Fee{
		RecvFee:    sdk.NewCoins(sdk.NewCoin(chainADenom, sdk.NewInt(50))),
		AckFee:     sdk.NewCoins(sdk.NewCoin(chainADenom, sdk.NewInt(25))),
		TimeoutFee: sdk.NewCoins(sdk.NewCoin(chainADenom, sdk.NewInt(10))),
	}

	t.Run("pay packet fee", func(t *testing.T) {

		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetId := channeltypes.NewPacketId(channelA.PortID, channelA.ChannelID, 1)
		packetFee := feetypes.NewPacketFee(fee, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), nil)

		t.Run("should succeed", func(t *testing.T) {
			payPacketFeeTxResp, err = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetId, packetFee)
			s.Require().NoError(err)
			s.AssertValidTxResponse(payPacketFeeTxResp)
			// wait so that incentivised packets will show up
			// wait 2 blocks
			time.Sleep(5 * time.Second)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.IsEqual(fee.RecvFee))
			s.Require().True(actualFee.AckFee.IsEqual(fee.AckFee))
			s.Require().True(actualFee.TimeoutFee.IsEqual(fee.TimeoutFee))
		})

		t.Run("balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
			// The balance should be lowered by the sum of the recv, ack and timeout fees.
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			gasFees := chainA.GetGasFeesInNativeDenom(chainATx.GasSpent) + chainA.GetGasFeesInNativeDenom(payPacketFeeTxResp.GasWanted)
			expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFees - fee.Total().AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})

	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	// wait for packets.

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("timeout fee is refunded", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		gasFees := chainA.GetGasFeesInNativeDenom(chainATx.GasSpent) + chainA.GetGasFeesInNativeDenom(payPacketFeeTxResp.GasWanted)
		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFees - fee.AckFee.AmountOf(chainADenom).Int64() - fee.RecvFee.AmountOf(chainADenom).Int64()
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
