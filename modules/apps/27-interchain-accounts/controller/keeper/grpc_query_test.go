package keeper_test

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *KeeperTestSuite) TestQueryInterchainAccount() {
	var req *types.QueryInterchainAccountRequest

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"empty request",
		},
		{
			"empty owner address",
			func() {
				req.Owner = ""
			},
			"failed to generate portID from owner address: owner address cannot be empty: invalid account address",
		},
		{
			"invalid connection, account address not found",
			func() {
				req.ConnectionId = ibctesting.InvalidID
			},
			"failed to retrieve account address",
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				suite.SetupTest()

				path := NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, ibctesting.TestAccAddress)
				suite.Require().NoError(err)

				req = &types.QueryInterchainAccountRequest{
					ConnectionId: ibctesting.FirstConnectionID,
					Owner:        ibctesting.TestAccAddress,
				}

				tc.malleate()

				res, err := suite.chainA.GetSimApp().ICAControllerKeeper.InterchainAccount(suite.chainA.GetContext(), req)

				if tc.errMsg == "" {
					expAddress, exists := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
					suite.Require().True(exists)

					suite.Require().NoError(err)
					suite.Require().Equal(expAddress, res.Address)
				} else {
					suite.Require().ErrorContains(err, tc.errMsg)
				}
			})
		}
	}
}

func (suite *KeeperTestSuite) TestQueryParams() {
	ctx := suite.chainA.GetContext()
	expParams := types.DefaultParams()
	res, _ := suite.chainA.GetSimApp().ICAControllerKeeper.Params(ctx, &types.QueryParamsRequest{})
	suite.Require().Equal(&expParams, res.Params)
}
