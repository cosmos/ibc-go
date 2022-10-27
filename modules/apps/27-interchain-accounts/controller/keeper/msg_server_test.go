package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

func (suite *KeeperTestSuite) TestRegisterInterchainAccount_MsgServer() {
	var (
		msg               *types.MsgRegisterInterchainAccount
		expectedChannelID = "channel-0"
	)

	testCases := []struct {
		name     string
		expPass  bool
		malleate func()
	}{
		{
			"success",
			true,
			func() {},
		},
		{
			"invalid connection id",
			false,
			func() {
				msg.ConnectionId = "connection-100"
			},
		},
		{
			"non-empty owner address is valid",
			true,
			func() {
				msg.Owner = "<invalid-owner>"
			},
		},
		{
			"empty address invalid",
			false,
			func() {
				msg.Owner = ""
			},
		},
		{
			"port is already bound for owner but capability is claimed by another module",
			false,
			func() {
				capability := suite.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(suite.chainA.GetContext(), TestPortID)
				err := suite.chainA.GetSimApp().TransferKeeper.ClaimCapability(suite.chainA.GetContext(), capability, host.PortPath(TestPortID))
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()

		path := NewICAPath(suite.chainA, suite.chainB)
		suite.coordinator.SetupConnections(path)

		msg = types.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, ibctesting.TestAccAddress, "")

		tc.malleate()

		ctx := suite.chainA.GetContext()
		msgServer := keeper.NewMsgServerImpl(&suite.chainA.GetSimApp().ICAControllerKeeper)
		res, err := msgServer.RegisterInterchainAccount(ctx, msg)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			suite.Require().Equal(expectedChannelID, res.ChannelId)

			events := ctx.EventManager().Events()
			suite.Require().Len(events, 2)
			suite.Require().Equal(events[0].Type, channeltypes.EventTypeChannelOpenInit)
			suite.Require().Equal(events[1].Type, sdk.EventTypeMessage)
		} else {
			suite.Require().Error(err)
			suite.Require().Nil(res)
		}
	}
}

func (suite *KeeperTestSuite) TestSubmitTx() {
	var (
		path *ibctesting.Path
		msg  *types.MsgSendTx
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
			},
			true,
		},
		{
			"failure - owner address is empty", func() {
				msg.Owner = ""
			},
			false,
		},
		{
			"failure - active channel does not exist for connection ID", func() {
				msg.Owner = TestOwnerAddress
				msg.ConnectionId = "connection-100"
			},
			false,
		},
		{
			"failure - active channel does not exist for port ID", func() {
				msg.Owner = "invalid-owner"
			},
			false,
		},
		{
			"failure - controller module does not own capability for this channel", func() {
				msg.Owner = "invalid-owner"
				portID, err := icatypes.NewControllerPortID(msg.Owner)
				suite.Require().NoError(err)

				// set the active channel with the incorrect portID in order to reach the capability check
				suite.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(suite.chainA.GetContext(), path.EndpointA.ConnectionID, portID, path.EndpointA.ChannelID)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			owner := TestOwnerAddress
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICAPath(path, owner)
			suite.Require().NoError(err)

			portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
			suite.Require().NoError(err)

			// get the address of the interchain account stored in state during handshake step
			interchainAccountAddr, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), path.EndpointA.ConnectionID, portID)
			suite.Require().True(found)

			// create bank transfer message that will execute on the host chain
			icaMsg := &banktypes.MsgSend{
				FromAddress: interchainAccountAddr,
				ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
				Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
			}

			data, err := icatypes.SerializeCosmosTx(suite.chainA.Codec, []proto.Message{icaMsg})
			suite.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: data,
				Memo: "memo",
			}

			timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute).UnixNano())
			connectionID := path.EndpointA.ConnectionID

			msg = types.NewMsgSendTx(owner, connectionID, timeoutTimestamp, packetData)

			tc.malleate() // malleate mutates test data

			ctx := suite.chainA.GetContext()
			msgServer := keeper.NewMsgServerImpl(&suite.chainA.GetSimApp().ICAControllerKeeper)
			res, err := msgServer.SendTx(ctx, msg)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
			}
		})
	}
}
