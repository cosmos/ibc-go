package transfer_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type TransferTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (s *TransferTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))
}

func NewTransferPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointA.ChannelConfig.Version = types.Version
	path.EndpointB.ChannelConfig.Version = types.Version

	return path
}

// Constructs the following sends based on the established channels/connections
// 1 - from chainA to chainB
// 2 - from chainB to chainC
// 3 - from chainC to chainB
func (s *TransferTestSuite) TestHandleMsgTransfer() {
	// setup between chainA and chainB
	// NOTE:
	// pathAtoB.EndpointA = endpoint on chainA
	// pathAtoB.EndpointB = endpoint on chainB
	pathAtoB := NewTransferPath(s.chainA, s.chainB)
	s.coordinator.Setup(pathAtoB)

	originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	timeoutHeight := clienttypes.NewHeight(1, 110)

	amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
	s.Require().True(ok)
	coinToSendToB := sdk.NewCoin(sdk.DefaultBondDenom, amount)

	// send from chainA to chainB
	msg := types.NewMsgTransfer(pathAtoB.EndpointA.ChannelConfig.PortID, pathAtoB.EndpointA.ChannelID, coinToSendToB, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	s.Require().NoError(err)

	// relay send
	err = pathAtoB.RelayPacket(packet)
	s.Require().NoError(err) // relay committed

	// check that module account escrow address has locked the tokens
	escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
	balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	s.Require().Equal(coinToSendToB, balance)

	// check that voucher exists on chain B
	voucherDenomTrace := types.ParseDenomTrace(types.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))
	balance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	coinSentFromAToB := types.GetTransferCoin(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID, sdk.DefaultBondDenom, amount)
	s.Require().Equal(coinSentFromAToB, balance)

	// setup between chainB to chainC
	// NOTE:
	// pathBtoC.EndpointA = endpoint on chainB
	// pathBtoC.EndpointB = endpoint on chainC
	pathBtoC := NewTransferPath(s.chainB, s.chainC)
	s.coordinator.Setup(pathBtoC)

	// send from chainB to chainC
	msg = types.NewMsgTransfer(pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID, coinSentFromAToB, s.chainB.SenderAccount.GetAddress().String(), s.chainC.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
	res, err = s.chainB.SendMsgs(msg)
	s.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
	s.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	s.Require().NoError(err) // relay committed

	// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
	fullDenomPath := types.GetPrefixedDenom(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID, voucherDenomTrace.GetFullDenomPath())

	// check that the balance is updated on chainC
	coinSentFromBToC := sdk.NewCoin(types.ParseDenomTrace(fullDenomPath).IBCDenom(), amount)
	balance = s.chainC.GetSimApp().BankKeeper.GetBalance(s.chainC.GetContext(), s.chainC.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
	s.Require().Equal(coinSentFromBToC, balance)

	// check that balance on chain B is empty
	balance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
	s.Require().Zero(balance.Amount.Int64())

	// send from chainC back to chainB
	msg = types.NewMsgTransfer(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID, coinSentFromBToC, s.chainC.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
	res, err = s.chainC.SendMsgs(msg)
	s.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
	s.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	s.Require().NoError(err) // relay committed

	// check that balance on chain A is updated
	balance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	s.Require().Equal(originalBalance.SubAmount(amount).Amount.Int64(), balance.Amount.Int64())

	// check that balance on chain B has the transferred amount
	balance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), coinSentFromAToB.Denom)
	s.Require().Equal(coinSentFromAToB, balance)

	// check that module account escrow address is empty
	escrowAddress = types.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
	balance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), escrowAddress, coinSentFromAToB.Denom)
	s.Require().Zero(balance.Amount.Int64())

	// check that balance on chain C is empty
	balance = s.chainC.GetSimApp().BankKeeper.GetBalance(s.chainC.GetContext(), s.chainC.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	s.Require().Zero(balance.Amount.Int64())
}

func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}
