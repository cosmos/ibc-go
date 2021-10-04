package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"

	ibctesting "github.com/cosmos/ibc-go/v2/testing"
)

func (suite *KeeperTestSuite) TestTrySendTx() {
	var (
		path   *ibctesting.Path
		msg    interface{}
		portID string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
			}, true,
		},
		{
			"success with []sdk.Message", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg1 := &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				msg2 := &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				msg = []sdk.Msg{msg1, msg2}
			}, true,
		},
		{
			"incorrect outgoing data", func() {
				msg = []byte{}
			}, false,
		},
		{
			"active channel not found", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				portID = "incorrect portID"
			}, false,
		},
		{
			"channel does not exist", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				suite.chainA.GetSimApp().ICAKeeper.SetActiveChannel(suite.chainA.GetContext(), portID, "channel-100")
			}, false,
		},
		{
			"data is nil", func() {
				msg = nil
			}, false,
		},
		{
			"data is not an SDK message", func() {
				msg = "not an sdk message"
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			memo := "memo"

			err := suite.SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			portID = path.EndpointA.ChannelConfig.PortID

			tc.malleate()

			_, err = suite.chainA.GetSimApp().ICAKeeper.TrySendTx(suite.chainA.GetContext(), portID, msg, memo)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	var (
		path       *ibctesting.Path
		msg        sdk.Msg
		txBytes    []byte
		packetData []byte
		sourcePort string
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"Interchain account successfully executes banktypes.MsgSend", func() {
				// build MsgSend
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				// build packet data
				txBytes, err := suite.chainA.GetSimApp().ICAKeeper.SerializeCosmosTx(suite.chainA.Codec, msg)
				suite.Require().NoError(err)

				data := types.InterchainAccountPacketData{Type: types.EXECUTE_TX,
					Data: txBytes}
				packetData = data.GetBytes()
			}, true,
		},
		{
			"Cannot deserialize txBytes", func() {
				txBytes = []byte("invalid tx bytes")
				data := types.InterchainAccountPacketData{Type: types.EXECUTE_TX,
					Data: txBytes}
				packetData = data.GetBytes()
			}, false,
		},
		{
			"Invalid packet type", func() {
				txBytes = []byte{}
				// Type here is an ENUM
				// Valid type is types.EXECUTE_TX
				data := types.InterchainAccountPacketData{Type: 100,
					Data: txBytes}
				packetData = data.GetBytes()
			}, false,
		},
		{
			"Cannot unmarshal interchain account packet data into types.InterchainAccountPacketData", func() {
				packetData = []byte{}
			}, false,
		},
		{
			"Unauthorised: Interchain account not found for given source portID", func() {
				sourcePort = "invalid-port-id"
			}, false,
		},
		{
			"Unauthorised: Signer of message is not the interchain account associated with sourcePortID", func() {
				// build MsgSend
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				// Incorrect FromAddress
				msg = &banktypes.MsgSend{FromAddress: suite.chainB.SenderAccount.GetAddress().String(), ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				// build packet data
				txBytes, err := suite.chainA.GetSimApp().ICAKeeper.SerializeCosmosTx(suite.chainA.Codec, msg)
				suite.Require().NoError(err)
				data := types.InterchainAccountPacketData{Type: types.EXECUTE_TX,
					Data: txBytes}
				packetData = data.GetBytes()
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			err := suite.SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			// send 100stake to interchain account wallet
			amount, _ := sdk.ParseCoinsNormalized("100stake")
			interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
			bankMsg := &banktypes.MsgSend{FromAddress: suite.chainB.SenderAccount.GetAddress().String(), ToAddress: interchainAccountAddr, Amount: amount}

			_, err = suite.chainB.SendMsgs(bankMsg)
			suite.Require().NoError(err)

			// valid source port
			sourcePort = path.EndpointA.ChannelConfig.PortID

			// malleate packetData for test cases
			tc.malleate()

			seq := uint64(1)
			packet := channeltypes.NewPacket(packetData, seq, sourcePort, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)

			// Pass it in here
			err = suite.chainB.GetSimApp().ICAKeeper.OnRecvPacket(suite.chainB.GetContext(), packet)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
