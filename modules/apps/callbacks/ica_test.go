package ibccallbacks_test

import (
	"fmt"
	"time"

	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (suite *CallbacksTestSuite) TestICACallbacks() {
	// Destination callbacks are not supported for ICA packets
	testCases := []struct {
		name            string
		icaMemo         string
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
			"success: dest callback with other json fields",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}, "something_else": {}}`, callbackAddr),
			"none",
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
			"none",
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
			"none",
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
		icaAddr := suite.SetupICATest()

		suite.ExecuteICATx(icaAddr, tc.icaMemo, 1)
		suite.AssertHasExecutedExpectedCallback(tc.expCallbackType, tc.expSuccess)
	}
}

func (suite *CallbacksTestSuite) TestICATimeoutCallbacks() {
	// ICA channels are closed after a timeout packet is executed
	testCases := []struct {
		name            string
		icaMemo         string
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
		icaAddr := suite.SetupICATest()

		suite.ExecuteICATimeout(icaAddr, tc.icaMemo, 1)
		suite.AssertHasExecutedExpectedCallback(tc.expCallbackType, tc.expSuccess)
	}
}

// ExecuteICATx executes a stakingtypes.MsgDelegate on chainB by sending a packet containing the msg to chainB
func (suite *CallbacksTestSuite) ExecuteICATx(icaAddress, memo string, seq uint64) {
	timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute).UnixNano())
	icaOwner := suite.chainA.SenderAccount.GetAddress().String()
	connectionID := suite.path.EndpointA.ConnectionID
	// build the interchain accounts packet data
	packetData := suite.buildICAMsgDelegatePacketData(icaAddress, memo)
	msg := icacontrollertypes.NewMsgSendTx(icaOwner, connectionID, timeoutTimestamp, packetData)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed
	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents().ToABCIEvents())
	suite.Require().NoError(err)

	err = suite.path.RelayPacket(packet)
	suite.Require().NoError(err)
}

// ExecuteICATx executes a stakingtypes.MsgDelegate on chainB by sending a packet containing the msg to chainB
func (suite *CallbacksTestSuite) ExecuteICATimeout(icaAddress, memo string, seq uint64) {
	timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())
	icaOwner := suite.chainA.SenderAccount.GetAddress().String()
	connectionID := suite.path.EndpointA.ConnectionID
	// build the interchain accounts packet data
	packetData := suite.buildICAMsgDelegatePacketData(icaAddress, memo)
	msg := icacontrollertypes.NewMsgSendTx(icaOwner, connectionID, timeoutTimestamp, packetData)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed
	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents().ToABCIEvents())
	suite.Require().NoError(err)

	module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID)
	suite.Require().NoError(err)

	cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	suite.Require().True(ok)

	err = cbs.OnTimeoutPacket(suite.chainA.GetContext(), packet, nil)
	suite.Require().NoError(err)
}

// buildICAMsgDelegatePacketData builds a packetData containing a stakingtypes.MsgDelegate to be executed on chainB
func (suite *CallbacksTestSuite) buildICAMsgDelegatePacketData(icaAddress string, memo string) icatypes.InterchainAccountPacketData {
	// prepare a simple stakingtypes.MsgDelegate to be used as the interchain account msg executed on chainB
	validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
	msgDelegate := &stakingtypes.MsgDelegate{
		DelegatorAddress: icaAddress,
		ValidatorAddress: validatorAddr.String(),
		Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
	}

	// ensure chainB is allowed to execute stakingtypes.MsgDelegate
	params := icahosttypes.NewParams(true, []string{sdk.MsgTypeURL(msgDelegate)})
	suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)

	data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msgDelegate}, icatypes.EncodingProtobuf)
	suite.Require().NoError(err)

	icaPacketData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
		Memo: memo,
	}

	return icaPacketData
}
