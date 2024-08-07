package transfer_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
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
	testCases := []struct {
		name                   string
		sourceDenomsToTransfer []string
	}{
		{
			"transfer single denom",
			[]string{sdk.DefaultBondDenom},
		},
		{
			"transfer multiple denoms",
			[]string{sdk.DefaultBondDenom, ibctesting.SecondaryDenom},
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

			originalBalances := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom)
				originalBalances = originalBalances.Add(originalBalance)
			}

			timeoutHeight := clienttypes.NewHeight(1, 110)

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			suite.Require().True(ok)
			originalCoins := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				coinToSendToB := sdk.NewCoin(denom, amount)
				originalCoins = originalCoins.Add(coinToSendToB)
			}

			// send from chainA to chainB
			msg := types.NewMsgTransfer(pathAToB.EndpointA.ChannelConfig.PortID, pathAToB.EndpointA.ChannelID, originalCoins, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "", nil)
			res, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err := ibctesting.ParsePacketFromEvents(res.Events)
			suite.Require().NoError(err)

			// relay send
			err = pathAToB.RelayPacket(packet)
			suite.Require().NoError(err) // relay committed

			escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
			coinsSentFromAToB := sdk.NewCoins()
			for _, coin := range originalCoins {
				// check that the balance for chainA is updated
				chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
				suite.Require().Equal(originalBalances.AmountOf(coin.Denom).Sub(amount).Int64(), chainABalance.Amount.Int64())

				// check that module account escrow address has locked the tokens
				chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
				suite.Require().Equal(coin, chainAEscrowBalance)

				// check that voucher exists on chain B
				chainBDenom := types.NewDenom(coin.Denom, traceAToB)
				chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), chainBDenom.IBCDenom())
				coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), amount)
				suite.Require().Equal(coinSentFromAToB, chainBBalance)

				coinsSentFromAToB = coinsSentFromAToB.Add(coinSentFromAToB)
			}

			// setup between chainB to chainC
			// NOTE:
			// pathBToC.EndpointA = endpoint on chainB
			// pathBToC.EndpointB = endpoint on chainC
			pathBToC := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
			pathBToC.Setup()
			traceBToC := types.NewHop(pathBToC.EndpointB.ChannelConfig.PortID, pathBToC.EndpointB.ChannelID)

			// send from chainB to chainC
			msg = types.NewMsgTransfer(pathBToC.EndpointA.ChannelConfig.PortID, pathBToC.EndpointA.ChannelID, coinsSentFromAToB, suite.chainB.SenderAccount.GetAddress().String(), suite.chainC.SenderAccount.GetAddress().String(), timeoutHeight, 0, "", nil)
			res, err = suite.chainB.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err = ibctesting.ParsePacketFromEvents(res.Events)
			suite.Require().NoError(err)

			err = pathBToC.RelayPacket(packet)
			suite.Require().NoError(err) // relay committed

			coinsSentFromBToC := sdk.NewCoins()
			// check balances for chainB and chainC after transfer from chainB to chainC
			for _, coin := range originalCoins {
				// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
				chainCDenom := types.NewDenom(coin.Denom, traceBToC, traceAToB)

				// check that the balance is updated on chainC
				coinSentFromBToC := sdk.NewCoin(chainCDenom.IBCDenom(), amount)
				chainCBalance := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
				suite.Require().Equal(coinSentFromBToC, chainCBalance)

				// check that balance on chain B is empty
				chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
				suite.Require().Zero(chainBBalance.Amount.Int64())

				coinsSentFromBToC = coinsSentFromBToC.Add(coinSentFromBToC)
			}

			// send from chainC back to chainB
			msg = types.NewMsgTransfer(pathBToC.EndpointB.ChannelConfig.PortID, pathBToC.EndpointB.ChannelID, coinsSentFromBToC, suite.chainC.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "", nil)
			res, err = suite.chainC.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err = ibctesting.ParsePacketFromEvents(res.Events)
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
			for _, coin := range coinsSentFromAToB {
				// check that balance on chain B has the transferred amount
				chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coin.Denom)
				suite.Require().Equal(coin, chainBBalance)

				// check that module account escrow address is empty
				escrowAddress = types.GetEscrowAddress(traceBToC.PortId, traceBToC.ChannelId)
				chainBEscrowBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddress, coin.Denom)
				suite.Require().Zero(chainBEscrowBalance.Amount.Int64())
			}

			// check balances for chainA after transfer from chainC to chainB
			for _, coin := range originalCoins {
				// check that the balance is unchanged
				chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
				suite.Require().Equal(originalBalances.AmountOf(coin.Denom).Sub(amount).Int64(), chainABalance.Amount.Int64())

				// check that module account escrow address is unchanged
				escrowAddress = types.GetEscrowAddress(pathAToB.EndpointA.ChannelConfig.PortID, pathAToB.EndpointA.ChannelID)
				chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
				suite.Require().Equal(coin, chainAEscrowBalance)
			}
		})
	}
}

func TestTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuite))
}
