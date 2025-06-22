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
func (s *KeeperTestSuite) TestMsgTransfer() {
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
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{
							{Denom: sdk.DefaultBondDenom, Enabled: true},
							{Denom: ibctesting.SecondaryDenom, Enabled: true},
						},
					},
				)
				s.Require().NoError(err)
			},
			nil,
		},
		{
			"failure: send transfers disabled",
			func() {
				s.chainA.GetSimApp().TransferKeeper.SetParams(s.chainA.GetContext(),
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
				msg.Sender = s.chainA.GetSimApp().AccountKeeper.GetModuleAddress(minttypes.ModuleName).String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: bank send disabled",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				s.Require().NoError(err)
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
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewTransferPath(s.chainA, s.chainB)
			path.Setup()

			msg = types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				ibctesting.TestCoin,
				s.chainA.SenderAccount.GetAddress().String(),
				s.chainB.SenderAccount.GetAddress().String(),
				clienttypes.Height{}, s.chainB.GetTimeoutTimestamp(), // only use timeout height
				"memo",
			)

			// send some coins of the second denom from bank module to the sender account as well
			err := s.chainA.GetSimApp().BankKeeper.MintCoins(s.chainA.GetContext(), types.ModuleName, sdk.NewCoins(ibctesting.SecondaryTestCoin))
			s.Require().NoError(err)
			err = s.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(s.chainA.GetContext(), types.ModuleName, s.chainA.SenderAccount.GetAddress(), sdk.NewCoins(ibctesting.SecondaryTestCoin))
			s.Require().NoError(err)

			tc.malleate()

			ctx := s.chainA.GetContext()

			token, err := s.chainA.GetSimApp().TransferKeeper.TokenFromCoin(ctx, msg.Token)
			s.Require().NoError(err)

			res, err := s.chainA.GetSimApp().TransferKeeper.Transfer(ctx, msg)

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
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().NotEqual(res.Sequence, uint64(0))
				ibctesting.AssertEvents(&s.Suite, expEvents, events)
			} else {
				s.Require().Nil(res)
				s.Require().True(errors.Is(err, tc.expError) || strings.Contains(err.Error(), tc.expError.Error()), err.Error())
				s.Require().Len(events, 0)
			}
		})
	}
}

// TestMsgTransfer tests Transfer rpc handler with IBC V2 protocol
func (s *KeeperTestSuite) TestMsgTransferIBCV2() {
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
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{
							{Denom: sdk.DefaultBondDenom, Enabled: true},
							{Denom: ibctesting.SecondaryDenom, Enabled: true},
						},
					},
				)
				s.Require().NoError(err)
			},
			nil,
		},
		{
			"failure: send transfers disabled",
			func() {
				s.chainA.GetSimApp().TransferKeeper.SetParams(s.chainA.GetContext(),
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
				msg.Sender = s.chainA.GetSimApp().AccountKeeper.GetModuleAddress(minttypes.ModuleName).String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: bank send disabled",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				s.Require().NoError(err)
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
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			timeoutTimestamp := uint64(s.chainA.GetContext().BlockTime().Add(time.Hour).Unix())

			msg = types.NewMsgTransfer(
				types.PortID,
				path.EndpointA.ClientID, // use eureka client id
				ibctesting.TestCoin,
				s.chainA.SenderAccount.GetAddress().String(),
				s.chainB.SenderAccount.GetAddress().String(),
				clienttypes.Height{}, timeoutTimestamp, // only use timeout timestamp
				"memo",
			)

			// send some coins of the second denom from bank module to the sender account as well
			err := s.chainA.GetSimApp().BankKeeper.MintCoins(s.chainA.GetContext(), types.ModuleName, sdk.NewCoins(ibctesting.SecondaryTestCoin))
			s.Require().NoError(err)
			err = s.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(s.chainA.GetContext(), types.ModuleName, s.chainA.SenderAccount.GetAddress(), sdk.NewCoins(ibctesting.SecondaryTestCoin))
			s.Require().NoError(err)

			tc.malleate()

			ctx := s.chainA.GetContext()

			token, err := s.chainA.GetSimApp().TransferKeeper.TokenFromCoin(ctx, msg.Token)
			s.Require().NoError(err)

			res, err := s.chainA.GetSimApp().TransferKeeper.Transfer(ctx, msg)

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
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().NotEqual(res.Sequence, uint64(0))
				ibctesting.AssertEvents(&s.Suite, expEvents, events)
			} else {
				s.Require().Nil(res)
				s.Require().True(errors.Is(err, tc.expError) || strings.Contains(err.Error(), tc.expError.Error()), err.Error())
			}
		})
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
		s.Run(tc.name, func() {
			s.SetupTest()
			_, err := s.chainA.GetSimApp().TransferKeeper.UpdateParams(s.chainA.GetContext(), tc.msg)
			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
