package ibccallbacks_test

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func (suite *CallbacksTestSuite) TestICACallbacks() {
	// Destination callbacks are not supported for ICA packets
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

		suite.ExecuteICATx(icaAddr, tc.transferMemo, 1)
	}
}

// ExecuteICATx executes a stakingtypes.MsgDelegate on chainB by sending a packet containing the msg to chainB
func (suite *CallbacksTestSuite) ExecuteICATx(icaAddress, memo string, seq uint64) {
	// build the interchain accounts packet
	packet := suite.buildICAMsgDelegatePacket(icaAddress, seq)

	// write packet commitment to state on chainA and commit state
	commitment := channeltypes.CommitPacket(suite.chainA.GetSimApp().AppCodec(), packet)
	suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, seq, commitment)
	suite.chainA.NextBlock()

	err := suite.path.RelayPacket(packet)
	suite.Require().NoError(err)
}

// buildICAMsgDelegatePacket builds a packet containing a stakingtypes.MsgDelegate to be executed on chainB
func (suite *CallbacksTestSuite) buildICAMsgDelegatePacket(icaAddress string, seq uint64) channeltypes.Packet {
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
	}

	packet := channeltypes.NewPacket(
		icaPacketData.GetBytes(),
		seq,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	return packet
}
