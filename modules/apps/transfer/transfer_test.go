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

func (suite *TransferTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}

// Constructs the following sends based on the established channels/connections
// 1 - from chainA to chainB
// 2 - from chainB to chainC
// 3 - from chainC to chainB
func (suite *TransferTestSuite) TestHandleMsgTransfer() {
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
				suite.Require().True(ok)
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
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup between chainA and chainB
			// NOTE:
			// pathAToB.EndpointA = endpoint on chainA
			// pathAToB.EndpointB = endpoint on chainB
			pathAToB := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			pathAToB.Setup()
			traceAToB := types.NewHop(pathAToB.EndpointB.ChannelConfig.PortID, pathAToB.EndpointB.ChannelID)

			sourceDenomToTransfer = sdk.DefaultBondDenom
			msgAmount = ibctesting.DefaultCoinAmount

			tc.malleate()

			originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sourceDenomToTransfer)

			timeoutHeight := clienttypes.NewHeight(1, 110)

			originalCoin := sdk.NewCoin(sourceDenomToTransfer, msgAmount)

			// send from chainA to chainB
			msg := types.NewMsgTransfer(pathAToB.EndpointA.ChannelConfig.PortID, pathAToB.EndpointA.ChannelID, originalCoin, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
			res, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err := ibctesting.ParseV1PacketFromEvents(res.Events)
			suite.Require().NoError(err)

			// Get the packet data to determine the amount of tokens being transferred (needed for sending entire balance)
			packetData, err := types.UnmarshalPacketData(packet.GetData(), pathAToB.EndpointA.GetChannel().Version, "")
			suite.Require().NoError(err)
			transferAmount, ok := sdkmath.NewIntFromString(packetData.Token.Amount)
			suite.Require().True(ok)

			// relay send
			err = pathAToB.RelayPacket(packet)
			suite.Require().NoError(err) // relay committed

			escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
			// check that the balance for chainA is updated
			chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			suite.Require().True(originalBalance.Amount.Sub(transferAmount).Equal(chainABalance.Amount))

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			suite.Require().True(transferAmount.Equal(chainAEscrowBalance.Amount))

			// check that voucher exists on chain B
			chainBDenom := types.NewDenom(originalCoin.Denom, traceAToB)
			chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), chainBDenom.IBCDenom())
			coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), transferAmount)
			suite.Require().Equal(coinSentFromAToB, chainBBalance)

			// setup between chainB to chainC
			// NOTE:
			// pathBToC.EndpointA = endpoint on chainB
			// pathBToC.EndpointB = endpoint on chainC
			pathBToC := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
			pathBToC.Setup()
			traceBToC := types.NewHop(pathBToC.EndpointB.ChannelConfig.PortID, pathBToC.EndpointB.ChannelID)

			// send from chainB to chainC
			msg = types.NewMsgTransfer(pathBToC.EndpointA.ChannelConfig.PortID, pathBToC.EndpointA.ChannelID, coinSentFromAToB, suite.chainB.SenderAccount.GetAddress().String(), suite.chainC.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
			res, err = suite.chainB.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err = ibctesting.ParseV1PacketFromEvents(res.Events)
			suite.Require().NoError(err)

			err = pathBToC.RelayPacket(packet)
			suite.Require().NoError(err) // relay committed

			coinsSentFromBToC := sdk.NewCoins()
			// check balances for chainB and chainC after transfer from chainB to chainC
			// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
			chainCDenom := types.NewDenom(originalCoin.Denom, traceBToC, traceAToB)

			// check that the balance is updated on chainC
			coinSentFromBToC := sdk.NewCoin(chainCDenom.IBCDenom(), transferAmount)
			chainCBalance := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
			suite.Require().Equal(coinSentFromBToC, chainCBalance)

			// check that balance on chain B is empty
			chainBBalance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
			suite.Require().Zero(chainBBalance.Amount.Int64())

			// send from chainC back to chainB
			msg = types.NewMsgTransfer(pathBToC.EndpointB.ChannelConfig.PortID, pathBToC.EndpointB.ChannelID, coinSentFromBToC, suite.chainC.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
			res, err = suite.chainC.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err = ibctesting.ParseV1PacketFromEvents(res.Events)
			suite.Require().NoError(err)

			err = pathBToC.RelayPacket(packet)
			suite.Require().NoError(err) // relay committed

			// check balances for chainC are empty after transfer from chainC to chainB
			for _, coin := range coinsSentFromBToC {
				// check that balance on chain C is empty
				chainCBalance := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), coin.Denom)
				suite.Require().Zero(chainCBalance.Amount.Int64())
			}

			// check balances for chainB after transfer from chainC to chainB
			// check that balance on chain B has the transferred amount
			chainBBalance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromAToB.Denom)
			suite.Require().Equal(coinSentFromAToB, chainBBalance)

			// check that module account escrow address is empty
			escrowAddress = types.GetEscrowAddress(traceBToC.PortId, traceBToC.ChannelId)
			chainBEscrowBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddress, coinSentFromAToB.Denom)
			suite.Require().Zero(chainBEscrowBalance.Amount.Int64())

			// check balances for chainA after transfer from chainC to chainB
			// check that the balance is unchanged
			chainABalance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			suite.Require().True(originalBalance.Amount.Sub(transferAmount).Equal(chainABalance.Amount))

			// check that module account escrow address is unchanged
			escrowAddress = types.GetEscrowAddress(pathAToB.EndpointA.ChannelConfig.PortID, pathAToB.EndpointA.ChannelID)
			chainAEscrowBalance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			suite.Require().True(transferAmount.Equal(chainAEscrowBalance.Amount))
		})
	}
}

func TestTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuite))
}
