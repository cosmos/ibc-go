package keeper_test

import (
	"encoding/json"
	"errors"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

// TestMsgTransfer tests Transfer rpc handler
func (suite *KeeperTestSuite) TestMsgTransfer() {
	var msg *types.MsgTransfer
	var path *ibctesting.Path

	testCoins := ibctesting.TestCoins

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: multiple coins",
			func() {},
			nil,
		},
		{
			"success: single coin",
			func() {
				msg.Tokens = []sdk.Coin{ibctesting.TestCoin}
			},
			nil,
		},
		{
			"bank send enabled for denoms",
			func() {
				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{
							{Denom: sdk.DefaultBondDenom, Enabled: true},
							{Denom: ibctesting.SecondaryDenom, Enabled: true},
						},
					},
				)
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"failure: send transfers disabled",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetParams(suite.chainA.GetContext(),
					types.Params{
						SendEnabled: false,
					},
				)
			},
			types.ErrSendDisabled,
		},
		{
			"failure: invalid sender",
			func() {
				msg.Sender = "address"
			},
			errors.New("decoding bech32 failed"),
		},
		{
			"failure: sender is a blocked address",
			func() {
				msg.Sender = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(minttypes.ModuleName).String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: bank send disabled for one of the denoms",
			func() {
				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				suite.Require().NoError(err)
			},
			types.ErrSendDisabled,
		},
		{
			"failure: channel does not exist",
			func() {
				msg.SourceChannel = "channel-100"
			},
			channeltypes.ErrChannelNotFound,
		},
		{
			"failure: multidenom with ics20-1",
			func() {
				// explicitly set to ics20-1 which does not support multi-denom
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.Version = types.V1 })
			},
			ibcerrors.ErrInvalidRequest,
		},
		{
			"failure: cannot unwind native tokens",
			func() {
				msg.Forwarding = types.NewForwarding(true)
				msg.Tokens = []sdk.Coin{ibctesting.TestCoin}
			},
			types.ErrInvalidForwarding,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			msg = types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				testCoins,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				suite.chainB.GetTimeoutHeight(), 0, // only use timeout height
				"memo",
				nil,
			)

			// send some coins of the second denom from bank module to the sender account as well
			err := suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(ibctesting.SecondaryTestCoin))
			suite.Require().NoError(err)
			err = suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, suite.chainA.SenderAccount.GetAddress(), sdk.NewCoins(ibctesting.SecondaryTestCoin))
			suite.Require().NoError(err)

			tc.malleate()

			ctx := suite.chainA.GetContext()

			var tokens []types.Token
			for _, coin := range msg.GetCoins() {
				token, err := suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(ctx, coin)
				suite.Require().NoError(err)
				tokens = append(tokens, token)
			}

			tokensBz, err := json.Marshal(types.Tokens(tokens))
			suite.Require().NoError(err)

			forwardingHopsBz, err := json.Marshal(msg.Forwarding.GetHops())
			suite.Require().NoError(err)

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(ctx, msg)

			// Verify events
			var expEvents []abci.Event
			events := ctx.EventManager().Events().ToABCIEvents()

			expEvents = sdk.Events{
				sdk.NewEvent(types.EventTypeTransfer,
					sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
					sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
					sdk.NewAttribute(types.AttributeKeyTokens, string(tokensBz)),
					sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
					sdk.NewAttribute(types.AttributeKeyForwardingHops, string(forwardingHopsBz)),
				),
				sdk.NewEvent(
					sdk.EventTypeMessage,
					sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
				),
			}.ToABCIEvents()

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotEqual(res.Sequence, uint64(0))
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			} else {
				suite.Require().Nil(res)
				suite.Require().True(errors.Is(err, tc.expError) || strings.Contains(err.Error(), tc.expError.Error()), err.Error())
				suite.Require().Len(events, 0)
			}
		})
	}
}

// TestUpdateParams tests UpdateParams rpc handler
func (suite *KeeperTestSuite) TestUpdateParams() {
	signer := suite.chainA.GetSimApp().TransferKeeper.GetAuthority()
	testCases := []struct {
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid signer and default params",
			types.NewMsgUpdateParams(signer, types.DefaultParams()),
			true,
		},
		{
			"failure: malformed signer address",
			types.NewMsgUpdateParams(ibctesting.InvalidID, types.DefaultParams()),
			false,
		},
		{
			"failure: empty signer address",
			types.NewMsgUpdateParams("", types.DefaultParams()),
			false,
		},
		{
			"failure: whitespace signer address",
			types.NewMsgUpdateParams("    ", types.DefaultParams()),
			false,
		},
		{
			"failure: unauthorized signer address",
			types.NewMsgUpdateParams(ibctesting.TestAccAddress, types.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := suite.chainA.GetSimApp().TransferKeeper.UpdateParams(suite.chainA.GetContext(), tc.msg)
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUnwindHops() {
	var msg *types.MsgTransfer
	var path *ibctesting.Path
	denom := types.NewDenom(ibctesting.TestCoin.Denom, types.NewHop(ibctesting.MockPort, "channel-0"), types.NewHop(ibctesting.MockPort, "channel-1"))
	coins := sdk.NewCoins(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount))
	testCases := []struct {
		name         string
		malleate     func()
		assertResult func(modified *types.MsgTransfer, err error)
	}{
		{
			"success",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), denom)
			},
			func(modified *types.MsgTransfer, err error) {
				suite.Require().NoError(err, "got unexpected error from unwindHops")
				msg.SourceChannel = denom.Trace[0].PortId
				msg.SourcePort = denom.Trace[0].ChannelId
				msg.Forwarding = types.NewForwarding(false, types.NewHop(denom.Trace[1].PortId, denom.Trace[1].ChannelId))
				suite.Require().Equal(*msg, *modified, "expected msg and modified msg are different")
			},
		},
		{
			"success: multiple unwind hops",
			func() {
				denom.Trace = append(denom.Trace, types.NewHop(ibctesting.MockPort, "channel-2"), types.NewHop(ibctesting.MockPort, "channel-3"))
				coins = sdk.NewCoins(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount))
				suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), denom)
				msg.Tokens = coins
			},
			func(modified *types.MsgTransfer, err error) {
				suite.Require().NoError(err, "got unexpected error from unwindHops")
				msg.SourceChannel = denom.Trace[0].PortId
				msg.SourcePort = denom.Trace[0].ChannelId
				msg.Forwarding = types.NewForwarding(false,
					types.NewHop(denom.Trace[3].PortId, denom.Trace[3].ChannelId),
					types.NewHop(denom.Trace[2].PortId, denom.Trace[2].ChannelId),
					types.NewHop(denom.Trace[1].PortId, denom.Trace[1].ChannelId),
				)
				suite.Require().Equal(*msg, *modified, "expected msg and modified msg are different")
			},
		},
		{
			"success - unwind hops are added to existing hops",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), denom)
				msg.Forwarding = types.NewForwarding(true, types.NewHop(ibctesting.MockPort, "channel-2"))
			},
			func(modified *types.MsgTransfer, err error) {
				suite.Require().NoError(err, "got unexpected error from unwindHops")
				msg.SourceChannel = denom.Trace[0].PortId
				msg.SourcePort = denom.Trace[0].ChannelId
				msg.Forwarding = types.NewForwarding(false,
					types.NewHop(denom.Trace[1].PortId, denom.Trace[1].ChannelId),
					types.NewHop(ibctesting.MockPort, "channel-2"),
				)
				suite.Require().Equal(*msg, *modified, "expected msg and modified msg are different")
			},
		},
		{
			"failure: no denom set on keeper",
			func() {},
			func(modified *types.MsgTransfer, err error) {
				suite.Require().ErrorIs(err, types.ErrDenomNotFound)
			},
		},
		{
			"failure: validateBasic() fails due to invalid channelID",
			func() {
				denom.Trace[0].ChannelId = "channel/0"
				coins = sdk.NewCoins(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount))
				msg.Tokens = coins
				suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), denom)
			},
			func(modified *types.MsgTransfer, err error) {
				suite.Require().ErrorContains(err, "invalid source channel ID")
			},
		},
		{
			"failure: denom is native",
			func() {
				denom.Trace = nil
				coins = sdk.NewCoins(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount))
				msg.Tokens = coins
				suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), denom)
			},
			func(modified *types.MsgTransfer, err error) {
				suite.Require().ErrorIs(err, types.ErrInvalidForwarding)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			msg = types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				coins,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				clienttypes.ZeroHeight(),
				suite.chainA.GetTimeoutTimestamp(),
				"memo",
				types.NewForwarding(true),
			)

			tc.malleate()
			gotMsg, err := suite.chainA.GetSimApp().TransferKeeper.UnwindHops(suite.chainA.GetContext(), msg)
			tc.assertResult(gotMsg, err)
		})
	}
}
