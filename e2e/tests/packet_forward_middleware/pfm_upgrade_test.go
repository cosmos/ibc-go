//go:build !test_e2e

package pfm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// TODO: Move to `e2e/tests/upgrades` in #8360
type PFMUpgradeTestSuite struct {
	testsuite.E2ETestSuite
}

func TestPFMUpgradeTestSuite(t *testing.T) {
	// TODO: Enable as we clean up these tests #8360
	t.Skip("Skipping as relayer is not relaying failed packets")
	testCfg := testsuite.LoadConfig()
	if testCfg.UpgradePlanName == "" {
		t.Fatalf("%s must be set when running an upgrade test", testsuite.ChainUpgradePlanEnv)
	}

	// testifysuite.Run(t, new(PFMUpgradeTestSuite))
}

func updateGenesisChainB(option *testsuite.ChainOptions) {
	option.ChainSpecs[1].ModifyGenesis = cosmos.ModifyGenesis([]cosmos.GenesisKV{
		{
			Key:   "app_state.gov.params.voting_period",
			Value: "15s",
		},
		{
			Key:   "app_state.gov.params.max_deposit_period",
			Value: "10s",
		},
		{
			Key:   "app_state.gov.params.min_deposit.0.denom",
			Value: "ustake",
		},
	})
}

func (s *PFMUpgradeTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 4, nil, updateGenesisChainB)
}

func (s *PFMUpgradeTestSuite) TestV8ToV10ChainUpgrade_PacketForward() {
	t := s.T()
	ctx := context.TODO()
	testName := t.Name()

	chains := s.GetAllChains()
	chainA, chainB, chainC, _ := chains[0], chains[1], chains[2], chains[3]

	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)

	relayer := s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), t.Name())
	s.StartRelayer(relayer, testName)

	chanAB := s.GetChainAToChainBChannel(testName)
	chanBC := s.GetChainBToChainCChannel(testName)
	chanCD := s.GetChainCToChainDChannel(testName)

	ab, err := query.Channel(ctx, chainA, transfertypes.PortID, chanAB.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(ab)

	bc, err := query.Channel(ctx, chainB, transfertypes.PortID, chanBC.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(bc)

	cd, err := query.Channel(ctx, chainC, transfertypes.PortID, chanCD.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(cd)

	escrowAddrA := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)

	denomB := chainB.Config().Denom
	ibcTokenA := testsuite.GetIBCToken(denomB, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("Send from B -> A", func(_ *testing.T) {
		aHeight, err := chainA.Height(ctx)
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainB, userB, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID, testvalues.DefaultTransferAmount(denomB), userB.FormattedAddress(), userA.FormattedAddress(), s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(txResp)

		bBal, err := s.GetChainBNativeBalance(ctx, userB)
		s.Require().NoError(err)
		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, bBal)

		_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainA.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, aHeight, aHeight+40, nil)
		s.Require().NoError(err)

		escrowBalB, err := query.Balance(ctx, chainB, escrowAddrA.String(), denomB)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalB.Int64())

		escrowBalA, err := query.Balance(ctx, chainA, userA.FormattedAddress(), ibcTokenA.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalA.Int64())
	})

	// Send the IBC denom that chain A received from the previous step
	t.Run("Send from A -> B -> C ->X D", func(_ *testing.T) {
		secondHopMetadata := &PacketMetadata{
			Forward: &ForwardMetadata{
				Receiver: "cosmos1wgz9ntx6e5vu4npeabcde88d7kfsymag62p6y2",
				Channel:  chanCD.ChannelID,
				Port:     chanCD.PortID,
			},
		}
		nextBz, err := json.Marshal(secondHopMetadata)
		s.Require().NoError(err)
		next := string(nextBz)

		firstHopMetadata := &PacketMetadata{
			Forward: &ForwardMetadata{
				Receiver: userC.FormattedAddress(),
				Channel:  chanBC.ChannelID,
				Port:     chanBC.PortID,
				Next:     &next,
			},
		}

		memo, err := json.Marshal(firstHopMetadata)
		s.Require().NoError(err)

		bHeight, err := chainB.Height(ctx)
		s.Require().NoError(err)

		ibcDenomOnA := ibcTokenA.IBCDenom()
		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(ibcDenomOnA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, string(memo))
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParsePacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)

		_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainB.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, bHeight, bHeight+40, nil)
		s.Require().NoError(err)

		actualBalance, err := query.Balance(ctx, chainA, userA.FormattedAddress(), ibcDenomOnA)
		s.Require().NoError(err)
		s.Require().Zero(actualBalance)

		escrowBalA, err := query.Balance(ctx, chainA, escrowAddrA.String(), ibcDenomOnA)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalA.Int64())

		// Assart Packet relayed
		s.Require().Eventually(func() bool {
			_, err := query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
				PortId:    chanAB.PortID,
				ChannelId: chanAB.ChannelID,
				Sequence:  packet.Sequence,
			})
			return err != nil && strings.Contains(err.Error(), "packet commitment hash not found")
		}, time.Second*70, time.Second)
	})
}
