package keeper_test

import (
	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestSendTx() {
	var (
		path             *ibctesting.Path
		packetData       icatypes.InterchainAccountPacketData
		timeoutTimestamp uint64
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				interchainAccountAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainB.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				s.Require().NoError(err)

				packetData = icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}
			},
			nil,
		},
		{
			"success with multiple sdk.Msg",
			func() {
				interchainAccountAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msgsBankSend := []proto.Message{
					&banktypes.MsgSend{
						FromAddress: interchainAccountAddr,
						ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
						Amount:      sdk.NewCoins(ibctesting.TestCoin),
					},
					&banktypes.MsgSend{
						FromAddress: interchainAccountAddr,
						ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
						Amount:      sdk.NewCoins(ibctesting.TestCoin),
					},
				}

				data, err := icatypes.SerializeCosmosTx(s.chainB.GetSimApp().AppCodec(), msgsBankSend, icatypes.EncodingProtobuf)
				s.Require().NoError(err)

				packetData = icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}
			},
			nil,
		},
		{
			"data is nil",
			func() {
				packetData = icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: nil,
				}
			},
			icatypes.ErrInvalidOutgoingData,
		},
		{
			"active channel not found",
			func() {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id" //nolint:goconst
			},
			icatypes.ErrActiveChannelNotFound,
		},
		{
			"channel in INIT state - optimistic packet sends fail",
			func() {
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.INIT })
			},
			icatypes.ErrActiveChannelNotFound,
		},
		{
			"sendPacket fails - channel closed",
			func() {
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
			},
			icatypes.ErrActiveChannelNotFound,
		},
		{
			"controller submodule disabled",
			func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			},
			types.ErrControllerSubModuleDisabled,
		},
		{
			"timeout timestamp is not in the future",
			func() {
				interchainAccountAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainB.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				s.Require().NoError(err)

				packetData = icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				timeoutTimestamp = uint64(s.chainA.GetContext().BlockTime().UnixNano())
			},
			icatypes.ErrInvalidTimeoutTimestamp,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.msg, func() {
				s.SetupTest()                 // reset
				timeoutTimestamp = ^uint64(0) // default

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				s.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				// nolint: staticcheck // SA1019: ibctesting.FirstConnectionID is deprecated: use path.EndpointA.ConnectionID instead. (staticcheck)
				_, err = s.chainA.GetSimApp().ICAControllerKeeper.SendTx(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, packetData, timeoutTimestamp)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().ErrorIs(err, tc.expErr)
				}
			})
		}
	}
}

func (s *KeeperTestSuite) TestOnTimeoutPacket() {
	var path *ibctesting.Path

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.msg, func() {
				s.SetupTest() // reset

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				s.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				packet := channeltypes.NewPacket(
					[]byte{},
					1,
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					clienttypes.NewHeight(0, 100),
					0,
				)

				err = s.chainA.GetSimApp().ICAControllerKeeper.OnTimeoutPacket(s.chainA.GetContext(), packet)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}
