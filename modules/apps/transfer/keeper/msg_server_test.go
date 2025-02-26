package keeper_test

import (
	"errors"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clienttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// TestMsgTransfer tests Transfer rpc handler
func (suite *KeeperTestSuite) TestMsgTransfer() {
	var msg *types.MsgTransfer
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				msg.Token = ibctesting.TestCoin
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
			"failure: bank send disabled",
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
			clienttypesv2.ErrCounterpartyNotFound,
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
				ibctesting.TestCoin,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				clienttypes.Height{}, suite.chainB.GetTimeoutTimestamp(), // only use timeout height
				"memo",
			)

			// send some coins of the second denom from bank module to the sender account as well
			err := suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(ibctesting.SecondaryTestCoin))
			suite.Require().NoError(err)
			err = suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, suite.chainA.SenderAccount.GetAddress(), sdk.NewCoins(ibctesting.SecondaryTestCoin))
			suite.Require().NoError(err)

			tc.malleate()

			ctx := suite.chainA.GetContext()

			token, err := suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(ctx, msg.Token)
			suite.Require().NoError(err)

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(ctx, msg)

			// Verify events
			var expEvents []abci.Event
			events := ctx.EventManager().Events().ToABCIEvents()

			expEvents = sdk.Events{
				sdk.NewEvent(types.EventTypeTransfer,
					sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
					sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
					sdk.NewAttribute(types.AttributeKeyDenom, token.Denom.Path()),
					sdk.NewAttribute(types.AttributeKeyAmount, msg.Token.Amount.String()),
					sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
				),
				sdk.NewEvent(
					sdk.EventTypeMessage,
					sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
				),
			}.ToABCIEvents()

			if tc.expError == nil {
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

// TestMsgTransferIBCV2 tests Transfer rpc handler with IBC V2 protocol
func (suite *KeeperTestSuite) TestMsgTransferIBCV2() {
	var msg *types.MsgTransfer
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				msg.Token = ibctesting.TestCoin
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
			"failure: bank send disabled",
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
			"failure: client does not exist",
			func() {
				msg.SourceChannel = "07-tendermint-500"
			},
			clienttypesv2.ErrCounterpartyNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Hour).Unix())

			msg = types.NewMsgTransfer(
				types.PortID,
				path.EndpointA.ClientID, // use eureka client id
				ibctesting.TestCoin,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				clienttypes.Height{}, timeoutTimestamp, // only use timeout timestamp
				"memo",
			)

			// send some coins of the second denom from bank module to the sender account as well
			err := suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(ibctesting.SecondaryTestCoin))
			suite.Require().NoError(err)
			err = suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, suite.chainA.SenderAccount.GetAddress(), sdk.NewCoins(ibctesting.SecondaryTestCoin))
			suite.Require().NoError(err)

			tc.malleate()

			ctx := suite.chainA.GetContext()

			token, err := suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(ctx, msg.Token)
			suite.Require().NoError(err)

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(ctx, msg)

			// Verify events
			var expEvents []abci.Event
			events := ctx.EventManager().Events().ToABCIEvents()

			expEvents = sdk.Events{
				sdk.NewEvent(types.EventTypeTransfer,
					sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
					sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
					sdk.NewAttribute(types.AttributeKeyDenom, token.Denom.Path()),
					sdk.NewAttribute(types.AttributeKeyAmount, msg.Token.Amount.String()),
					sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
				),
				sdk.NewEvent(
					sdk.EventTypeMessage,
					sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
				),
			}.ToABCIEvents()

			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotEqual(res.Sequence, uint64(0))
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			} else {
				suite.Require().Nil(res)
				suite.Require().True(errors.Is(err, tc.expError) || strings.Contains(err.Error(), tc.expError.Error()), err.Error())
			}
		})
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
			types.NewMsgUpdateParams(signer, types.DefaultParams()),
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
			_, err := suite.chainA.GetSimApp().TransferKeeper.UpdateParams(suite.chainA.GetContext(), tc.msg)
			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
