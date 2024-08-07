package keeper_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestTotalEscrowPerDenomInvariant() {
	testCases := []struct {
		name            string
		coinsToTransfer sdk.Coins
		malleate        func()
		expPass         bool
	}{
		{
			"success",
			sdk.NewCoins(ibctesting.TestCoin, ibctesting.SecondaryTestCoin),
			func() {},
			true,
		},
		{
			"success with single denom",
			sdk.NewCoins(ibctesting.TestCoin),
			func() {},
			true,
		},
		{
			"fails with broken invariant",
			sdk.NewCoins(ibctesting.TestCoin),
			func() {
				// set amount for denom higher than actual value in escrow
				amount := ibctesting.TestCoin.Amount.Add(sdkmath.NewInt(100))
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, amount))
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			msg := types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				tc.coinsToTransfer,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				suite.chainA.GetTimeoutHeight(), 0, "",
				nil,
			)

			res, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err)
			suite.Require().NotNil(res)

			tc.malleate()

			out, broken := keeper.TotalEscrowPerDenomInvariants(&suite.chainA.GetSimApp().TransferKeeper)(suite.chainA.GetContext())

			if tc.expPass {
				suite.Require().False(broken)
				suite.Require().Empty(out)
			} else {
				suite.Require().True(broken)
				suite.Require().NotEmpty(out)
			}
		})
	}
}
