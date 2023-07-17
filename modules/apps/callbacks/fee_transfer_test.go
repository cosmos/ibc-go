package ibccallbacks_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	feetypes "github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

var (
	defaultRecvFee    = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}}
	defaultAckFee     = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(200)}}
	defaultTimeoutFee = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(300)}}
)

func (suite *CallbacksTestSuite) TestIncentivizedTransferCallbacks() {
	testCases := []struct {
		name            string
		transferMemo    string
		expCallbackType types.CallbackType
		expSuccess      bool
	}{
		{
			"success: transfer with no memo",
			"",
			"none",
			true,
		},
		{
			"success: dest callback",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, callbackAddr),
			types.CallbackTypeWriteAcknowledgement,
			true,
		},
		{
			"success: dest callback with other json fields",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}, "something_else": {}}`, callbackAddr),
			types.CallbackTypeWriteAcknowledgement,
			true,
		},
		{
			"success: dest callback with malformed json",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}, malformed}`, callbackAddr),
			"none",
			true,
		},
		{
			"success: source callback",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, callbackAddr),
			types.CallbackTypeAcknowledgement,
			true,
		},
		{
			"success: source callback with other json fields",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}, "something_else": {}}`, callbackAddr),
			types.CallbackTypeAcknowledgement,
			true,
		},
		{
			"success: source callback with malformed json",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}, malformed}`, callbackAddr),
			"none",
			true,
		},
		{
			"failure: dest callback with low gas (error)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "50000"}}`, callbackAddr),
			types.CallbackTypeWriteAcknowledgement,
			false,
		},
		{
			"failure: source callback with low gas (error)",
			fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "50000"}}`, callbackAddr),
			types.CallbackTypeAcknowledgement,
			false,
		},
		{
			"failure: dest callback with low gas (panic)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "100"}}`, callbackAddr),
			types.CallbackTypeWriteAcknowledgement,
			false,
		},
		{
			"failure: source callback with low gas (panic)",
			fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "100"}}`, callbackAddr),
			types.CallbackTypeAcknowledgement,
			false,
		},
	}

	for _, tc := range testCases {
		suite.SetupFeeTransferTest()

		fee := feetypes.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

		suite.ExecutePayPacketFeeMsg(fee)
		preRelaySenderBalance := sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))
		suite.ExecuteTransfer(tc.transferMemo)
		// we manually subtract the transfer amount from the preRelaySenderBalance because ExecuteTransfer
		// also relays the packet, which will trigger the fee payments.
		preRelaySenderBalance = preRelaySenderBalance.Sub(ibctesting.TestCoin)

		// after incentivizing the packets
		suite.AssertHasExecutedExpectedCallbackWithFee(tc.expCallbackType, tc.expSuccess, false, preRelaySenderBalance, fee)
	}
}

func (suite *CallbacksTestSuite) TestIncentivizedTransferTimeoutCallbacks() {
	testCases := []struct {
		name            string
		transferMemo    string
		expCallbackType types.CallbackType
		expSuccess      bool
	}{
		{
			"success: transfer with no memo",
			"",
			"none",
			true,
		},
		{
			"success: dest callback",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, callbackAddr),
			"none",
			true,
		},
		{
			"success: source callback",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, callbackAddr),
			types.CallbackTypeTimeoutPacket,
			true,
		},
		{
			"success: dest callback with low gas (error)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "50000"}}`, callbackAddr),
			"none",
			true,
		},
		{
			"failure: source callback with low gas (error)",
			fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "50000"}}`, callbackAddr),
			types.CallbackTypeTimeoutPacket,
			false,
		},
		{
			"success: dest callback with low gas (panic)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "100"}}`, callbackAddr),
			"none",
			true,
		},
		{
			"failure: source callback with low gas (panic)",
			fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "100"}}`, callbackAddr),
			types.CallbackTypeTimeoutPacket,
			false,
		},
	}

	for _, tc := range testCases {
		suite.SetupFeeTransferTest()

		fee := feetypes.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

		suite.ExecutePayPacketFeeMsg(fee)
		preRelaySenderBalance := sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))
		suite.ExecuteTransferTimeout(tc.transferMemo, 1)

		// after incentivizing the packets
		suite.AssertHasExecutedExpectedCallbackWithFee(tc.expCallbackType, tc.expSuccess, true, preRelaySenderBalance, fee)
	}
}

func (suite *CallbacksTestSuite) ExecutePayPacketFeeMsg(fee feetypes.Fee) {
	msg := feetypes.NewMsgPayPacketFee(
		fee, suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
		suite.chainA.SenderAccount.GetAddress().String(), nil,
	)

	// fetch the account balance before fees are escrowed and assert the difference below
	preEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NotNil(res)
	suite.Require().NoError(err) // message committed

	postEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	suite.Require().Equal(postEscrowBalance.AddAmount(fee.Total().AmountOf(sdk.DefaultBondDenom)), preEscrowBalance)

	// register counterparty address on chainB
	// relayerAddress is address of sender account on chainB, but we will use it on chainA
	// to differentiate from the chainA.SenderAccount for checking successful relay payouts
	relayerAddress := suite.chainB.SenderAccount.GetAddress()

	msgRegister := feetypes.NewMsgRegisterCounterpartyPayee(
		suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
		suite.chainB.SenderAccount.GetAddress().String(), relayerAddress.String(),
	)
	_, err = suite.chainB.SendMsgs(msgRegister)
	suite.Require().NoError(err) // message committed
}
