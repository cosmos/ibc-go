package keeper_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// TestMsgTransfer tests Transfer rpc handler
func (s *KeeperTestSuite) TestMsgTransfer() {
	var msg *types.MsgTransfer

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"bank send enabled for denom",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: true}},
					},
				)
				s.Require().NoError(err)
			},
			true,
		},
		{
			"send transfers disabled",
			func() {
				s.chainA.GetSimApp().TransferKeeper.SetParams(s.chainA.GetContext(),
					types.Params{
						SendEnabled: false,
					},
				)
			},
			false,
		},
		{
			"invalid sender",
			func() {
				msg.Sender = "address"
			},
			false,
		},
		{
			"sender is a blocked address",
			func() {
				msg.Sender = s.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
			},
			false,
		},
		{
			"bank send disabled for denom",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				s.Require().NoError(err)
			},
			false,
		},
		{
			"channel does not exist",
			func() {
				msg.SourceChannel = "channel-100"
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path := ibctesting.NewTransferPath(s.chainA, s.chainB)
			s.coordinator.Setup(path)

			coin := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))
			msg = types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				coin, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(),
				s.chainB.GetTimeoutHeight(), 0, // only use timeout height
				"memo",
			)

			tc.malleate()

			ctx := s.chainA.GetContext()
			res, err := s.chainA.GetSimApp().TransferKeeper.Transfer(sdk.WrapSDKContext(ctx), msg)

			// Verify events
			events := ctx.EventManager().Events().ToABCIEvents()
			expEvents := ibctesting.EventsMap{
				"ibc_transfer": {
					"sender":   s.chainA.SenderAccount.GetAddress().String(),
					"receiver": s.chainB.SenderAccount.GetAddress().String(),
					"amount":   coin.Amount.String(),
					"denom":    coin.Denom,
					"memo":     "memo",
				},
			}

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().NotEqual(res.Sequence, uint64(0))
				ibctesting.AssertEvents(&s.Suite, expEvents, events)
			} else {
				s.Require().Error(err)
				s.Require().Nil(res)
				s.Require().Len(events, 0)
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
			types.NewMsgUpdateParams(validAuthority, types.DefaultParams()),
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
			_, err := s.chainA.GetSimApp().TransferKeeper.UpdateParams(s.chainA.GetContext(), tc.msg)
			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
