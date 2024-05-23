package keeper_test

import (
	"errors"
	"strings"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// TestMsgTransfer tests Transfer rpc handler
func (suite *KeeperTestSuite) TestMsgTransfer() {
	var msg *types.MsgTransfer
	var path *ibctesting.Path
	var coin1 sdk.Coin
	var coin2 sdk.Coin

	testCases := []struct {
		name       string
		malleate   func()
		expError   error
		multiDenom bool
	}{
		{
			"success: single denom",
			func() {},
			nil,
			false,
		},
		{
			"success: multidenom",
			func() {
				coin2 = sdk.NewCoin("bond", sdkmath.NewInt(100))
				coins := sdk.NewCoins(coin1, coin2)

				// send some coins of the second denom from bank module to the sender account as well
				suite.Require().NoError(suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(coin2)))
				suite.Require().NoError(suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, suite.chainA.SenderAccount.GetAddress(), sdk.NewCoins(coin2)))

				msg = types.NewMsgTransfer(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					coins,
					suite.chainA.SenderAccount.GetAddress().String(),
					suite.chainB.SenderAccount.GetAddress().String(),
					suite.chainB.GetTimeoutHeight(), 0, // only use timeout height
					"memo",
				)
			},
			nil,
			true,
		},
		{
			"bank send enabled for denom",
			func() {
				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: true}},
					},
				)
				suite.Require().NoError(err)
			},
			nil,
			false,
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
			false,
		},
		{
			"failure: invalid sender",
			func() {
				msg.Sender = "address"
			},
			errors.New("decoding bech32 failed"),
			false,
		},
		{
			"failure: sender is a blocked address",
			func() {
				msg.Sender = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
			},
			ibcerrors.ErrUnauthorized,
			false,
		},
		{
			"failure: bank send disabled for denom",
			func() {
				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				suite.Require().NoError(err)
			},
			types.ErrSendDisabled,
			false,
		},
		{
			"failure: bank send disabled for coin in multi coin transfer",
			func() {
				coin2 = sdk.NewCoin("bond", sdkmath.NewInt(100))
				coins := sdk.NewCoins(coin1, coin2)

				// send some coins of the second denom from bank module to the sender account as well
				suite.Require().NoError(suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(coin2)))
				suite.Require().NoError(suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, suite.chainA.SenderAccount.GetAddress(), sdk.NewCoins(coin2)))

				msg = types.NewMsgTransfer(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					coins,
					suite.chainA.SenderAccount.GetAddress().String(),
					suite.chainB.SenderAccount.GetAddress().String(),
					suite.chainB.GetTimeoutHeight(), 0, // only use timeout height
					"memo",
				)

				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: coin2.Denom, Enabled: false}},
					},
				)
				suite.Require().NoError(err)
			},
			types.ErrSendDisabled,
			true,
		},
		{
			"failure: channel does not exist",
			func() {
				msg.SourceChannel = "channel-100"
			},
			ibcerrors.ErrInvalidRequest,
			false,
		},
		{
			"failure: multidenom with ics20-1",
			func() {
				coin2 = sdk.NewCoin("bond", sdkmath.NewInt(100))
				coins := sdk.NewCoins(coin1, coin2)

				// send some coins of the second denom from bank module to the sender account as well
				suite.Require().NoError(suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(coin2)))
				suite.Require().NoError(suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, suite.chainA.SenderAccount.GetAddress(), sdk.NewCoins(coin2)))

				msg = types.NewMsgTransfer(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					coins,
					suite.chainA.SenderAccount.GetAddress().String(),
					suite.chainB.SenderAccount.GetAddress().String(),
					suite.chainB.GetTimeoutHeight(), 0, // only use timeout height
					"memo",
				)

				// explicitly set to ics20-1 which does not support multi-denom
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.Version = types.V1
				})
			},
			ibcerrors.ErrInvalidRequest,
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			coin1 = sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))
			msg = types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				sdk.NewCoins(coin1),
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				suite.chainB.GetTimeoutHeight(), 0, // only use timeout height
				"memo",
			)

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(ctx, msg)

			// Verify events
			events := ctx.EventManager().Events().ToABCIEvents()

			var expEvents []abci.Event
			if tc.multiDenom {
				expEvents = sdk.Events{
					sdk.NewEvent(types.EventTypeTransfer,
						sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
						sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
						sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
						sdk.NewAttribute(types.AttributeKeyTokens, sdk.NewCoins(coin1, coin2).String()),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
					),
				}.ToABCIEvents()
			} else {
				expEvents = sdk.Events{
					sdk.NewEvent(types.EventTypeTransfer,
						sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
						sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
						sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
						sdk.NewAttribute(types.AttributeKeyTokens, coin1.String()),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
					),
				}.ToABCIEvents()
			}

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotEqual(res.Sequence, uint64(0))
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			} else {
				suite.Require().Nil(res)
				suite.Require().Error(err)
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
