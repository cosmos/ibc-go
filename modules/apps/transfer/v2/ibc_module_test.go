package v2_test

import (
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

type TransferTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain

	pathAToB *ibctesting.Path
	pathBToC *ibctesting.Path
}

const invalidPortID = "invalidportid"

func (suite *TransferTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))

	// setup between chainA and chainB
	// NOTE:
	// pathAToB.EndpointA = endpoint on chainA
	// pathAToB.EndpointB = endpoint on chainB
	suite.pathAToB = ibctesting.NewPath(suite.chainA, suite.chainB)

	// setup between chainB and chainC
	// pathBToC.EndpointA = endpoint on chainB
	// pathBToC.EndpointB = endpoint on chainC
	suite.pathBToC = ibctesting.NewPath(suite.chainB, suite.chainC)

	// setup eureka paths between the chains
	suite.pathAToB.SetupV2()
	suite.pathBToC.SetupV2()
}

func TestTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuite))
}

func (suite *TransferTestSuite) TestOnSendPacket() {
	var payload channeltypesv2.Payload
	testCases := []struct {
		name                   string
		sourceDenomsToTransfer []string
		malleate               func()
		expError               error
	}{
		{
			"transfer single denom",
			[]string{sdk.DefaultBondDenom},
			func() {},
			nil,
		},
		{
			"transfer multiple denoms",
			[]string{sdk.DefaultBondDenom, ibctesting.SecondaryDenom},
			func() {},
			nil,
		},
		{
			"transfer with invalid source port",
			[]string{sdk.DefaultBondDenom},
			func() {
				payload.SourcePort = invalidPortID
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"transfer with invalid destination port",
			[]string{sdk.DefaultBondDenom},
			func() {
				payload.DestinationPort = invalidPortID
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"transfer with slashes in base denom",
			[]string{"base/coin"},
			func() {},
			types.ErrInvalidDenomForTransfer,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			originalBalances := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom)
				originalBalances = originalBalances.Add(originalBalance)
			}

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			suite.Require().True(ok)
			originalCoins := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				coinToSendToB := sdk.NewCoin(denom, amount)
				originalCoins = originalCoins.Add(coinToSendToB)
			}

			tokens := make([]types.Token, len(originalCoins))

			for i, coin := range originalCoins {
				var err error
				tokens[i], err = suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(suite.chainA.GetContext(), coin)
				suite.Require().NoError(err)
			}

			transferData := types.NewFungibleTokenPacketDataV2(
				tokens,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				"",
				types.ForwardingPacketData{},
			)
			bz := suite.chainA.Codec.MustMarshal(&transferData)
			payload = channeltypesv2.NewPayload(
				types.PortID, types.PortID, types.V2,
				types.EncodingProtobuf, bz,
			)

			// malleate payload
			tc.malleate()

			ctx := suite.chainA.GetContext()
			cbs := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			err := cbs.OnSendPacket(ctx, suite.pathAToB.EndpointA.ClientID, suite.pathAToB.EndpointB.ClientID, 1, payload, suite.chainA.SenderAccount.GetAddress())

			if tc.expError != nil {
				suite.Require().Contains(err.Error(), tc.expError.Error())
			} else {
				suite.Require().NoError(err)

				escrowAddress := types.GetEscrowAddress(types.PortID, suite.pathAToB.EndpointA.ClientID)
				for _, coin := range originalCoins {
					// check that the balance for chainA is updated
					chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
					suite.Require().Equal(originalBalances.AmountOf(coin.Denom).Sub(amount).Int64(), chainABalance.Amount.Int64())

					// check that module account escrow address has locked the tokens
					chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
					suite.Require().Equal(coin, chainAEscrowBalance)

				}
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnRecvPacket() {
	var payload channeltypesv2.Payload
	testCases := []struct {
		name                   string
		sourceDenomsToTransfer []string
		malleate               func()
		expErr                 bool
	}{
		{
			"transfer single denom",
			[]string{sdk.DefaultBondDenom},
			func() {},
			false,
		},
		{
			"transfer multiple denoms",
			[]string{sdk.DefaultBondDenom, ibctesting.SecondaryDenom},
			func() {},
			false,
		},
		{
			"transfer with invalid source port",
			[]string{sdk.DefaultBondDenom},
			func() {
				payload.SourcePort = invalidPortID
			},
			true,
		},
		{
			"transfer with invalid dest port",
			[]string{sdk.DefaultBondDenom},
			func() {
				payload.DestinationPort = invalidPortID
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			originalBalances := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom)
				originalBalances = originalBalances.Add(originalBalance)
			}

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			suite.Require().True(ok)
			originalCoins := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				coinToSendToB := sdk.NewCoin(denom, amount)
				originalCoins = originalCoins.Add(coinToSendToB)
			}

			tokens := make([]types.Token, len(originalCoins))

			for i, coin := range originalCoins {
				var err error
				tokens[i], err = suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(suite.chainA.GetContext(), coin)
				suite.Require().NoError(err)
			}

			transferData := types.NewFungibleTokenPacketDataV2(
				tokens,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				"",
				types.ForwardingPacketData{},
			)
			bz := suite.chainA.Codec.MustMarshal(&transferData)
			payload = channeltypesv2.NewPayload(
				types.PortID, types.PortID, types.V2,
				types.EncodingProtobuf, bz,
			)
			msg := channeltypesv2.NewMsgSendPacket(
				suite.pathAToB.EndpointA.ClientID,
				timeoutTimestamp,
				suite.chainA.SenderAccount.GetAddress().String(),
				payload,
			)

			_, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			ctx := suite.chainB.GetContext()
			cbs := suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			// malleate payload after it has been sent but before OnRecvPacket callback is called
			tc.malleate()

			recvResult := cbs.OnRecvPacket(
				ctx, suite.pathAToB.EndpointA.ClientID, suite.pathAToB.EndpointB.ClientID,
				1, payload, suite.chainB.SenderAccount.GetAddress(),
			)

			if !tc.expErr {

				suite.Require().Equal(channeltypesv2.PacketStatus_Success, recvResult.Status)
				suite.Require().Equal(channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(), recvResult.Acknowledgement)

				escrowAddress := types.GetEscrowAddress(types.PortID, suite.pathAToB.EndpointA.ClientID)
				for _, coin := range originalCoins {
					// check that the balance for chainA is updated
					chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
					suite.Require().Equal(originalBalances.AmountOf(coin.Denom).Sub(amount).Int64(), chainABalance.Amount.Int64())

					// check that module account escrow address has locked the tokens
					chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
					suite.Require().Equal(coin, chainAEscrowBalance)

					traceAToB := types.NewHop(types.PortID, suite.pathAToB.EndpointB.ClientID)

					// check that voucher exists on chain B
					chainBDenom := types.NewDenom(coin.Denom, traceAToB)
					chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), chainBDenom.IBCDenom())
					coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), amount)
					suite.Require().Equal(coinSentFromAToB, chainBBalance)

				}
			} else {
				suite.Require().Equal(channeltypesv2.PacketStatus_Failure, recvResult.Status)
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnAckPacket() {
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

			originalBalances := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom)
				originalBalances = originalBalances.Add(originalBalance)
			}

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			suite.Require().True(ok)
			originalCoins := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				coinToSendToB := sdk.NewCoin(denom, amount)
				originalCoins = originalCoins.Add(coinToSendToB)
			}

			tokens := make([]types.Token, len(originalCoins))

			for i, coin := range originalCoins {
				var err error
				tokens[i], err = suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(suite.chainA.GetContext(), coin)
				suite.Require().NoError(err)
			}

			transferData := types.NewFungibleTokenPacketDataV2(
				tokens,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				"",
				types.ForwardingPacketData{},
			)
			bz := suite.chainA.Codec.MustMarshal(&transferData)
			payload := channeltypesv2.NewPayload(
				types.PortID, types.PortID, types.V2,
				types.EncodingProtobuf, bz,
			)
			msg := channeltypesv2.NewMsgSendPacket(
				suite.pathAToB.EndpointA.ClientID,
				timeoutTimestamp,
				suite.chainA.SenderAccount.GetAddress().String(),
				payload,
			)

			_, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			ctx := suite.chainA.GetContext()
			cbs := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

			err = cbs.OnAcknowledgementPacket(
				ctx, suite.pathAToB.EndpointA.ClientID, suite.pathAToB.EndpointB.ClientID,
				1, ack.Acknowledgement(), payload, suite.chainA.SenderAccount.GetAddress(),
			)
			suite.Require().NoError(err)

			// on successful ack, the tokens sent in packets should still be in escrow
			escrowAddress := types.GetEscrowAddress(types.PortID, suite.pathAToB.EndpointA.ClientID)
			for _, coin := range originalCoins {
				// check that the balance for chainA is updated
				chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
				suite.Require().Equal(originalBalances.AmountOf(coin.Denom).Sub(amount).Int64(), chainABalance.Amount.Int64())

				// check that module account escrow address has locked the tokens
				chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
				suite.Require().Equal(coin, chainAEscrowBalance)
			}

			// create a custom error ack and replay the callback to ensure it fails with IBC v2 callbacks
			errAck := channeltypes.NewErrorAcknowledgement(types.ErrInvalidAmount)
			err = cbs.OnAcknowledgementPacket(
				ctx, suite.pathAToB.EndpointA.ClientID, suite.pathAToB.EndpointB.ClientID,
				1, errAck.Acknowledgement(), payload, suite.chainA.SenderAccount.GetAddress(),
			)
			suite.Require().Error(err)

			// create the sentinel error ack and replay the callback to ensure the tokens are correctly refunded
			// we can replay the callback here because the replay protection is handled in the IBC handler
			err = cbs.OnAcknowledgementPacket(
				ctx, suite.pathAToB.EndpointA.ClientID, suite.pathAToB.EndpointB.ClientID,
				1, channeltypesv2.ErrorAcknowledgement[:], payload, suite.chainA.SenderAccount.GetAddress(),
			)
			suite.Require().NoError(err)

			// on error ack, the tokens sent in packets should be returned to sender
			for _, coin := range originalCoins {
				// check that the balance for chainA is refunded
				chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
				suite.Require().Equal(originalBalances.AmountOf(coin.Denom), chainABalance.Amount)

				// check that module account escrow address has no tokens
				chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
				suite.Require().Equal(sdk.NewCoin(coin.Denom, sdkmath.ZeroInt()), chainAEscrowBalance)
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnTimeoutPacket() {
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

			originalBalances := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom)
				originalBalances = originalBalances.Add(originalBalance)
			}

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			suite.Require().True(ok)
			originalCoins := sdk.NewCoins()
			for _, denom := range tc.sourceDenomsToTransfer {
				coinToSendToB := sdk.NewCoin(denom, amount)
				originalCoins = originalCoins.Add(coinToSendToB)
			}

			tokens := make([]types.Token, len(originalCoins))

			for i, coin := range originalCoins {
				var err error
				tokens[i], err = suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(suite.chainA.GetContext(), coin)
				suite.Require().NoError(err)
			}

			transferData := types.NewFungibleTokenPacketDataV2(
				tokens,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				"",
				types.ForwardingPacketData{},
			)
			bz := suite.chainA.Codec.MustMarshal(&transferData)
			payload := channeltypesv2.NewPayload(
				types.PortID, types.PortID, types.V2,
				types.EncodingProtobuf, bz,
			)
			msg := channeltypesv2.NewMsgSendPacket(
				suite.pathAToB.EndpointA.ClientID,
				timeoutTimestamp,
				suite.chainA.SenderAccount.GetAddress().String(),
				payload,
			)

			_, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			// on successful send, the tokens sent in packets should be in escrow
			escrowAddress := types.GetEscrowAddress(types.PortID, suite.pathAToB.EndpointA.ClientID)
			for _, coin := range originalCoins {
				// check that the balance for chainA is updated
				chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
				suite.Require().Equal(originalBalances.AmountOf(coin.Denom).Sub(amount).Int64(), chainABalance.Amount.Int64())

				// check that module account escrow address has locked the tokens
				chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
				suite.Require().Equal(coin, chainAEscrowBalance)
			}

			ctx := suite.chainA.GetContext()
			cbs := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			err = cbs.OnTimeoutPacket(
				ctx, suite.pathAToB.EndpointA.ClientID, suite.pathAToB.EndpointB.ClientID,
				1, payload, suite.chainA.SenderAccount.GetAddress(),
			)
			suite.Require().NoError(err)

			// on timeout, the tokens sent in packets should be returned to sender
			for _, coin := range originalCoins {
				// check that the balance for chainA is refunded
				chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
				suite.Require().Equal(originalBalances.AmountOf(coin.Denom), chainABalance.Amount)

				// check that module account escrow address has no tokens
				chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
				suite.Require().Equal(sdk.NewCoin(coin.Denom, sdkmath.ZeroInt()), chainAEscrowBalance)
			}
		})
	}
}
