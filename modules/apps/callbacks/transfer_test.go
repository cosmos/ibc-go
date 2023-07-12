package ibccallbacks_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

const (
	callbackAddr = "cosmos1q4hx350dh0843y34n0vm4lfj6eh5qz4sqfrnq0"
)

func (suite *CallbacksTestSuite) TestTransferCallbacks() {
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
			types.CallbackTypeReceivePacket,
			true,
		},
		{
			"success: dest callback with other json fields",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}, "something_else": {}}`, callbackAddr),
			types.CallbackTypeReceivePacket,
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
			types.CallbackTypeReceivePacket,
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
			types.CallbackTypeReceivePacket,
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
		suite.SetupTransferTest()

		suite.ExecuteTransfer(tc.transferMemo)
		suite.AssertHasExecutedExpectedCallback(tc.expCallbackType, tc.expSuccess)
	}
}

func (suite *CallbacksTestSuite) TestTransferTimeoutCallbacks() {
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
		suite.SetupTransferTest()

		suite.ExecuteTransferTimeout(tc.transferMemo, 1)
		suite.AssertHasExecutedExpectedCallback(tc.expCallbackType, tc.expSuccess)
	}
}

// ExecuteTransfer executes a transfer message on chainA for ibctesting.TestCoin (100 "stake").
// It checks that the transfer is successful and that the packet is relayed to chainB.
func (suite *CallbacksTestSuite) ExecuteTransfer(memo string) {
	escrowAddress := transfertypes.GetEscrowAddress(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	// record the balance of the escrow address before the transfer
	escrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	// record the balance of the receiving address before the transfer
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID, sdk.DefaultBondDenom))
	receiverBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())

	amount := ibctesting.TestCoin
	msg := transfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		clienttypes.NewHeight(1, 100), 0, memo,
	)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents().ToABCIEvents())
	suite.Require().NoError(err)

	// relay send
	err = suite.path.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	// check that the escrow address balance increased by 100
	suite.Require().Equal(escrowBalance.Add(amount), suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom))
	// check that the receiving address balance increased by 100
	suite.Require().Equal(receiverBalance.AddAmount(sdk.NewInt(100)), suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom()))
}

// ExecuteTransferTimeout executes a transfer message on chainA for 100 denom.
// This message is not relayed to chainB, and it times out on chainA.
func (suite *CallbacksTestSuite) ExecuteTransferTimeout(memo string, nextSeqRecv uint64) {
	timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
	timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

	amount := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))
	msg := transfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		timeoutHeight, timeoutTimestamp, memo,
	)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents().ToABCIEvents())
	suite.Require().NoError(err) // packet committed
	suite.Require().NotNil(packet)

	// need to update chainA's client representing chainB to prove missing ack
	err = suite.path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = suite.path.EndpointA.TimeoutPacket(packet)
	suite.Require().NoError(err) // timeout committed
}
