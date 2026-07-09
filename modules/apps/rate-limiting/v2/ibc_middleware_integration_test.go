package v2_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ratelimittypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

const rateLimitChannelValue = int64(1000)

type RateLimitMiddlewareTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	path        *ibctesting.Path
}

func TestRateLimitMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(RateLimitMiddlewareTestSuite))
}

func (s *RateLimitMiddlewareTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.path = ibctesting.NewPath(s.chainA, s.chainB)
	s.path.SetupV2()
}

func (s *RateLimitMiddlewareTestSuite) TestV2TransferSuccessUpdatesFlows() {
	amount := sdkmath.NewInt(10)
	recvDenom := s.voucherDenom()

	s.setRateLimit(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 100, 100)
	s.setRateLimit(s.chainB, recvDenom, s.path.EndpointB.ClientID, 100, 100)

	senderInitialBalance := s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	payload := s.transferPayload(amount)

	packet, err := s.path.EndpointA.MsgSendPacket(s.timeoutTimestamp(time.Hour), payload)
	s.Require().NoError(err)

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), amount)
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, true)

	ack, err := s.path.EndpointB.MsgRecvPacketWithAck(packet)
	s.Require().NoError(err)
	s.Require().Len(ack.AppAcknowledgements, 1)
	s.Require().Equal(channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(), ack.AppAcknowledgements[0])

	s.assertFlow(s.chainB, recvDenom, s.path.EndpointB.ClientID, amount, sdkmath.ZeroInt())
	s.Require().Equal(amount, s.balance(s.chainB, s.chainB.SenderAccount.GetAddress(), recvDenom).Amount)

	err = s.path.EndpointA.MsgAcknowledgePacket(packet, ack)
	s.Require().NoError(err)

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), amount)
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, false)
	s.Require().Equal(senderInitialBalance.Amount.Sub(amount), s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom).Amount)
}

func (s *RateLimitMiddlewareTestSuite) TestV2TransferSendDenied() {
	amount := sdkmath.NewInt(11)
	s.setRateLimit(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 1, 100)

	senderInitialBalance := s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	payload := s.transferPayload(amount)

	_, err := s.path.EndpointA.MsgSendPacket(s.timeoutTimestamp(time.Hour), payload)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), ratelimittypes.ErrQuotaExceeded.Error())

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 1, false)
	s.Require().Equal(senderInitialBalance, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom))
}

func (s *RateLimitMiddlewareTestSuite) TestV2TransferReceiveDeniedUndoSendOnErrorAck() {
	amount := sdkmath.NewInt(11)
	recvDenom := s.voucherDenom()

	s.setRateLimit(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 100, 100)
	s.setRateLimit(s.chainB, recvDenom, s.path.EndpointB.ClientID, 100, 1)

	senderInitialBalance := s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	payload := s.transferPayload(amount)

	packet, err := s.path.EndpointA.MsgSendPacket(s.timeoutTimestamp(time.Hour), payload)
	s.Require().NoError(err)
	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), amount)
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, true)

	ack, err := s.path.EndpointB.MsgRecvPacketWithAck(packet)
	s.Require().NoError(err)
	s.Require().Len(ack.AppAcknowledgements, 1)
	s.Require().Equal(channeltypesv2.ErrorAcknowledgement[:], ack.AppAcknowledgements[0])
	s.assertFlow(s.chainB, recvDenom, s.path.EndpointB.ClientID, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	s.Require().True(s.balance(s.chainB, s.chainB.SenderAccount.GetAddress(), recvDenom).IsZero())

	err = s.path.EndpointA.MsgAcknowledgePacket(packet, ack)
	s.Require().NoError(err)

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, false)
	s.Require().Equal(senderInitialBalance, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom))
}

func (s *RateLimitMiddlewareTestSuite) TestV2TransferTimeoutUndoSend() {
	amount := sdkmath.NewInt(10)
	s.setRateLimit(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 100, 100)

	senderInitialBalance := s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	payload := s.transferPayload(amount)

	packet, err := s.path.EndpointA.MsgSendPacket(s.timeoutTimestamp(time.Second), payload)
	s.Require().NoError(err)
	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), amount)
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, true)
	s.Require().Equal(senderInitialBalance.Amount.Sub(amount), s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom).Amount)

	s.Require().NoError(s.path.EndpointA.UpdateClient())
	err = s.path.EndpointA.MsgTimeoutPacket(packet)
	s.Require().NoError(err)

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, false)
	s.Require().Equal(senderInitialBalance, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom))
}

func (s *RateLimitMiddlewareTestSuite) setRateLimit(chain *ibctesting.TestChain, denom, clientID string, maxPercentSend, maxPercentRecv int64) {
	chain.GetSimApp().RateLimitKeeper.SetRateLimit(chain.GetContext(), ratelimittypes.RateLimit{
		Path: &ratelimittypes.Path{
			Denom:             denom,
			ChannelOrClientId: clientID,
		},
		Quota: &ratelimittypes.Quota{
			MaxPercentSend: sdkmath.NewInt(maxPercentSend),
			MaxPercentRecv: sdkmath.NewInt(maxPercentRecv),
			DurationHours:  1,
		},
		Flow: &ratelimittypes.Flow{
			Inflow:       sdkmath.ZeroInt(),
			Outflow:      sdkmath.ZeroInt(),
			ChannelValue: sdkmath.NewInt(rateLimitChannelValue),
		},
	})
}

func (s *RateLimitMiddlewareTestSuite) transferPayload(amount sdkmath.Int) channeltypesv2.Payload {
	packetData := transfertypes.NewFungibleTokenPacketData(
		sdk.DefaultBondDenom,
		amount.String(),
		s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(),
		"",
	)
	bz := s.chainA.Codec.MustMarshal(&packetData)

	return channeltypesv2.NewPayload(transfertypes.PortID, transfertypes.PortID, transfertypes.V1, transfertypes.EncodingProtobuf, bz)
}

func (s *RateLimitMiddlewareTestSuite) voucherDenom() string {
	return transfertypes.NewDenom(
		sdk.DefaultBondDenom,
		transfertypes.NewHop(transfertypes.PortID, s.path.EndpointB.ClientID),
	).IBCDenom()
}

func (s *RateLimitMiddlewareTestSuite) timeoutTimestamp(duration time.Duration) uint64 {
	return uint64(s.chainB.GetContext().BlockTime().Add(duration).Unix())
}

func (s *RateLimitMiddlewareTestSuite) assertFlow(chain *ibctesting.TestChain, denom, clientID string, expectedInflow, expectedOutflow sdkmath.Int) {
	rateLimit, found := chain.GetSimApp().RateLimitKeeper.GetRateLimit(chain.GetContext(), denom, clientID)
	s.Require().True(found)
	s.Require().True(rateLimit.Flow.Inflow.Equal(expectedInflow), "expected inflow %s, got %s", expectedInflow, rateLimit.Flow.Inflow)
	s.Require().True(rateLimit.Flow.Outflow.Equal(expectedOutflow), "expected outflow %s, got %s", expectedOutflow, rateLimit.Flow.Outflow)
}

func (s *RateLimitMiddlewareTestSuite) assertPendingPacket(chain *ibctesting.TestChain, denom, clientID string, sequence uint64, expected bool) {
	found, err := chain.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(chain.GetContext(), clientID, sequence, denom)
	s.Require().NoError(err)
	s.Require().Equal(expected, found)
}

func (s *RateLimitMiddlewareTestSuite) balance(chain *ibctesting.TestChain, address sdk.AccAddress, denom string) sdk.Coin {
	return chain.GetSimApp().BankKeeper.GetBalance(chain.GetContext(), address, denom)
}
