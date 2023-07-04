package keeper_test

import (
	"time"

	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestRegisterInterchainAccount_MsgServer() {
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
				capability := s.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(s.chainA.GetContext(), TestPortID)
				err := s.chainA.GetSimApp().TransferKeeper.ClaimCapability(s.chainA.GetContext(), capability, host.PortPath(TestPortID))
				s.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB)
		s.coordinator.SetupConnections(path)

		msg = types.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, ibctesting.TestAccAddress, "")

		tc.malleate()

		ctx := s.chainA.GetContext()
		msgServer := keeper.NewMsgServerImpl(&s.chainA.GetSimApp().ICAControllerKeeper)
		res, err := msgServer.RegisterInterchainAccount(ctx, msg)

		if tc.expPass {
			s.Require().NoError(err)
			s.Require().NotNil(res)
			s.Require().Equal(expectedChannelID, res.ChannelId)

			events := ctx.EventManager().Events()
			s.Require().Len(events, 2)
			s.Require().Equal(events[0].Type, channeltypes.EventTypeChannelOpenInit)
			s.Require().Equal(events[1].Type, sdk.EventTypeMessage)
		} else {
			s.Require().Error(err)
			s.Require().Nil(res)
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
				s.Require().NoError(err)

				// set the active channel with the incorrect portID in order to reach the capability check
				s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), path.EndpointA.ConnectionID, portID, path.EndpointA.ChannelID)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			owner := TestOwnerAddress
			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

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
				Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
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
			msgServer := keeper.NewMsgServerImpl(&s.chainA.GetSimApp().ICAControllerKeeper)
			res, err := msgServer.SendTx(ctx, msg)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
			} else {
				s.Require().Error(err)
				s.Require().Nil(res)
			}
		})
	}
}

// TestUpdateParams tests UpdateParams rpc handler
func (s *KeeperTestSuite) TestUpdateParams() {
	validAuthority := s.chainA.GetSimApp().TransferKeeper.GetAuthority()
	testCases := []struct {
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid authority and default params",
			types.NewMsgUpdateParams(validAuthority, types.NewParams(!types.DefaultControllerEnabled)),
			true,
		},
		{
			"failure: malformed authority address",
			types.NewMsgUpdateParams(ibctesting.InvalidID, types.DefaultParams()),
			false,
		},
		{
			"failure: empty authority address",
			types.NewMsgUpdateParams("", types.DefaultParams()),
			false,
		},
		{
			"failure: whitespace authority address",
			types.NewMsgUpdateParams("    ", types.DefaultParams()),
			false,
		},
		{
			"failure: unauthorized authority address",
			types.NewMsgUpdateParams(ibctesting.TestAccAddress, types.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			_, err := s.chainA.GetSimApp().ICAControllerKeeper.UpdateParams(s.chainA.GetContext(), tc.msg)
			if tc.expPass {
				s.Require().NoError(err)
				p := s.chainA.GetSimApp().ICAControllerKeeper.GetParams(s.chainA.GetContext())
				s.Require().Equal(tc.msg.Params, p)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
