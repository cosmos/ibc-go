package keeper_test

import (
	"time"

	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestRegisterInterchainAccount_MsgServer() {
	var (
		msg               *types.MsgRegisterInterchainAccount
		expectedOrderding channeltypes.Order
		expectedChannelID = "channel-0"
	)

	testCases := []struct {
		name     string
		expErr   error
		malleate func()
	}{
		{
			"success",
			nil,
			func() {},
		},
		{
			"success: ordering falls back to UNORDERED if not specified",
			nil,
			func() {
				msg.Ordering = channeltypes.NONE
				expectedOrderding = channeltypes.UNORDERED
			},
		},
		{
			"invalid connection id",
			connectiontypes.ErrConnectionNotFound,
			func() {
				msg.ConnectionId = "connection-100"
			},
		},
		{
			"non-empty owner address is valid",
			nil,
			func() {
				msg.Owner = "<invalid-owner>"
			},
		},
		{
			"empty address invalid",
			icatypes.ErrInvalidAccountAddress,
			func() {
				msg.Owner = ""
			},
		},
		{
			"port is already bound for owner but capability is claimed by another module",
			icatypes.ErrPortAlreadyBound,
			func() {
				capability := suite.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(suite.chainA.GetContext(), TestPortID)
				err := suite.chainA.GetSimApp().TransferKeeper.ClaimCapability(suite.chainA.GetContext(), capability, host.PortPath(TestPortID))
				suite.Require().NoError(err)
			},
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				expectedOrderding = ordering

				suite.SetupTest()

				path := NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				msg = types.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, ibctesting.TestAccAddress, "", ordering)

				tc.malleate()

				ctx := suite.chainA.GetContext()
				msgServer := keeper.NewMsgServerImpl(&suite.chainA.GetSimApp().ICAControllerKeeper)
				res, err := msgServer.RegisterInterchainAccount(ctx, msg)

				if tc.expErr == nil {
					suite.Require().NoError(err)
					suite.Require().NotNil(res)
					suite.Require().Equal(expectedChannelID, res.ChannelId)

					events := ctx.EventManager().Events()
					suite.Require().Len(events, 2)
					suite.Require().Equal(events[0].Type, channeltypes.EventTypeChannelOpenInit)
					suite.Require().Equal(events[1].Type, sdk.EventTypeMessage)

					path.EndpointA.ChannelConfig.PortID = res.PortId
					path.EndpointA.ChannelID = res.ChannelId
					channel := path.EndpointA.GetChannel()
					suite.Require().Equal(expectedOrderding, channel.Ordering)
				} else {
					suite.Require().ErrorIs(err, tc.expErr)
					suite.Require().Nil(res)
				}
			})
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
		expErr   error
	}{
		{
			"success", func() {
			},
			nil,
		},
		{
			"failure - owner address is empty", func() {
				msg.Owner = ""
			},
			icatypes.ErrInvalidAccountAddress,
		},
		{
			"failure - active channel does not exist for connection ID", func() {
				msg.Owner = TestOwnerAddress
				msg.ConnectionId = "connection-100"
			},
			icatypes.ErrActiveChannelNotFound,
		},
		{
			"failure - active channel does not exist for port ID", func() {
				msg.Owner = "invalid-owner"
			},
			icatypes.ErrActiveChannelNotFound,
		},
		{
			"failure - controller module does not own capability for this channel", func() {
				msg.Owner = "invalid-owner"
				portID, err := icatypes.NewControllerPortID(msg.Owner)
				suite.Require().NoError(err)

				// set the active channel with the incorrect portID in order to reach the capability check
				suite.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(suite.chainA.GetContext(), path.EndpointA.ConnectionID, portID, path.EndpointA.ChannelID)
			},
			icatypes.ErrActiveChannelNotFound,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()

				owner := TestOwnerAddress
				path = NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

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
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{icaMsg}, icatypes.EncodingProtobuf)
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

				if tc.expErr == nil {
					suite.Require().NoError(err)
					suite.Require().NotNil(res)
				} else {
					suite.Require().ErrorIs(err, tc.expErr)
					suite.Require().Nil(res)
				}
			})
		}
	}
}

// TestUpdateParams tests UpdateParams rpc handler
func (suite *KeeperTestSuite) TestUpdateParams() {
	signer := suite.chainA.GetSimApp().TransferKeeper.GetAuthority()
	testCases := []struct {
		name   string
		msg    *types.MsgUpdateParams
		expErr error
	}{
		{
			"success: valid signer and default params",
			types.NewMsgUpdateParams(signer, types.NewParams(!types.DefaultControllerEnabled)),
			nil,
		},
		{
			"failure: malformed signer address",
			types.NewMsgUpdateParams(ibctesting.InvalidID, types.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: empty signer address",
			types.NewMsgUpdateParams("", types.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: whitespace signer address",
			types.NewMsgUpdateParams("    ", types.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: unauthorized signer address",
			types.NewMsgUpdateParams(ibctesting.TestAccAddress, types.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := suite.chainA.GetSimApp().ICAControllerKeeper.UpdateParams(suite.chainA.GetContext(), tc.msg)
			if tc.expErr == nil {
				suite.Require().NoError(err)
				p := suite.chainA.GetSimApp().ICAControllerKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
