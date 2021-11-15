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

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	var (
		path       *ibctesting.Path
		msg        sdk.Msg
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
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				// build packet data
				data, err := types.SerializeCosmosTx(suite.chainA.Codec, []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}
				packetData = icaPacketData.GetBytes()
			}, true,
		},
		{
			"Cannot deserialize packet data messages", func() {
				data := []byte("invalid packet data")

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}
				packetData = icaPacketData.GetBytes()
			}, false,
		},
		{
			"Invalid packet type", func() {
				// build packet data
				data, err := types.SerializeCosmosTx(suite.chainA.Codec, []sdk.Msg{&banktypes.MsgSend{}})
				suite.Require().NoError(err)

				// Type here is an ENUM
				// Valid type is types.EXECUTE_TX
				icaPacketData := types.InterchainAccountPacketData{
					Type: 100,
					Data: data,
				}
				packetData = icaPacketData.GetBytes()
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
				data, err := types.SerializeCosmosTx(suite.chainA.Codec, []sdk.Msg{msg})
				suite.Require().NoError(err)
				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}
				packetData = icaPacketData.GetBytes()
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			// send 100stake to interchain account wallet
			amount, _ := sdk.ParseCoinsNormalized("100stake")
			interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
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
			err = suite.chainB.GetSimApp().ICAHostKeeper.OnRecvPacket(suite.chainB.GetContext(), packet)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
