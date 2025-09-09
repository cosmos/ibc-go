package keeper_test

import (
	"time"

	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestRegisterInterchainAccount_MsgServer() {
	var (
		msg               *types.MsgRegisterInterchainAccount
		expectedOrderding channeltypes.Order
		expectedChannelID = "channel-0"
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"success: ordering falls back to UNORDERED if not specified",
			func() {
				msg.Ordering = channeltypes.NONE
				expectedOrderding = channeltypes.UNORDERED
			},
			nil,
		},
		{
			"success: non-empty owner address is valid",
			func() {
				msg.Owner = "<invalid-owner>"
			},
			nil,
		},
		{
			"invalid connection id",
			func() {
				msg.ConnectionId = "connection-100"
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"empty address invalid",
			func() {
				msg.Owner = ""
			},
			icatypes.ErrInvalidAccountAddress,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				expectedOrderding = ordering

				s.SetupTest()

				path := NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				msg = types.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, ibctesting.TestAccAddress, "", ordering)

				tc.malleate()

				ctx := s.chainA.GetContext()
				msgServer := keeper.NewMsgServerImpl(s.chainA.GetSimApp().ICAControllerKeeper)
				res, err := msgServer.RegisterInterchainAccount(ctx, msg)

				if tc.expErr == nil {
					s.Require().NoError(err)
					s.Require().NotNil(res)
					s.Require().Equal(expectedChannelID, res.ChannelId)

					events := ctx.EventManager().Events()
					s.Require().Len(events, 2)
					s.Require().Equal(events[0].Type, channeltypes.EventTypeChannelOpenInit)
					s.Require().Equal(events[1].Type, sdk.EventTypeMessage)

					path.EndpointA.ChannelConfig.PortID = res.PortId
					path.EndpointA.ChannelID = res.ChannelId
					channel := path.EndpointA.GetChannel()
					s.Require().Equal(expectedOrderding, channel.Ordering)
				} else {
					s.Require().ErrorIs(err, tc.expErr)
					s.Require().Nil(res)
				}
			})
		}
	}
}

func (s *KeeperTestSuite) TestSubmitTx() {
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
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()

				owner := TestOwnerAddress
				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, owner)
				s.Require().NoError(err)

				portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
				s.Require().NoError(err)

				// get the address of the interchain account stored in state during handshake step
				interchainAccountAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), path.EndpointA.ConnectionID, portID)
				s.Require().True(found)

				// create bank transfer message that will execute on the host chain
				icaMsg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{icaMsg}, icatypes.EncodingProtobuf)
				s.Require().NoError(err)

				packetData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
					Memo: "memo",
				}

				timeoutTimestamp := uint64(s.chainA.GetContext().BlockTime().Add(time.Minute).UnixNano())
				connectionID := path.EndpointA.ConnectionID

				msg = types.NewMsgSendTx(owner, connectionID, timeoutTimestamp, packetData)

				tc.malleate() // malleate mutates test data

				ctx := s.chainA.GetContext()
				msgServer := keeper.NewMsgServerImpl(s.chainA.GetSimApp().ICAControllerKeeper)
				res, err := msgServer.SendTx(ctx, msg)

				if tc.expErr == nil {
					s.Require().NoError(err)
					s.Require().NotNil(res)
				} else {
					s.Require().ErrorIs(err, tc.expErr)
					s.Require().Nil(res)
				}
			})
		}
	}
}

// TestUpdateParams tests UpdateParams rpc handler
func (s *KeeperTestSuite) TestUpdateParams() {
	signer := s.chainA.GetSimApp().TransferKeeper.GetAuthority()
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
		s.Run(tc.name, func() {
			s.SetupTest()
			_, err := s.chainA.GetSimApp().ICAControllerKeeper.UpdateParams(s.chainA.GetContext(), tc.msg)
			if tc.expErr == nil {
				s.Require().NoError(err)
				p := s.chainA.GetSimApp().ICAControllerKeeper.GetParams(s.chainA.GetContext())
				s.Require().Equal(tc.msg.Params, p)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
