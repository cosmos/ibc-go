package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"

	ibctesting "github.com/cosmos/ibc-go/testing"
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
			owner := TestOwnerAddress
			suite.coordinator.SetupConnections(path)

			err := suite.SetupICAPath(path, owner)
			suite.Require().NoError(err)

			portID = path.EndpointA.ChannelConfig.PortID
			tc.malleate()

			_, err = suite.chainA.GetSimApp().ICAKeeper.TrySendTx(suite.chainA.GetContext(), portID, msg)

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
				data := types.IBCAccountPacketData{Type: types.EXECUTE_TX,
					Data: txBytes}
				packetData = data.GetBytes()
			}, true,
		},
		{
			"Cannot deserialize txBytes", func() {
				txBytes = []byte("invalid tx bytes")
				data := types.IBCAccountPacketData{Type: types.EXECUTE_TX,
					Data: txBytes}
				packetData = data.GetBytes()
			}, false,
		},
		{
			"Cannot deserialize txBytes: invalid IBCTxRaw", func() {
				txBody := []byte("invalid tx body")
				txRaw := &types.IBCTxRaw{
					BodyBytes: txBody,
				}

				txBytes = suite.chainB.Codec.MustMarshal(txRaw)
				data := types.IBCAccountPacketData{Type: types.EXECUTE_TX,
					Data: txBytes}
				packetData = data.GetBytes()
			}, false,
		},
		{
			"Wrong data type", func() {
				txBytes = []byte{}
				data := types.IBCAccountPacketData{Type: 100,
					Data: txBytes}
				packetData = data.GetBytes()
			}, false,
		},
		{
			"Cannot unmarshal interchain account packet data", func() {
				packetData = []byte{}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			// setup interchain account
			owner := suite.chainA.SenderAccount.GetAddress().String()
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			err := suite.SetupICAPath(path, owner)
			suite.Require().NoError(err)

			// send 100stake to interchain account wallet
			amount, _ := sdk.ParseCoinsNormalized("100stake")
			interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
			bankMsg := &banktypes.MsgSend{FromAddress: suite.chainB.SenderAccount.GetAddress().String(), ToAddress: interchainAccountAddr, Amount: amount}
			_, err = suite.chainB.SendMsgs(bankMsg)

			txBytes, err = suite.chainA.GetSimApp().ICAKeeper.SerializeCosmosTx(suite.chainA.Codec, msg)
			// Next we need to define the packet/data to pass into OnRecvPacket
			seq := uint64(1)

			tc.malleate()

			packet := channeltypes.NewPacket(packetData, seq, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)

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
