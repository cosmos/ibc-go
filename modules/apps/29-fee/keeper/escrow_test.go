package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

func (suite *KeeperTestSuite) TestEscrowPacketFee() {
	var (
		err error
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup
			refundAcc := suite.chainA.SenderAccount.GetAddress()
			ackFee := &sdk.Coin{Denom: "stake", Amount: sdk.NewInt(100)}
			recieveFee := &sdk.Coin{Denom: "stake", Amount: sdk.NewInt(100)}
			timeoutFee := &sdk.Coin{Denom: "stake", Amount: sdk.NewInt(100)}
			fee := types.Fee{ackFee, recieveFee, timeoutFee}
			packetId := channeltypes.PacketId{ChannelId: "channel-0", PortId: "fee", Sequence: uint64(1)}

			tc.malleate()

			// escrow the packet fee
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, fee, packetId)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}
