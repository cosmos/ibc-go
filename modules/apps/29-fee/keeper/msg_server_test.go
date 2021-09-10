package keeper_test

import (
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

func (suite *KeeperTestSuite) TestRegisterCounterpartyAddress() {
	validAddr := suite.chainA.SenderAccount.GetAddress().String()
	validAddr2 := suite.chainB.SenderAccount.GetAddress().String()

	testCases := []struct {
		msg     *types.MsgRegisterCounterpartyAddress
		expPass bool
	}{
		{
			types.NewMsgRegisterCounterpartyAddress(validAddr, validAddr2),
			true,
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()
		_, err := suite.chainA.SendMsgs(tc.msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed
		} else {
			suite.Require().Error(err)
		}

	}
}
