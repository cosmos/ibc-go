//go:build !test_e2e

package pfm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	"cosmossdk.io/math"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

type PFMTimeoutTestSuite struct {
	testsuite.E2ETestSuite
}

func TestForwardTransferTimeoutSuite(t *testing.T) {
	// TODO: Enable as we clean up these tests #8360
	t.Skip("Skipping as relayer is not relaying failed packets")
	// testifysuite.Run(t, new(PFMTimeoutTestSuite))
}

func (s *PFMTimeoutTestSuite) TestTimeoutOnForward() {
	t := s.T()
	t.Parallel()

	ctx := context.TODO()

	chains := s.GetAllChains()
	a, b, c, d := chains[0], chains[1], chains[2], chains[3]

	relayer := s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), t.Name())
	s.StartRelayer(relayer, t.Name())

	// Fund user accounts with initial balances and get the transfer channel information between each set of chains
	usrA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	usrB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	usrC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	usrD := s.CreateUserOnChainD(ctx, testvalues.StartingTokenAmount)

	abChan := s.GetChainAToChainBChannel(t.Name())
	baChan := abChan.Counterparty
	bcChan := s.GetChainBToChainCChannel(t.Name())
	cbChan := bcChan.Counterparty
	cdChan := s.GetChainCToChainDChannel(t.Name())
	dcChan := cdChan.Counterparty

	retries := uint8(0)
	secondHopMetadata := &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrD.FormattedAddress(),
			Channel:  cdChan.ChannelID,
			Port:     cdChan.PortID,
			Retries:  &retries,
		},
	}

	nextBz, err := json.Marshal(secondHopMetadata)
	s.Require().NoError(err)
	next := string(nextBz)

	firstHopMetadata := &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrC.FormattedAddress(),
			Channel:  bcChan.ChannelID,
			Port:     bcChan.PortID,
			Next:     &next,
			Retries:  &retries,
			Timeout:  time.Second * 10, // Set low timeout for forward from chainB<>chainC
		},
	}

	memo, err := json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	opts := ibc.TransferOptions{
		Memo: string(memo),
	}

	bHeight, err := b.Height(ctx)
	s.Require().NoError(err)

	transferAmount := math.NewInt(100_000)
	// Attempt to send packet from a -> b -> c -> d
	amount := ibc.WalletAmount{
		Address: usrB.FormattedAddress(),
		Denom:   a.Config().Denom,
		Amount:  transferAmount,
	}

	transferTx, err := a.SendIBCTransfer(ctx, abChan.ChannelID, usrA.KeyName(), amount, opts)
	s.Require().NoError(err)

	// Poll for MsgRecvPacket on chainB
	_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, b.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, bHeight, bHeight+20, nil)
	s.Require().NoError(err)

	// Stop the relayer and wait for the timeout to happen on chainC
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	err = relayer.StopRelayer(ctx, eRep)
	s.Require().NoError(err)

	time.Sleep(time.Second * 11)
	s.StartRelayer(relayer, t.Name())

	aHeight, err := a.Height(ctx)
	s.Require().NoError(err)

	bHeight, err = b.Height(ctx)
	s.Require().NoError(err)

	// Poll for the MsgTimeout on chainB and the MsgAck on chainA
	_, err = cosmos.PollForMessage[*chantypes.MsgTimeout](ctx, b.(*cosmos.CosmosChain), b.Config().EncodingConfig.InterfaceRegistry, bHeight, bHeight+30, nil)
	s.Require().NoError(err)

	_, err = testutil.PollForAck(ctx, a, aHeight, aHeight+30, transferTx.Packet)
	s.Require().NoError(err)

	err = testutil.WaitForBlocks(ctx, 1, a)
	s.Require().NoError(err)

	// Assert balances to ensure that the funds are still on the original sending chain
	chainABalance, err := a.GetBalance(ctx, usrA.FormattedAddress(), a.Config().Denom)
	s.Require().NoError(err)

	// Compose the prefixed denoms and ibc denom for asserting balances
	firstHopDenom := transfertypes.GetPrefixedDenom(baChan.PortID, baChan.ChannelID, a.Config().Denom)
	secondHopDenom := transfertypes.GetPrefixedDenom(cbChan.PortID, cbChan.ChannelID, firstHopDenom)
	thirdHopDenom := transfertypes.GetPrefixedDenom(dcChan.PortID, dcChan.ChannelID, secondHopDenom)

	firstHopDenomTrace := transfertypes.ParseDenomTrace(firstHopDenom)
	secondHopDenomTrace := transfertypes.ParseDenomTrace(secondHopDenom)
	thirdHopDenomTrace := transfertypes.ParseDenomTrace(thirdHopDenom)

	firstHopIBCDenom := firstHopDenomTrace.IBCDenom()
	secondHopIBCDenom := secondHopDenomTrace.IBCDenom()
	thirdHopIBCDenom := thirdHopDenomTrace.IBCDenom()

	chainBBalance, err := b.GetBalance(ctx, usrB.FormattedAddress(), firstHopIBCDenom)
	s.Require().NoError(err)

	chainCBalance, err := c.GetBalance(ctx, usrC.FormattedAddress(), secondHopIBCDenom)
	s.Require().NoError(err)

	chainDBalance, err := d.GetBalance(ctx, usrD.FormattedAddress(), thirdHopIBCDenom)
	s.Require().NoError(err)

	initBal := math.NewInt(10_000_000_000)
	zeroBal := math.NewInt(0)

	s.Require().True(chainCBalance.Equal(zeroBal))
	s.Require().True(chainBBalance.Equal(zeroBal))
	s.Require().True(chainABalance.Equal(initBal))
	s.Require().True(chainDBalance.Equal(zeroBal))

	firstHopEscrowAccount := transfertypes.GetEscrowAddress(abChan.PortID, abChan.ChannelID).String()
	secondHopEscrowAccount := transfertypes.GetEscrowAddress(bcChan.PortID, bcChan.ChannelID).String()
	thirdHopEscrowAccount := transfertypes.GetEscrowAddress(cdChan.PortID, abChan.ChannelID).String()

	firstHopEscrowBalance, err := a.GetBalance(ctx, firstHopEscrowAccount, a.Config().Denom)
	s.Require().NoError(err)

	secondHopEscrowBalance, err := b.GetBalance(ctx, secondHopEscrowAccount, firstHopIBCDenom)
	s.Require().NoError(err)

	thirdHopEscrowBalance, err := c.GetBalance(ctx, thirdHopEscrowAccount, secondHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(firstHopEscrowBalance.Equal(zeroBal))
	s.Require().True(secondHopEscrowBalance.Equal(zeroBal))
	s.Require().True(thirdHopEscrowBalance.Equal(zeroBal))

	// Send IBC transfer from ChainA -> ChainB -> ChainC -> ChainD that will succeed
	secondHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrD.FormattedAddress(),
			Channel:  cdChan.ChannelID,
			Port:     cdChan.PortID,
		},
	}
	nextBz, err = json.Marshal(secondHopMetadata)
	s.Require().NoError(err)
	next = string(nextBz)

	firstHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrC.FormattedAddress(),
			Channel:  bcChan.ChannelID,
			Port:     bcChan.PortID,
			Next:     &next,
		},
	}

	memo, err = json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	opts = ibc.TransferOptions{
		Memo: string(memo),
	}

	aHeight, err = a.Height(ctx)
	s.Require().NoError(err)

	transferTx, err = a.SendIBCTransfer(ctx, abChan.ChannelID, usrA.KeyName(), amount, opts)
	s.Require().NoError(err)

	_, err = testutil.PollForAck(ctx, a, aHeight, aHeight+30, transferTx.Packet)
	s.Require().NoError(err)

	err = testutil.WaitForBlocks(ctx, 10, a)
	s.Require().NoError(err)

	// Assert balances are updated to reflect tokens now being on ChainD
	chainABalance, err = a.GetBalance(ctx, usrA.FormattedAddress(), a.Config().Denom)
	s.Require().NoError(err)

	chainBBalance, err = b.GetBalance(ctx, usrB.FormattedAddress(), firstHopIBCDenom)
	s.Require().NoError(err)

	chainCBalance, err = c.GetBalance(ctx, usrC.FormattedAddress(), secondHopIBCDenom)
	s.Require().NoError(err)

	chainDBalance, err = d.GetBalance(ctx, usrD.FormattedAddress(), thirdHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(chainABalance.Equal(initBal.Sub(transferAmount)))
	s.Require().True(chainBBalance.Equal(zeroBal))
	s.Require().True(chainCBalance.Equal(zeroBal))
	s.Require().True(chainDBalance.Equal(transferAmount))

	firstHopEscrowBalance, err = a.GetBalance(ctx, firstHopEscrowAccount, a.Config().Denom)
	s.Require().NoError(err)

	secondHopEscrowBalance, err = b.GetBalance(ctx, secondHopEscrowAccount, firstHopIBCDenom)
	s.Require().NoError(err)

	thirdHopEscrowBalance, err = c.GetBalance(ctx, thirdHopEscrowAccount, secondHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(firstHopEscrowBalance.Equal(transferAmount))
	s.Require().True(secondHopEscrowBalance.Equal(transferAmount))
	s.Require().True(thirdHopEscrowBalance.Equal(transferAmount))

	// Compose IBC tx that will attempt to go from ChainD -> ChainC -> ChainB -> ChainA but timeout between ChainB->ChainA
	amount = ibc.WalletAmount{
		Address: usrC.FormattedAddress(),
		Denom:   thirdHopDenom,
		Amount:  transferAmount,
	}

	secondHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrA.FormattedAddress(),
			Channel:  baChan.ChannelID,
			Port:     baChan.PortID,
			Timeout:  1 * time.Second,
		},
	}

	nextBz, err = json.Marshal(secondHopMetadata)
	s.Require().NoError(err)
	next = string(nextBz)

	firstHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrB.FormattedAddress(),
			Channel:  cbChan.ChannelID,
			Port:     cbChan.PortID,
			Next:     &next,
		},
	}

	memo, err = json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	chainDHeight, err := d.Height(ctx)
	s.Require().NoError(err)

	transferTx, err = d.SendIBCTransfer(ctx, dcChan.ChannelID, usrD.KeyName(), amount, ibc.TransferOptions{Memo: string(memo)})
	s.Require().NoError(err)

	_, err = testutil.PollForAck(ctx, d, chainDHeight, chainDHeight+25, transferTx.Packet)
	s.Require().NoError(err)

	err = testutil.WaitForBlocks(ctx, 5, d)
	s.Require().NoError(err)

	// Assert balances to ensure timeout happened and user funds are still present on ChainD
	chainABalance, err = a.GetBalance(ctx, usrA.FormattedAddress(), a.Config().Denom)
	s.Require().NoError(err)

	chainBBalance, err = b.GetBalance(ctx, usrB.FormattedAddress(), firstHopIBCDenom)
	s.Require().NoError(err)

	chainCBalance, err = c.GetBalance(ctx, usrC.FormattedAddress(), secondHopIBCDenom)
	s.Require().NoError(err)

	chainDBalance, err = d.GetBalance(ctx, usrD.FormattedAddress(), thirdHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(chainABalance.Equal(initBal.Sub(transferAmount)))
	s.Require().True(chainBBalance.Equal(zeroBal))
	s.Require().True(chainCBalance.Equal(zeroBal))
	s.Require().True(chainDBalance.Equal(transferAmount))

	firstHopEscrowBalance, err = a.GetBalance(ctx, firstHopEscrowAccount, a.Config().Denom)
	s.Require().NoError(err)

	secondHopEscrowBalance, err = b.GetBalance(ctx, secondHopEscrowAccount, firstHopIBCDenom)
	s.Require().NoError(err)

	thirdHopEscrowBalance, err = c.GetBalance(ctx, thirdHopEscrowAccount, secondHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(firstHopEscrowBalance.Equal(transferAmount))
	s.Require().True(secondHopEscrowBalance.Equal(transferAmount))
	s.Require().True(thirdHopEscrowBalance.Equal(transferAmount))

	// ---

	// Compose IBC tx that will go from ChainD -> ChainC -> ChainB -> ChainA and succeed.
	amount = ibc.WalletAmount{
		Address: usrC.FormattedAddress(),
		Denom:   thirdHopDenom,
		Amount:  transferAmount,
	}

	secondHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrA.FormattedAddress(),
			Channel:  baChan.ChannelID,
			Port:     baChan.PortID,
		},
	}
	nextBz, err = json.Marshal(secondHopMetadata)
	s.Require().NoError(err)
	next = string(nextBz)

	firstHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrB.FormattedAddress(),
			Channel:  cbChan.ChannelID,
			Port:     cbChan.PortID,
			Next:     &next,
		},
	}

	memo, err = json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	chainDHeight, err = d.Height(ctx)
	s.Require().NoError(err)

	transferTx, err = d.SendIBCTransfer(ctx, dcChan.ChannelID, usrD.KeyName(), amount, ibc.TransferOptions{Memo: string(memo)})
	s.Require().NoError(err)

	_, err = testutil.PollForAck(ctx, d, chainDHeight, chainDHeight+25, transferTx.Packet)
	s.Require().NoError(err)

	err = testutil.WaitForBlocks(ctx, 5, d)
	s.Require().NoError(err)

	// Assert balances to ensure timeout happened and user funds are still present on ChainD
	chainABalance, err = a.GetBalance(ctx, usrA.FormattedAddress(), a.Config().Denom)
	s.Require().NoError(err)

	chainBBalance, err = b.GetBalance(ctx, usrB.FormattedAddress(), firstHopIBCDenom)
	s.Require().NoError(err)

	chainCBalance, err = c.GetBalance(ctx, usrC.FormattedAddress(), secondHopIBCDenom)
	s.Require().NoError(err)

	chainDBalance, err = d.GetBalance(ctx, usrD.FormattedAddress(), thirdHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(chainABalance.Equal(initBal))
	s.Require().True(chainBBalance.Equal(zeroBal))
	s.Require().True(chainCBalance.Equal(zeroBal))
	s.Require().True(chainDBalance.Equal(zeroBal))

	firstHopEscrowBalance, err = a.GetBalance(ctx, firstHopEscrowAccount, a.Config().Denom)
	s.Require().NoError(err)

	secondHopEscrowBalance, err = b.GetBalance(ctx, secondHopEscrowAccount, firstHopIBCDenom)
	s.Require().NoError(err)

	thirdHopEscrowBalance, err = c.GetBalance(ctx, thirdHopEscrowAccount, secondHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(firstHopEscrowBalance.Equal(zeroBal))
	s.Require().True(secondHopEscrowBalance.Equal(zeroBal))
	s.Require().True(thirdHopEscrowBalance.Equal(zeroBal))

	// ----- 2

	// Compose IBC tx that will go from ChainD -> ChainC -> ChainB -> ChainA and succeed.
	amount = ibc.WalletAmount{
		Address: usrB.FormattedAddress(),
		Denom:   a.Config().Denom,
		Amount:  transferAmount,
	}

	firstHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: usrA.FormattedAddress(),
			Channel:  baChan.ChannelID,
			Port:     baChan.PortID,
			Timeout:  1 * time.Second,
		},
	}

	memo, err = json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	aHeight, err = a.Height(ctx)
	s.Require().NoError(err)

	transferTx, err = a.SendIBCTransfer(ctx, abChan.ChannelID, usrA.KeyName(), amount, ibc.TransferOptions{Memo: string(memo)})
	s.Require().NoError(err)

	_, err = testutil.PollForAck(ctx, a, aHeight, aHeight+25, transferTx.Packet)
	s.Require().NoError(err)

	err = testutil.WaitForBlocks(ctx, 5, a)
	s.Require().NoError(err)

	// Assert balances to ensure timeout happened and user funds are still present on ChainD
	chainABalance, err = a.GetBalance(ctx, usrA.FormattedAddress(), a.Config().Denom)
	s.Require().NoError(err)

	chainBBalance, err = b.GetBalance(ctx, usrB.FormattedAddress(), firstHopIBCDenom)
	s.Require().NoError(err)

	chainCBalance, err = c.GetBalance(ctx, usrC.FormattedAddress(), secondHopIBCDenom)
	s.Require().NoError(err)

	chainDBalance, err = d.GetBalance(ctx, usrD.FormattedAddress(), thirdHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(chainABalance.Equal(initBal))
	s.Require().True(chainBBalance.Equal(zeroBal))
	s.Require().True(chainCBalance.Equal(zeroBal))
	s.Require().True(chainDBalance.Equal(zeroBal))

	firstHopEscrowBalance, err = a.GetBalance(ctx, firstHopEscrowAccount, a.Config().Denom)
	s.Require().NoError(err)

	secondHopEscrowBalance, err = b.GetBalance(ctx, secondHopEscrowAccount, firstHopIBCDenom)
	s.Require().NoError(err)

	thirdHopEscrowBalance, err = c.GetBalance(ctx, thirdHopEscrowAccount, secondHopIBCDenom)
	s.Require().NoError(err)

	s.Require().True(firstHopEscrowBalance.Equal(zeroBal))
	s.Require().True(secondHopEscrowBalance.Equal(zeroBal))
	s.Require().True(thirdHopEscrowBalance.Equal(zeroBal))
}

// TODO: Try to replace this with PFM's own version of this struct #8360
type PacketMetadata struct {
	Forward *ForwardMetadata `json:"forward"`
}

type ForwardMetadata struct {
	Receiver       string        `json:"receiver"`
	Port           string        `json:"port"`
	Channel        string        `json:"channel"`
	Timeout        time.Duration `json:"timeout"`
	Retries        *uint8        `json:"retries,omitempty"`
	Next           *string       `json:"next,omitempty"`
	RefundSequence *uint64       `json:"refund_sequence,omitempty"`
}
