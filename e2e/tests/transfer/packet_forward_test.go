//go:build !test_e2e

package transfer

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestTransferPacketForwardTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferPacketForwardTestSuite))
}

type TransferPacketForwardTestSuite struct {
	transferTester
}

func (s *TransferPacketForwardTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 3, nil)
}

func (s *TransferPacketForwardTestSuite) TestMsgTransfer_PacketForward_HappyPath() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()

	chains := s.GetAllChains()
	chainA := chains[0]
	chainB := chains[1]
	chainC := chains[2]

	relayer := s.GetRelayerForTest(testName)
	channelA, channelBFromAB := s.CreatePath(ctx, relayer, chainA, chainB, ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	channelBToC, channelC := s.CreatePath(ctx, relayer, chainB, chainC, ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)

	senderOnA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	receiverOnB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	receiverOnC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)

	transferAmount := testvalues.DefaultTransferAmount(chainA.Config().Denom)

	memo, err := json.Marshal(map[string]any{
		"forward": map[string]string{
			"receiver": receiverOnC.FormattedAddress(),
			"port":     transfertypes.PortID,
			"channel":  channelBToC.ChannelID,
		},
	})
	s.Require().NoError(err)

	transferTxResp := s.Transfer(
		ctx,
		chainA,
		senderOnA,
		channelA.PortID,
		channelA.ChannelID,
		transferAmount,
		senderOnA.FormattedAddress(),
		receiverOnB.FormattedAddress(),
		s.GetTimeoutHeight(ctx, chainB),
		0,
		string(memo),
	)
	s.AssertTxSuccess(transferTxResp)

	packet, err := ibctesting.ParseV1PacketFromEvents(transferTxResp.Events)
	s.Require().NoError(err)

	s.StartRelayer(relayer, testName)

	chainBDenom := testsuite.GetIBCToken(chainA.Config().Denom, channelBFromAB.PortID, channelBFromAB.ChannelID)
	chainCDenom := testsuite.GetIBCToken(chainBDenom.Path(), channelC.PortID, channelC.ChannelID)

	err = test.WaitForCondition(2*time.Minute, 5*time.Second, func() (bool, error) {
		chainCBalance, err := query.Balance(ctx, chainC, receiverOnC.FormattedAddress(), chainCDenom.IBCDenom())
		if err != nil {
			return false, err
		}

		chainBBalance, err := query.Balance(ctx, chainB, receiverOnB.FormattedAddress(), chainBDenom.IBCDenom())
		if err != nil {
			return false, err
		}

		return chainCBalance.Equal(transferAmount.Amount) && chainBBalance.IsZero(), nil
	})
	s.Require().NoError(err)

	chainCBalance, err := query.Balance(ctx, chainC, receiverOnC.FormattedAddress(), chainCDenom.IBCDenom())
	s.Require().NoError(err)
	s.Require().True(chainCBalance.Equal(transferAmount.Amount))

	chainBBalance, err := query.Balance(ctx, chainB, receiverOnB.FormattedAddress(), chainBDenom.IBCDenom())
	s.Require().NoError(err)
	s.Require().True(chainBBalance.IsZero())

	err = test.WaitForCondition(2*time.Minute, 5*time.Second, func() (bool, error) {
		_, err := query.GRPCQuery[channeltypes.QueryPacketCommitmentResponse](ctx, chainA, &channeltypes.QueryPacketCommitmentRequest{
			PortId:    channelA.PortID,
			ChannelId: channelA.ChannelID,
			Sequence:  packet.Sequence,
		})
		if err == nil {
			return false, nil
		}

		if strings.Contains(err.Error(), "packet commitment hash not found") {
			return true, nil
		}

		return false, err
	})
	s.Require().NoError(err)
}
