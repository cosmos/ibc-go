package transfer_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type TransferTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func TestTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuite))
}

func (s *TransferTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))
}

// Constructs the following sends based on the established channels/connections
// 1 - from chainA to chainB
// 2 - from chainB to chainC
// 3 - from chainC to chainB
func (s *TransferTestSuite) TestHandleMsgTransfer() {
	var (
		sourceDenomToTransfer string
		msgAmount             sdkmath.Int
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"transfer single denom",
			func() {},
		},
		{
			"transfer amount larger than int64",
			func() {
				var ok bool
				msgAmount, ok = sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
				s.Require().True(ok)
			},
		},
		{
			"transfer entire balance",
			func() {
				msgAmount = types.UnboundedSpendLimit()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			// setup between chainA and chainB
			// NOTE:
			// pathAToB.EndpointA = endpoint on chainA
			// pathAToB.EndpointB = endpoint on chainB
			pathAToB := ibctesting.NewTransferPath(s.chainA, s.chainB)
			pathAToB.Setup()
			traceAToB := types.NewHop(pathAToB.EndpointB.ChannelConfig.PortID, pathAToB.EndpointB.ChannelID)

			sourceDenomToTransfer = sdk.DefaultBondDenom
			msgAmount = ibctesting.DefaultCoinAmount

			tc.malleate()

			originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sourceDenomToTransfer)

			timeoutHeight := clienttypes.NewHeight(1, 110)

			originalCoin := sdk.NewCoin(sourceDenomToTransfer, msgAmount)

			// send from chainA to chainB
			msg := types.NewMsgTransfer(pathAToB.EndpointA.ChannelConfig.PortID, pathAToB.EndpointA.ChannelID, originalCoin, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
			res, err := s.chainA.SendMsgs(msg)
			s.Require().NoError(err) // message committed

			packet, err := ibctesting.ParseV1PacketFromEvents(res.Events)
			s.Require().NoError(err)

			// Get the packet data to determine the amount of tokens being transferred (needed for sending entire balance)
			packetData, err := types.UnmarshalPacketData(packet.GetData(), pathAToB.EndpointA.GetChannel().Version, "")
			s.Require().NoError(err)
			transferAmount, ok := sdkmath.NewIntFromString(packetData.Token.Amount)
			s.Require().True(ok)

			// relay send
			err = pathAToB.RelayPacket(packet)
			s.Require().NoError(err) // relay committed

			escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
			// check that the balance for chainA is updated
			chainABalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			s.Require().True(originalBalance.Amount.Sub(transferAmount).Equal(chainABalance.Amount))

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			s.Require().True(transferAmount.Equal(chainAEscrowBalance.Amount))

			// check that voucher exists on chain B
			chainBDenom := types.NewDenom(originalCoin.Denom, traceAToB)
			chainBBalance := s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), chainBDenom.IBCDenom())
			coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), transferAmount)
			s.Require().Equal(coinSentFromAToB, chainBBalance)

			// setup between chainB to chainC
			// NOTE:
			// pathBToC.EndpointA = endpoint on chainB
			// pathBToC.EndpointB = endpoint on chainC
			pathBToC := ibctesting.NewTransferPath(s.chainB, s.chainC)
			pathBToC.Setup()
			traceBToC := types.NewHop(pathBToC.EndpointB.ChannelConfig.PortID, pathBToC.EndpointB.ChannelID)

			// send from chainB to chainC
			msg = types.NewMsgTransfer(pathBToC.EndpointA.ChannelConfig.PortID, pathBToC.EndpointA.ChannelID, coinSentFromAToB, s.chainB.SenderAccount.GetAddress().String(), s.chainC.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
			res, err = s.chainB.SendMsgs(msg)
			s.Require().NoError(err) // message committed

			packet, err = ibctesting.ParseV1PacketFromEvents(res.Events)
			s.Require().NoError(err)

			err = pathBToC.RelayPacket(packet)
			s.Require().NoError(err) // relay committed

			coinsSentFromBToC := sdk.NewCoins()
			// check balances for chainB and chainC after transfer from chainB to chainC
			// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
			chainCDenom := types.NewDenom(originalCoin.Denom, traceBToC, traceAToB)

			// check that the balance is updated on chainC
			coinSentFromBToC := sdk.NewCoin(chainCDenom.IBCDenom(), transferAmount)
			chainCBalance := s.chainC.GetSimApp().BankKeeper.GetBalance(s.chainC.GetContext(), s.chainC.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
			s.Require().Equal(coinSentFromBToC, chainCBalance)

			// check that balance on chain B is empty
			chainBBalance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
			s.Require().Zero(chainBBalance.Amount.Int64())

			// send from chainC back to chainB
			msg = types.NewMsgTransfer(pathBToC.EndpointB.ChannelConfig.PortID, pathBToC.EndpointB.ChannelID, coinSentFromBToC, s.chainC.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
			res, err = s.chainC.SendMsgs(msg)
			s.Require().NoError(err) // message committed

			packet, err = ibctesting.ParseV1PacketFromEvents(res.Events)
			s.Require().NoError(err)

			err = pathBToC.RelayPacket(packet)
			s.Require().NoError(err) // relay committed

			// check balances for chainC are empty after transfer from chainC to chainB
			for _, coin := range coinsSentFromBToC {
				// check that balance on chain C is empty
				chainCBalance := s.chainC.GetSimApp().BankKeeper.GetBalance(s.chainC.GetContext(), s.chainC.SenderAccount.GetAddress(), coin.Denom)
				s.Require().Zero(chainCBalance.Amount.Int64())
			}

			// check balances for chainB after transfer from chainC to chainB
			// check that balance on chain B has the transferred amount
			chainBBalance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), coinSentFromAToB.Denom)
			s.Require().Equal(coinSentFromAToB, chainBBalance)

			// check that module account escrow address is empty
			escrowAddress = types.GetEscrowAddress(traceBToC.PortId, traceBToC.ChannelId)
			chainBEscrowBalance := s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), escrowAddress, coinSentFromAToB.Denom)
			s.Require().Zero(chainBEscrowBalance.Amount.Int64())

			// check balances for chainA after transfer from chainC to chainB
			// check that the balance is unchanged
			chainABalance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			s.Require().True(originalBalance.Amount.Sub(transferAmount).Equal(chainABalance.Amount))

			// check that module account escrow address is unchanged
			escrowAddress = types.GetEscrowAddress(pathAToB.EndpointA.ChannelConfig.PortID, pathAToB.EndpointA.ChannelID)
			chainAEscrowBalance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			s.Require().True(transferAmount.Equal(chainAEscrowBalance.Amount))
		})
	}
}
