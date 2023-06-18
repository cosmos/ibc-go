package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestQueryInterchainAccount() {
	var req *types.QueryInterchainAccountRequest

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
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"empty owner address",
			func() {
				req.Owner = ""
			},
			false,
		},
		{
			"invalid connection, account address not found",
			func() {
				req.ConnectionId = "invalid-connection-id"
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path := NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, ibctesting.TestAccAddress)
			s.Require().NoError(err)

			req = &types.QueryInterchainAccountRequest{
				ConnectionId: ibctesting.FirstConnectionID,
				Owner:        ibctesting.TestAccAddress,
			}

			tc.malleate()

			res, err := s.chainA.GetSimApp().ICAControllerKeeper.InterchainAccount(sdk.WrapSDKContext(s.chainA.GetContext()), req)

			if tc.expPass {
				expAddress, exists := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(exists)

				s.Require().NoError(err)
				s.Require().Equal(expAddress, res.Address)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryParams() {
	ctx := sdk.WrapSDKContext(s.chainA.GetContext())
	expParams := types.DefaultParams()
	res, _ := s.chainA.GetSimApp().ICAControllerKeeper.Params(ctx, &types.QueryParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}
