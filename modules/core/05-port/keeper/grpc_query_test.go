package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/testing/mock"
)

func (suite *KeeperTestSuite) TestNegotiateAppVersion() {
	var (
		req        *types.NegotiateAppVersionRequest
		expVersion string
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
			"invalid port ID",
			func() {
				req = &types.NegotiateAppVersionRequest{
					PortId: "",
				}
			},
			false,
		},
		{
			"module not found",
			func() {
				req = &types.NegotiateAppVersionRequest{
					PortId: "mock-port-id",
				}
			},
			false,
		},
		{
			"version negotiation failure",
			func() {

				expVersion = mock.Version

				req = &types.NegotiateAppVersionRequest{
					PortId: "mock", // retrieves the mock testing module
					Counterparty: &channeltypes.Counterparty{
						PortId:    "mock-port-id",
						ChannelId: "mock-channel-id",
					},
					ProposedVersion: "invalid-proposed-version",
				}
			},
			false,
		},
		{
			"success",
			func() {

				expVersion = mock.Version

				req = &types.NegotiateAppVersionRequest{
					PortId: "mock", // retrieves the mock testing module
					Counterparty: &channeltypes.Counterparty{
						PortId:    "mock-port-id",
						ChannelId: "mock-channel-id",
					},
					ProposedVersion: mock.Version,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()

			ctx := sdk.WrapSDKContext(suite.ctx)
			res, err := suite.keeper.NegotiateAppVersion(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expVersion, res.Version)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
