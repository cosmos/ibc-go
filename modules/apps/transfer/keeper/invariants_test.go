package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

func (suite *KeeperTestSuite) TestTotalEscrowPerDenomInvariant() {
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
				amount := math.NewInt(200)
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.DefaultBondDenom, amount)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path := NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)
			ctx := suite.chainA.GetContext()

			amount := math.NewInt(100)

			// send coins from chain A to chain B so that we have them in escrow
			coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
			msg := types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				coin,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				suite.chainA.GetTimeoutHeight(), 0, "",
			)

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(sdk.WrapSDKContext(ctx), msg)
			suite.Require().NoError(err)
			suite.Require().NotNil(res)

			tc.malleate()

			out, broken := keeper.TotalEscrowPerDenomInvariants(&suite.chainA.GetSimApp().TransferKeeper)(ctx)

			if tc.expPass {
				suite.Require().False(broken)
				suite.Require().Empty(out)
			} else {
				suite.Require().True(broken)

				escrow := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(ctx, sdk.DefaultBondDenom)
				expOut := fmt.Sprintf("denom: %s, actual escrow (%s) is < expected escrow (%s)\n", sdk.DefaultBondDenom, amount, escrow)
				suite.Require().Contains(out, expOut)
			}
		})
	}
}
