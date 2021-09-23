package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
)

func (suite *KeeperTestSuite) TestQueryInterchainAccountAddress() {
	var (
		req     *types.QueryInterchainAccountAddressRequest
		expAddr string
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid counterparty portID",
			func() {
				req = &types.QueryInterchainAccountAddressRequest{
					CounterpartyPortId: "   ",
				}
			},
			false,
		},
		{
			"interchain account address not found",
			func() {
				req = &types.QueryInterchainAccountAddressRequest{
					CounterpartyPortId: "ics-27",
				}
			},
			false,
		},
		{
			"success",
			func() {
				expAddr = authtypes.NewBaseAccountWithAddress(types.GenerateAddress("ics-27")).GetAddress().String()
				req = &types.QueryInterchainAccountAddressRequest{
					CounterpartyPortId: "ics-27",
				}

				suite.chainA.GetSimApp().ICAKeeper.SetInterchainAccountAddress(suite.chainA.GetContext(), "ics-27", expAddr)
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())

			res, err := suite.chainA.GetSimApp().ICAKeeper.InterchainAccountAddress(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expAddr, res.GetInterchainAccountAddress())
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
