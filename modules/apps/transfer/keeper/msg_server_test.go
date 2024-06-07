package keeper_test

import (
	"encoding/json"
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

	coin2 := sdk.NewCoin("bond", sdkmath.NewInt(100))
	testCoins := append(ibctesting.TestCoins, coin2) //nolint:gocritic

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
							{Denom: "bond", Enabled: true},
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
				msg.Sender = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
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
			)

			// send some coins of the second denom from bank module to the sender account as well
			err := suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(coin2))
			suite.Require().NoError(err)
			err = suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, suite.chainA.SenderAccount.GetAddress(), sdk.NewCoins(coin2))
			suite.Require().NoError(err)

			tc.malleate()

			ctx := suite.chainA.GetContext()

			var tokens []types.Token
			for _, coin := range msg.GetCoins() {
				token, err := suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(ctx, coin)
				suite.Require().NoError(err)
				tokens = append(tokens, token)
			}

			jsonTokens, err := json.Marshal(types.Tokens(tokens))
			suite.Require().NoError(err)

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(ctx, msg)

			// Verify events
			var expEvents []abci.Event
			events := ctx.EventManager().Events().ToABCIEvents()

			expEvents = sdk.Events{
				sdk.NewEvent(types.EventTypeTransfer,
					sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
					sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
					sdk.NewAttribute(types.AttributeKeyTokens, string(jsonTokens)),
					sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
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
