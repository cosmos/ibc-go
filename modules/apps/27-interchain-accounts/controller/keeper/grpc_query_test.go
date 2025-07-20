package keeper_test

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestQueryInterchainAccount() {
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
			s.Run(tc.name, func() {
				s.SetupTest()

				path := NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, ibctesting.TestAccAddress)
				s.Require().NoError(err)

				req = &types.QueryInterchainAccountRequest{
					ConnectionId: ibctesting.FirstConnectionID,
					Owner:        ibctesting.TestAccAddress,
				}

				tc.malleate()

				res, err := s.chainA.GetSimApp().ICAControllerKeeper.InterchainAccount(s.chainA.GetContext(), req)

				if tc.errMsg == "" {
					expAddress, exists := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
					s.Require().True(exists)

					s.Require().NoError(err)
					s.Require().Equal(expAddress, res.Address)
				} else {
					s.Require().ErrorContains(err, tc.errMsg)
				}
			})
		}
	}
}

func (s *KeeperTestSuite) TestQueryParams() {
	ctx := s.chainA.GetContext()
	expParams := types.DefaultParams()
	res, _ := s.chainA.GetSimApp().ICAControllerKeeper.Params(ctx, &types.QueryParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}
