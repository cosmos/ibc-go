package keeper_test

import (
	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
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
		expPass  bool
	}{
		{
			"success",
			func() {
				interchainAccountAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainB.GetSimApp().AppCodec(), []proto.Message{msg})
				s.Require().NoError(err)

				packetData = icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}
			},
			true,
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
						Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
					},
					&banktypes.MsgSend{
						FromAddress: interchainAccountAddr,
						ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
						Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
					},
				}

				data, err := icatypes.SerializeCosmosTx(s.chainB.GetSimApp().AppCodec(), msgsBankSend)
				s.Require().NoError(err)

				packetData = icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}
			},
			true,
		},
		{
			"data is nil",
			func() {
				packetData = icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: nil,
				}
			},
			false,
		},
		{
			"active channel not found",
			func() {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id"
			},
			false,
		},
		{
			"channel in INIT state - optimistic packet sends fail",
			func() {
				channel, found := s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				s.Require().True(found)

				channel.State = channeltypes.INIT
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
			},
			false,
		},
		{
			"sendPacket fails - channel closed",
			func() {
				err := path.EndpointA.SetChannelState(channeltypes.CLOSED)
				s.Require().NoError(err)
			},
			false,
		},
		{
			"timeout timestamp is not in the future",
			func() {
				interchainAccountAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainB.GetSimApp().AppCodec(), []proto.Message{msg})
				s.Require().NoError(err)

				packetData = icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				timeoutTimestamp = uint64(s.chainA.GetContext().BlockTime().UnixNano())
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.msg, func() {
			s.SetupTest()                 // reset
			timeoutTimestamp = ^uint64(0) // default

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			//nolint: staticcheck // SA1019: ibctesting.FirstConnectionID is deprecated: use path.EndpointA.ConnectionID instead. (staticcheck)
			_, err = s.chainA.GetSimApp().ICAControllerKeeper.SendTx(s.chainA.GetContext(), nil, ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, packetData, timeoutTimestamp)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestOnTimeoutPacket() {
	var path *ibctesting.Path

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest() // reset

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

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

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
