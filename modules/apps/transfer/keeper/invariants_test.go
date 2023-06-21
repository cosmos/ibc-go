package keeper_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

func (s *KeeperTestSuite) TestTotalEscrowPerDenomInvariant() {
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
			"fails with broken invariant",
			func() {
				// set amount for denom higher than actual value in escrow
				amount := sdkmath.NewInt(200)
				s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(s.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, amount))
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			path := NewTransferPath(s.chainA, s.chainB)
			s.coordinator.Setup(path)

			amount := sdkmath.NewInt(100)

			// send coins from chain A to chain B so that we have them in escrow
			coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
			msg := types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				coin,
				s.chainA.SenderAccount.GetAddress().String(),
				s.chainB.SenderAccount.GetAddress().String(),
				s.chainA.GetTimeoutHeight(), 0, "",
			)

			res, err := s.chainA.SendMsgs(msg)
			s.Require().NoError(err)
			s.Require().NotNil(res)

			tc.malleate()

			out, broken := keeper.TotalEscrowPerDenomInvariants(&s.chainA.GetSimApp().TransferKeeper)(s.chainA.GetContext())

			if tc.expPass {
				s.Require().False(broken)
				s.Require().Empty(out)
			} else {
				s.Require().True(broken)
				s.Require().NotEmpty(out)
			}
		})
	}
}
