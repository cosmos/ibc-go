package e2e

import (
	"context"
	"fmt"
	"github.com/cosmos/ibc-go/v3/e2e/e2efee"
	"github.com/cosmos/ibc-go/v3/e2e/setup"
	"github.com/ory/dockertest/v3"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
	"strings"
	"testing"
	"time"
)

func TestFeeMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
	suite.Suite
	srcChain, dstChain *e2efee.FeeMiddlewareChain
	logger             *zap.Logger
	pool               *dockertest.Pool
	network            string
	relayers           map[string]ibc.Relayer
}

func (s *FeeMiddlewareTestSuite) createRelayerAndChannel(ctx context.Context, req *require.Assertions, eRep *testreporter.RelayerExecReporter) (ibc.Relayer, ibc.ChannelOutput, ibc.ChannelOutput) {

	home := s.T().TempDir() // Must be before chain cleanup to avoid test error during cleanup.
	r := setup.NewRelayer(s.T(), s.logger, s.pool, s.network, home)

	pathName := fmt.Sprintf("%s-path", s.T().Name())
	pathName = strings.ReplaceAll(pathName, "/", "-")
	//s.relayers[pathName] = r

	ic := ibctest.NewInterchain().
		AddChain(s.srcChain).
		AddChain(s.dstChain).
		AddRelayer(r, "r").
		AddLink(ibctest.InterchainLink{
			Chain1:  s.srcChain,
			Chain2:  s.dstChain,
			Relayer: r,
			Path:    pathName,
		})

	req.NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:         s.T().Name(),
		HomeDir:          home,
		Pool:             s.pool,
		NetworkID:        s.network,
		SkipPathCreation: true,
	}))

	req.NoError(r.GeneratePath(ctx, eRep, s.srcChain.Config().ChainID, s.dstChain.Config().ChainID, pathName))
	req.NoError(r.CreateClients(ctx, eRep, pathName))

	// The client isn't created immediately -- wait for two blocks to ensure the clients are ready.
	req.NoError(test.WaitForBlocks(ctx, 2, s.srcChain, s.dstChain))
	req.NoError(r.CreateConnections(ctx, eRep, pathName))
	req.NoError(r.CreateChannel(ctx, eRep, pathName, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          "unordered",
		Version:        "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}",
	}))

	// Now validate that the channels correctly report as created.
	// GetChannels takes around two seconds with rly,
	// so getting the channels concurrently is a measurable speedup.
	eg, egCtx := errgroup.WithContext(ctx)
	var srcChainChannels, dstChainChannels []ibc.ChannelOutput
	eg.Go(func() error {
		var err error
		srcChainChannels, err = r.GetChannels(egCtx, eRep, s.srcChain.Config().ChainID)
		return err
	})
	eg.Go(func() error {
		var err error
		dstChainChannels, err = r.GetChannels(egCtx, eRep, s.dstChain.Config().ChainID)
		return err
	})
	req.NoError(eg.Wait(), "failure retrieving channels")

	return r, srcChainChannels[len(srcChainChannels)-1], dstChainChannels[len(dstChainChannels)-1]
}
func (s *FeeMiddlewareTestSuite) SetupSuite() {

	ctx := context.Background()
	pool, network := ibctest.DockerSetup(s.T())

	s.logger = zap.NewExample()
	s.pool = pool
	s.network = network
	s.relayers = map[string]ibc.Relayer{}
	//home := t.TempDir() // Must be before chain cleanup to avoid test error during cleanup.

	//srcChain, dstChain, relayer := setup.StandardTwoChainEnvironment(t, req, eRep)

	chainAConfig := setup.NewSimappConfig("simapp-a", "chain-a", "atoma")
	chainBConfig := setup.NewSimappConfig("simapp-b", "chain-b", "atomb")
	logger := zaptest.NewLogger(s.T())
	srcChain := cosmos.NewCosmosChain(s.T().Name(), chainAConfig, 1, 0, logger)
	dstChain := cosmos.NewCosmosChain(s.T().Name(), chainBConfig, 1, 0, logger)

	s.T().Cleanup(func() {
		if !s.T().Failed() {
			for _, c := range []*cosmos.CosmosChain{srcChain, dstChain} {
				if err := c.Cleanup(ctx); err != nil {
					s.T().Logf("Chain cleanup for %s failed: %v", c.Config().ChainID, err)
				}
			}
		}
	})

	s.srcChain = &e2efee.FeeMiddlewareChain{CosmosChain: srcChain}
	s.dstChain = &e2efee.FeeMiddlewareChain{CosmosChain: dstChain}

}

func (s *FeeMiddlewareTestSuite) SetupTest() {
	//t := s.T()
	//ctx := context.TODO()
	//rep := testreporter.NewNopReporter()
	//req := require.New(rep.TestifyT(t))
	//eRep := rep.RelayerExecReporter(t)

	//home := t.TempDir() // Must be before chain cleanup to avoid test error during cleanup.
	//r := setup.NewRelayer(s.T(), s.logger, s.pool, s.network, home)

	//pathName := fmt.Sprintf("%s-path", s.T().Name())
	//pathName = strings.ReplaceAll(pathName, "/", "-")
	//s.relayers[pathName] = r
	//
	//ic := ibctest.NewInterchain().
	//	AddChain(s.srcChain).
	//	AddChain(s.dstChain).
	//	AddRelayer(r, "r").
	//	AddLink(ibctest.InterchainLink{
	//		Chain1:  s.srcChain,
	//		Chain2:  s.dstChain,
	//		Relayer: r,
	//		Path:    pathName,
	//	})
	//
	//req.NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
	//	TestName:         t.Name(),
	//	HomeDir:          home,
	//	Pool:             s.pool,
	//	NetworkID:        s.network,
	//	SkipPathCreation: true,
	//}))
	//
	//req.NoError(r.GeneratePath(ctx, eRep, s.srcChain.Config().ChainID, s.dstChain.Config().ChainID, pathName))
	//req.NoError(r.CreateClients(ctx, eRep, pathName))
	//
	//// The client isn't created immediately -- wait for two blocks to ensure the clients are ready.
	//req.NoError(test.WaitForBlocks(ctx, 2, s.srcChain, s.dstChain))
	//req.NoError(r.CreateConnections(ctx, eRep, pathName))
	//req.NoError(r.CreateChannel(ctx, eRep, pathName, ibc.CreateChannelOptions{
	//	SourcePortName: "transfer",
	//	DestPortName:   "transfer",
	//	Order:          "unordered",
	//	Version:        "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}",
	//}))
}

func (s *FeeMiddlewareTestSuite) TestFeeMiddlewareSync() {
	t := s.T()
	ctx := context.TODO()
	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	relayer, srcChainChannel, dstChainChannel := s.createRelayerAndChannel(ctx, req, eRep)

	srcChain, dstChain := s.srcChain, s.dstChain

	startingTokenAmount := int64(10_000_000)

	users := ibctest.GetAndFundTestUsers(t, ctx, strings.ReplaceAll(t.Name(), " ", "-"), startingTokenAmount, srcChain, dstChain, srcChain, dstChain)

	srcRelayUser := users[0]
	invalidDstRelayUser := users[1]

	chain1Wallet := users[2]
	chain2Wallet := users[3]

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Register Counter Party Payee", func(t *testing.T) {
		req.NoError(dstChain.RegisterCounterPartyPayee(ctx, srcRelayUser.Bech32Address(srcChain.Config().Bech32Prefix), invalidDstRelayUser.Bech32Address(dstChain.Config().Bech32Prefix), dstChainChannel.PortID, dstChainChannel.ChannelID))
		time.Sleep(5 * time.Second)
	})

	t.Run("Verify Counter Party Payee", func(t *testing.T) {
		address, err := dstChain.QueryCounterPartyPayee(ctx, invalidDstRelayUser.Bech32Address(dstChain.Config().Bech32Prefix), dstChainChannel.ChannelID)
		req.NoError(err)
		req.Equal(srcRelayUser.Bech32Address(srcChain.Config().Bech32Prefix), address)
	})

	chain1WalletToChain2WalletAmount := ibc.WalletAmount{
		Address: chain2Wallet.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
		Denom:   srcChain.Config().Denom,
		Amount:  10000,
	}

	var srcTx ibc.Tx

	t.Run("Send IBC transfer", func(t *testing.T) {
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, srcChainChannel.ChannelID, chain1Wallet.KeyName, chain1WalletToChain2WalletAmount, nil)
		req.NoError(err)
		req.NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
	})

	t.Run("Verify tokens have been escrowed", func(t *testing.T) {
		actualBalance, err := srcChain.GetBalance(ctx, chain1Wallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		req.Equal(expected, actualBalance)
	})

	recvFee := int64(50)
	ackFee := int64(25)
	timeoutFee := int64(10)

	t.Run("Pay packet fee", func(t *testing.T) {
		t.Run("Before paying packet fee there should be no incentivized packets", func(t *testing.T) {
			packets, err := srcChain.QueryPackets(ctx, srcChainChannel.PortID, srcChainChannel.PortID)
			req.NoError(err)
			req.Len(packets.IncentivizedPackets, 0)
		})

		t.Run("Paying packet fee should succeed", func(t *testing.T) {
			err := srcChain.PayPacketFee(ctx, chain1Wallet.KeyName, srcChainChannel.PortID, srcChainChannel.ChannelID, 1, recvFee, ackFee, timeoutFee)
			req.NoError(err)

			// wait so that incentivised packets will show up
			time.Sleep(5 * time.Second)
		})

		// TODO: query method not umarshalling json correctly yet.
		//t.Run("After paying packet fee there should be incentivized packets", func(t *testing.T) {
		//	packets, err := srcChain.QueryPackets(ctx, "transfer", "channel-0")
		//	req.NoError(err)
		//	req.Len(packets.IncentivizedPackets, 1)
		//})
	})

	t.Run("Balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := srcChain.GetBalance(ctx, chain1Wallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - recvFee - ackFee - timeoutFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Start relayer", func(t *testing.T) {
		//r := s.relayers["TestFeeMiddlewareTestSuite-TestFeeMiddlewareSync-path"]
		err := relayer.StartRelayer(ctx, eRep, "TestFeeMiddlewareTestSuite-TestFeeMiddlewareSync-path")
		req.NoError(err, fmt.Sprintf("failed to start relayer: %s", err))
		t.Cleanup(func() {
			if !t.Failed() {
				if err := relayer.StopRelayer(ctx, eRep); err != nil {
					t.Logf("error stopping relayer: %v", err)
				}
			}
		})
		// wait for relayer to start.
		time.Sleep(time.Second * 10)
	})

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Packets should have been relayed", func(t *testing.T) {
		packets, err := srcChain.QueryPackets(ctx, srcChainChannel.PortID, srcChainChannel.ChannelID)
		req.NoError(err)
		req.Len(packets.IncentivizedPackets, 0)
	})

	t.Run("Verify recv fees are refunded when no forward relayer is found", func(t *testing.T) {
		actualBalance, err := srcChain.GetBalance(ctx, chain1Wallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		gasFee := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFee - ackFee
		req.Equal(expected, actualBalance)
	})
}

func (s *FeeMiddlewareTestSuite) TestFeeMiddlewareAsync() {
	t := s.T()
	ctx := context.TODO()
	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	relayer, srcChainChannel, dstChainChannel := s.createRelayerAndChannel(ctx, req, eRep)

	srcChain, dstChain := s.srcChain, s.dstChain

	startingTokenAmount := int64(10_000_000)

	users := ibctest.GetAndFundTestUsers(t, ctx, strings.ReplaceAll(t.Name(), " ", "-"), startingTokenAmount, srcChain, dstChain, srcChain, dstChain)

	srcRelayUser := users[0]
	invalidDstRelayUser := users[1]

	chain1Wallet := users[2]
	chain2Wallet := users[3]

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Register Counter Party Payee", func(t *testing.T) {
		req.NoError(dstChain.RegisterCounterPartyPayee(ctx, srcRelayUser.Bech32Address(srcChain.Config().Bech32Prefix), invalidDstRelayUser.Bech32Address(dstChain.Config().Bech32Prefix), dstChainChannel.PortID, dstChainChannel.ChannelID))
		time.Sleep(5 * time.Second)
	})

	t.Run("Verify Counter Party Payee", func(t *testing.T) {
		address, err := dstChain.QueryCounterPartyPayee(ctx, invalidDstRelayUser.Bech32Address(dstChain.Config().Bech32Prefix), dstChainChannel.ChannelID)
		req.NoError(err)
		req.Equal(srcRelayUser.Bech32Address(srcChain.Config().Bech32Prefix), address)
	})

	chain1WalletToChain2WalletAmount := ibc.WalletAmount{
		Address: chain2Wallet.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
		Denom:   srcChain.Config().Denom,
		Amount:  10000,
	}

	var srcTx ibc.Tx

	t.Run("Send IBC transfer", func(t *testing.T) {
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, srcChainChannel.PortID, chain1Wallet.KeyName, chain1WalletToChain2WalletAmount, nil)
		req.NoError(err)
		req.NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
	})

	t.Run("Verify tokens have been escrowed", func(t *testing.T) {
		actualBalance, err := srcChain.GetBalance(ctx, chain1Wallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		req.Equal(expected, actualBalance)
	})

	recvFee := int64(50)
	ackFee := int64(25)
	timeoutFee := int64(10)

	t.Run("Pay packet fee", func(t *testing.T) {
		t.Run("Before paying packet fee there should be no incentivized packets", func(t *testing.T) {
			packets, err := srcChain.QueryPackets(ctx, srcChainChannel.PortID, srcChainChannel.PortID)
			req.NoError(err)
			req.Len(packets.IncentivizedPackets, 0)
		})

		t.Run("Paying packet fee should succeed", func(t *testing.T) {
			err := srcChain.PayPacketFee(ctx, chain1Wallet.KeyName, srcChainChannel.PortID, srcChainChannel.ChannelID, 1, recvFee, ackFee, timeoutFee)
			req.NoError(err)

			// wait so that incentivised packets will show up
			time.Sleep(5 * time.Second)
		})

		// TODO: query method not umarshalling json correctly yet.
		//t.Run("After paying packet fee there should be incentivized packets", func(t *testing.T) {
		//	packets, err := srcChain.QueryPackets(ctx, "transfer", "channel-0")
		//	req.NoError(err)
		//	req.Len(packets.IncentivizedPackets, 1)
		//})
	})

	t.Run("Balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := srcChain.GetBalance(ctx, chain1Wallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - recvFee - ackFee - timeoutFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Start relayer", func(t *testing.T) {
		//r := s.relayers["TestFeeMiddlewareTestSuite-TestFeeMiddlewareSync-path"]
		err := relayer.StartRelayer(ctx, eRep, "TestFeeMiddlewareTestSuite-TestFeeMiddlewareSync-path")
		req.NoError(err, fmt.Sprintf("failed to start relayer: %s", err))
		t.Cleanup(func() {
			if !t.Failed() {
				if err := relayer.StopRelayer(ctx, eRep); err != nil {
					t.Logf("error stopping relayer: %v", err)
				}
			}
		})
		// wait for relayer to start.
		time.Sleep(time.Second * 10)
	})

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Packets should have been relayed", func(t *testing.T) {
		packets, err := srcChain.QueryPackets(ctx, srcChainChannel.PortID, srcChainChannel.ChannelID)
		req.NoError(err)
		req.Len(packets.IncentivizedPackets, 0)
	})

	t.Run("Verify recv fees are refunded when no forward relayer is found", func(t *testing.T) {
		actualBalance, err := srcChain.GetBalance(ctx, chain1Wallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		gasFee := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFee - ackFee
		req.Equal(expected, actualBalance)
	})
}
