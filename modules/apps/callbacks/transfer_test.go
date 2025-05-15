package ibccallbacks_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/testing/simapp"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *CallbacksTestSuite) TestTransferCallbacks() {
	testCases := []struct {
		name         string
		transferMemo string
		expCallback  types.CallbackType
		expSuccess   bool
	}{
		{
			"success: transfer with no memo",
			"",
			"none",
			true,
		},
		{
			"success: dest callback",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.SuccessContract),
			types.CallbackTypeReceivePacket,
			true,
		},
		{
			"success: dest callback with other json fields",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}, "something_else": {}}`, simapp.SuccessContract),
			types.CallbackTypeReceivePacket,
			true,
		},
		{
			"success: dest callback with malformed json",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}, malformed}`, simapp.SuccessContract),
			"none",
			true,
		},
		{
			"failure: dest callback with missing address",
			`{"dest_callback": {"address": ""}}`,
			"none",
			false,
		},
		{
			"success: source callback",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
			types.CallbackTypeAcknowledgementPacket,
			true,
		},
		{
			"success: source callback with other json fields",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}, "something_else": {}}`, simapp.SuccessContract),
			types.CallbackTypeAcknowledgementPacket,
			true,
		},
		{
			"success: source callback with malformed json",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}, malformed}`, simapp.SuccessContract),
			"none",
			true,
		},
		{
			"success: source callback with missing address",
			`{"src_callback": {"address": ""}}`,
			"none",
			true,
		},
		{
			"failure: dest callback with low gas (panic)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogPanicContract),
			types.CallbackTypeReceivePacket,
			false,
		},
		{
			"failure: source callback with low gas (panic)",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.OogPanicContract),
			types.CallbackTypeSendPacket,
			false,
		},
		{
			"failure: dest callback with low gas (error)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogErrorContract),
			types.CallbackTypeReceivePacket,
			false,
		},
		{
			"failure: source callback with low gas (error)",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.OogErrorContract),
			types.CallbackTypeSendPacket,
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			s.ExecuteTransfer(tc.transferMemo, tc.expSuccess)
			s.AssertHasExecutedExpectedCallback(tc.expCallback, tc.expSuccess)
		})
	}
}

func (s *CallbacksTestSuite) TestTransferTimeoutCallbacks() {
	testCases := []struct {
		name         string
		transferMemo string
		expCallback  types.CallbackType
		expSuccess   bool
	}{
		{
			"success: transfer with no memo",
			"",
			"none",
			true,
		},
		{
			"success: dest callback",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.SuccessContract),
			"none", // timeouts don't reach destination chain execution
			true,
		},
		{
			"success: source callback",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
			types.CallbackTypeTimeoutPacket,
			true,
		},
		{
			"success: dest callback with low gas (panic)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogPanicContract),
			"none", // timeouts don't reach destination chain execution
			true,
		},
		{
			"success: dest callback with low gas (error)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogErrorContract),
			"none", // timeouts don't reach destination chain execution
			true,
		},
		{
			"failure: source callback with low gas (panic)",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.OogPanicContract),
			types.CallbackTypeSendPacket,
			false,
		},
		{
			"failure: source callback with low gas (error)",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.OogErrorContract),
			types.CallbackTypeSendPacket,
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			s.ExecuteTransferTimeout(tc.transferMemo)
			s.AssertHasExecutedExpectedCallback(tc.expCallback, tc.expSuccess)
		})
	}
}

// ExecuteTransfer executes a transfer message on chainA for ibctesting.TestCoin (100 "stake").
// It checks that the transfer is successful and that the packet is relayed to chainB.
func (s *CallbacksTestSuite) ExecuteTransfer(memo string, recvSuccess bool) {
	escrowAddress := transfertypes.GetEscrowAddress(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	// record the balance of the escrow address before the transfer
	escrowBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	// record the balance of the receiving address before the transfer
	denom := transfertypes.NewDenom(sdk.DefaultBondDenom, transfertypes.NewHop(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID))
	receiverBalance := GetSimApp(s.chainB).BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), denom.IBCDenom())
	// record the balance of the sending address before the transfer
	senderBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	amount := ibctesting.TestCoin
	msg := transfertypes.NewMsgTransfer(
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		amount,
		s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(),
		clienttypes.NewHeight(1, 100), 0, memo,
	)

	res, err := s.chainA.SendMsgs(msg)
	if err != nil {
		return // we return if send packet is rejected
	}

	// packet found, relay from A to B
	err = s.path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	packet, err := ibctesting.ParseV1PacketFromEvents(res.GetEvents())
	s.Require().NoError(err)

	res, err = s.path.EndpointB.RecvPacketWithResult(packet)
	s.Require().NoError(err)

	acknowledgement, err := ibctesting.ParseAckFromEvents(res.Events)
	s.Require().NoError(err)

	err = s.path.EndpointA.AcknowledgePacket(packet, acknowledgement)
	s.Require().NoError(err)

	var ack channeltypes.Acknowledgement
	err = transfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack)
	s.Require().NoError(err)

	s.Require().Equal(recvSuccess, ack.Success(), "acknowledgement success is not as expected")

	if recvSuccess {
		// check that the escrow address balance increased by 100
		s.Require().Equal(escrowBalance.Add(amount), GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom))
		// check that the receiving address balance increased by 100
		s.Require().Equal(receiverBalance.AddAmount(sdkmath.NewInt(100)), GetSimApp(s.chainB).BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), denom.IBCDenom()))
		// check that the sending address balance decreased by 100
		s.Require().Equal(senderBalance.Sub(amount), GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom))
	} else {
		// check that the escrow address balance is the same as before the transfer
		s.Require().Equal(escrowBalance, GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom))
		// check that the receiving address balance is the same as before the transfer
		s.Require().Equal(receiverBalance, GetSimApp(s.chainB).BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), denom.IBCDenom()))
		// check that the sending address balance is the same as before the transfer
		s.Require().Equal(senderBalance, GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom))
	}
}

// ExecuteTransferTimeout executes a transfer message on chainA for 100 denom.
// This message is not relayed to chainB, and it times out on chainA.
func (s *CallbacksTestSuite) ExecuteTransferTimeout(memo string) {
	timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
	timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

	amount := ibctesting.TestCoin
	msg := transfertypes.NewMsgTransfer(
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		amount,
		s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(),
		timeoutHeight, timeoutTimestamp, memo,
	)

	res, err := s.chainA.SendMsgs(msg)
	if err != nil {
		return // we return if send packet is rejected
	}

	packet, err := ibctesting.ParseV1PacketFromEvents(res.GetEvents())
	s.Require().NoError(err) // packet committed
	s.Require().NotNil(packet)

	// need to update chainA's client representing chainB to prove missing ack
	err = s.path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	err = s.path.EndpointA.TimeoutPacket(packet)
	s.Require().NoError(err) // timeout committed
}
