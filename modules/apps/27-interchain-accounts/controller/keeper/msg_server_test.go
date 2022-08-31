package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

func (suite *KeeperTestSuite) TestRegisterAccount() {
	var (
		msg               *icacontrollertypes.MsgRegisterAccount
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

		msg = icacontrollertypes.NewMsgRegisterAccount(ibctesting.FirstConnectionID, ibctesting.TestAccAddress, "")

		tc.malleate()

		ctx := suite.chainA.GetContext()
		res, err := suite.chainA.GetSimApp().ICAControllerKeeper.RegisterAccount(ctx, msg)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			suite.Require().Equal(expectedChannelID, res.ChannelId)

			events := ctx.EventManager().Events()
			suite.Require().Len(events, 2)
			suite.Require().Equal(events[0].Type, channeltypes.EventTypeChannelOpenInit)
			suite.Require().Equal(events[1].Type, sdktypes.EventTypeMessage)
		} else {
			suite.Require().Error(err)
			suite.Require().Nil(res)
		}
	}
}

func (suite *KeeperTestSuite) TestSubmitTx() {
	var (
		path         *ibctesting.Path
		owner        string
		connectionId string
		icaMsg       sdk.Msg
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
				owner = TestOwnerAddress
				connectionId = path.EndpointA.ConnectionID
			},
			true,
		},
		{
			"failure - owner address is empty", func() {
				owner = ""
				connectionId = path.EndpointA.ConnectionID
			},
			false,
		},
		{
			"failure - active channel does not exist for connection ID", func() {
				owner = TestOwnerAddress
				connectionId = "connection-100"
			},
			false,
		},
		{
			"failure - active channel does not exist for port ID", func() {
				owner = "cosmos153lf4zntqt33a4v0sm5cytrxyqn78q7kz8j8x5"
				connectionId = path.EndpointA.ConnectionID
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			tc.malleate() // malleate mutates test data

			err := SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
			suite.Require().NoError(err)

			// Get the address of the interchain account stored in state during handshake step
			interchainAccountAddr, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), path.EndpointA.ConnectionID, portID)
			suite.Require().True(found)

			icaAddr, err := sdk.AccAddressFromBech32(interchainAccountAddr)
			suite.Require().NoError(err)

			// Check if account is created
			interchainAccount := suite.chainB.GetSimApp().AccountKeeper.GetAccount(suite.chainB.GetContext(), icaAddr)
			suite.Require().Equal(interchainAccount.GetAddress().String(), interchainAccountAddr)

			// Create bank transfer message to execute on the host
			icaMsg = &banktypes.MsgSend{
				FromAddress: interchainAccountAddr,
				ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
				Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
			}

			data, err := icatypes.SerializeCosmosTx(suite.chainA.Codec, []sdk.Msg{icaMsg})
			suite.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: data,
				Memo: "memo",
			}

			// timeoutTimestamp set to max value with the unsigned bit shifted to sastisfy hermes timestamp conversion
			// it is the responsibility of the auth module developer to ensure an appropriate timeout timestamp
			timeoutTimestamp := suite.chainA.GetContext().BlockTime().Add(time.Minute).UnixNano()

			msg := types.NewMsgSubmitTx(owner, connectionId, clienttypes.NewHeight(0, 0), uint64(timeoutTimestamp), packetData)
			res, err := suite.chainA.GetSimApp().ICAControllerKeeper.SubmitTx(sdk.WrapSDKContext(suite.chainA.GetContext()), msg)

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
