package keeper_test

import (
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

func (suite *KeeperTestSuite) TestRegisterCounterpartyAddress() {
	var (
		addr  string
		addr2 string
	)

	testCases := []struct {
		name     string
		expPass  bool
		malleate func()
	}{
		{
			"CounterpartyAddress registered",
			true,
			func() {},
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()
		ctx := suite.chainA.GetContext()

		addr = suite.chainA.SenderAccount.GetAddress().String()
		addr2 = suite.chainB.SenderAccount.GetAddress().String()
		msg := types.NewMsgRegisterCounterpartyAddress(addr, addr2)
		tc.malleate()

		_, err := suite.chainA.SendMsgs(msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed

			counterpartyAddress, _ := suite.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyAddress(ctx, suite.chainA.SenderAccount.GetAddress())
			suite.Require().Equal(addr2, counterpartyAddress.String())
		} else {
			suite.Require().Error(err)
		}
	}
}
