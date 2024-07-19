package ibccallbacks_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/testing/simapp"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

var (
	defaultRecvFee    = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}}
	defaultAckFee     = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(200)}}
	defaultTimeoutFee = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(300)}}
)

func (s *CallbacksTestSuite) TestIncentivizedTransferCallbacks() {
	var transferMemo string

	testCases := []struct {
		name        string
		malleate    func()
		expCallback types.CallbackType
		expSuccess  bool
	}{
		{
			"success: transfer with no memo",
			func() {},
			"none",
			true,
		},
		{
			"success: dest callback",
			func() {
				transferMemo = fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.SuccessContract)
			},
			types.CallbackTypeReceivePacket,
			true,
		},
		{
			"success: dest callback with other json fields",
			func() {
				transferMemo = fmt.Sprintf(`{"dest_callback": {"address": "%s"}, "something_else": {}}`, simapp.SuccessContract)
			},
			types.CallbackTypeReceivePacket,
			true,
		},
		{
			"success: dest callback with malformed json",
			func() {
				transferMemo = fmt.Sprintf(`{"dest_callback": {"address": "%s"}, malformed}`, simapp.SuccessContract)
			},
			"none",
			true,
		},
		{
			"success: dest callback with missing address",
			func() {
				//nolint:goconst
				transferMemo = `{"dest_callback": {"address": ""}}`
			},
			"none",
			true,
		},
		{
			"success: source callback",
			func() {
				transferMemo = fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract)
			},
			types.CallbackTypeAcknowledgementPacket,
			true,
		},
		{
			"success: source callback with other json fields",
			func() {
				transferMemo = fmt.Sprintf(`{"src_callback": {"address": "%s"}, "something_else": {}}`, simapp.SuccessContract)
			},
			types.CallbackTypeAcknowledgementPacket,
			true,
		},
		{
			"success: source callback with malformed json",
			func() {
				transferMemo = fmt.Sprintf(`{"src_callback": {"address": "%s"}, malformed}`, simapp.SuccessContract)
			},
			"none",
			true,
		},
		{
			"success: source callback with missing address",
			func() {
				//nolint:goconst
				transferMemo = `{"src_callback": {"address": ""}}`
			},
			"none",
			true,
		},
		{
			"success: acknowledgement",
			func() {
				transferMemo = fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract)

				k := GetSimApp(s.chainA).MockContractKeeper

				k.IBCOnAcknowledgementPacketCallbackFn = func(
					cachedCtx sdk.Context,
					_ channeltypes.Packet,
					acknowledgement []byte,
					_ sdk.AccAddress,
					contractAddress,
					_ string,
				) error {
					expAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()
					s.Require().Equal(expAck, acknowledgement)

					return k.ProcessMockCallback(cachedCtx, types.CallbackTypeAcknowledgementPacket, contractAddress)
				}
			},
			types.CallbackTypeAcknowledgementPacket,
			true,
		},
		{
			"failure: dest callback with low gas (panic)",
			func() {
				transferMemo = fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogPanicContract)
			},
			types.CallbackTypeReceivePacket,
			false,
		},
		{
			"failure: dest callback with low gas (error)",
			func() {
				transferMemo = fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogErrorContract)
			},
			types.CallbackTypeReceivePacket,
			false,
		},
		{
			"failure: source callback with low gas (panic)",
			func() {
				transferMemo = fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.OogPanicContract)
			},
			types.CallbackTypeSendPacket,
			false,
		},
		{
			"failure: source callback with low gas (error)",
			func() {
				transferMemo = fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.OogErrorContract)
			},
			types.CallbackTypeSendPacket,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupFeeTransferTest()
			transferMemo = ""

			tc.malleate()

			fee := feetypes.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

			s.ExecutePayPacketFeeMsg(fee)
			preRelaySenderBalance := sdk.NewCoins(GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))
			s.ExecuteTransfer(transferMemo)
			// we manually subtract the transfer amount from the preRelaySenderBalance because ExecuteTransfer
			// also relays the packet, which will trigger the fee payments.
			preRelaySenderBalance = preRelaySenderBalance.Sub(ibctesting.TestCoin)

			// after incentivizing the packets
			s.AssertHasExecutedExpectedCallbackWithFee(tc.expCallback, tc.expSuccess, false, preRelaySenderBalance, fee)
		})
	}
}

func (s *CallbacksTestSuite) TestIncentivizedTransferTimeoutCallbacks() {
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
			"none",
			true, // timeouts don't reach destination chain execution
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
			false,
		},
		{
			"failure: source callback with low gas (panic)",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.OogPanicContract),
			types.CallbackTypeSendPacket,
			false,
		},
		{
			"success: dest callback with low gas (error)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogErrorContract),
			"none", // timeouts don't reach destination chain execution
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
		tc := tc

		s.Run(tc.name, func() {
			s.SetupFeeTransferTest()

			fee := feetypes.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

			s.ExecutePayPacketFeeMsg(fee)
			preRelaySenderBalance := sdk.NewCoins(GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))
			s.ExecuteTransferTimeout(tc.transferMemo)

			// after incentivizing the packets
			s.AssertHasExecutedExpectedCallbackWithFee(tc.expCallback, tc.expSuccess, true, preRelaySenderBalance, fee)
		})
	}
}

func (s *CallbacksTestSuite) ExecutePayPacketFeeMsg(fee feetypes.Fee) {
	msg := feetypes.NewMsgPayPacketFee(
		fee, s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
		s.chainA.SenderAccount.GetAddress().String(), nil,
	)

	// fetch the account balance before fees are escrowed and assert the difference below
	preEscrowBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	postEscrowBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	s.Require().Equal(postEscrowBalance.AddAmount(fee.Total().AmountOf(sdk.DefaultBondDenom)), preEscrowBalance)

	// register counterparty address on chainB
	payeeAddr, err := sdk.AccAddressFromBech32(ibctesting.TestAccAddress)
	s.Require().NoError(err)

	msgRegister := feetypes.NewMsgRegisterCounterpartyPayee(
		s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID,
		s.chainB.SenderAccount.GetAddress().String(), payeeAddr.String(),
	)
	_, err = s.chainB.SendMsgs(msgRegister)
	s.Require().NoError(err) // message committed
}
