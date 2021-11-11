package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v2/testing"
)

func (suite *KeeperTestSuite) TestTrySendTx() {
	var (
		path          *ibctesting.Path
		icaPacketData types.InterchainAccountPacketData
		portID        string
		chanCap       *capabilitytypes.Capability
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
			}, true,
		},
		{
			"success with multiple sdk.Msg", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg1 := &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				msg2 := &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				data, err := types.SerializeCosmosTx(suite.chainB.GetSimApp().AppCodec(), []sdk.Msg{msg1, msg2})
				suite.Require().NoError(err)
				icaPacketData.Data = data
			}, true,
		},
		{
			"data is nil", func() {
				icaPacketData.Data = nil
			}, false,
		},
		{
			"active channel not found", func() {
				portID = "incorrect portID"
			}, false,
		},
		{
			"channel does not exist", func() {
				suite.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(suite.chainA.GetContext(), portID, "channel-100")
			}, false,
		},
		{
			"SendPacket fails - channel closed", func() {
				err := path.EndpointA.SetChannelClosed()
				suite.Require().NoError(err)
			}, false,
		},
		{
			"invalid channel capability provided", func() {
				chanCap = nil
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			// default setup
			portID = path.EndpointA.ChannelConfig.PortID

			amount, _ := sdk.ParseCoinsNormalized("100stake")
			interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)

			msg := &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
			data, err := types.SerializeCosmosTx(suite.chainB.GetSimApp().AppCodec(), []sdk.Msg{msg})
			suite.Require().NoError(err)

			// default packet data, must be modified in malleate for test cases expected to fail
			icaPacketData = types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: data,
				Memo: "memo",
			}

			var ok bool
			chanCap, ok = suite.chainA.GetSimApp().ScopedICAMockKeeper.GetCapability(path.EndpointA.Chain.GetContext(), host.ChannelCapabilityPath(portID, path.EndpointA.ChannelID))
			suite.Require().True(ok)

			tc.malleate()

			_, err = suite.chainA.GetSimApp().ICAControllerKeeper.TrySendTx(suite.chainA.GetContext(), chanCap, portID, icaPacketData)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnTimeoutPacket() {
	var (
		path *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			packet := channeltypes.NewPacket([]byte{}, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)
			err = suite.chainA.GetSimApp().ICAControllerKeeper.OnTimeoutPacket(suite.chainA.GetContext(), packet)

			activeChannelID, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetActiveChannelID(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Empty(activeChannelID)
				suite.Require().False(found)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
