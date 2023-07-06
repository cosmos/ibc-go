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
		expCallbackType string
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
			fmt.Sprintf(`{"callback": {"dest_callback_address": "%s"}}`, callbackAddr),
			types.CallbackTypeReceivePacket,
			true,
		},
		{
			"success: source callback",
			fmt.Sprintf(`{"callback": {"src_callback_address": "%s"}}`, callbackAddr),
			types.CallbackTypeAcknowledgement,
			true,
		},
		{
			"failure: dest callback with low gas",
			fmt.Sprintf(`{"callback": {"dest_callback_address": "%s", "gas_limit": 100}}`, callbackAddr),
			types.CallbackTypeReceivePacket,
			false,
		},
		{
			"failure: source callback with low gas",
			fmt.Sprintf(`{"callback": {"src_callback_address": "%s", "gas_limit": 100}}`, callbackAddr),
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

// ExecuteTransfer executes a transfer message on chainA for 100 denom.
// It checks that the transfer is successful and that the packet is relayed to chainB.
func (suite *CallbacksTestSuite) ExecuteTransfer(memo string) {
	escrowAddress := transfertypes.GetEscrowAddress(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	// record the balance of the escrow address before the transfer
	escrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	// record the balance of the receiving address before the transfer
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID, sdk.DefaultBondDenom))
	receiverBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())

	amount := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))
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
