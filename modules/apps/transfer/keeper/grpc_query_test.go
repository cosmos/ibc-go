package keeper_test

import (
	"errors"
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestQueryDenom() {
	var (
		req      *types.QueryDenomRequest
		expDenom types.Denom
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"success: correct ibc denom",
			func() {
				expDenom = types.NewDenom(
					"uatom",                                //nolint:goconst
					types.NewHop("transfer", "channelToA"), //nolint:goconst
					types.NewHop("transfer", "channelToB"), //nolint:goconst
				)
				suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), expDenom)

				req = &types.QueryDenomRequest{
					Hash: expDenom.IBCDenom(),
				}
			},
			nil,
		},
		{
			"success: correct hex hash",
			func() {
				expDenom = types.NewDenom(
					"uatom",                                //nolint:goconst
					types.NewHop("transfer", "channelToA"), //nolint:goconst
					types.NewHop("transfer", "channelToB"), //nolint:goconst
				)
				suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), expDenom)

				req = &types.QueryDenomRequest{
					Hash: expDenom.Hash().String(),
				}
			},
			nil,
		},
		{
			"failure: invalid hash",
			func() {
				req = &types.QueryDenomRequest{
					Hash: "!@#!@#!",
				}
			},
			errors.New("invalid denom trace hash"),
		},
		{
			"failure: not found denom trace",
			func() {
				expDenom = types.NewDenom(
					"uatom",                                //nolint:goconst
					types.NewHop("transfer", "channelToA"), //nolint:goconst
					types.NewHop("transfer", "channelToB"), //nolint:goconst
				)

				req = &types.QueryDenomRequest{
					Hash: expDenom.IBCDenom(),
				}
			},
			errors.New("denomination not found"),
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			tc := tc
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			res, err := suite.chainA.GetSimApp().TransferKeeper.Denom(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(&expDenom, res.Denom)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr, err.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryDenoms() {
	var (
		req       *types.QueryDenomsRequest
		expDenoms = types.Denoms(nil)
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty pagination",
			func() {
				req = &types.QueryDenomsRequest{}
			},
			nil,
		},
		{
			"success",
			func() {
				expDenoms = append(expDenoms, types.NewDenom("uatom"))
				expDenoms = append(expDenoms, types.NewDenom("uatom", types.NewHop("transfer", "channelToB")))
				expDenoms = append(expDenoms, types.NewDenom("uatom", types.NewHop("transfer", "channelToA"), types.NewHop("transfer", "channelToB")))

				for _, trace := range expDenoms {
					suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), trace)
				}

				req = &types.QueryDenomsRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			res, err := suite.chainA.GetSimApp().TransferKeeper.Denoms(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expDenoms.Sort(), res.Denoms)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr, err.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryParams() {
	ctx := suite.chainA.GetContext()
	expParams := types.DefaultParams()
	res, _ := suite.chainA.GetSimApp().TransferKeeper.Params(ctx, &types.QueryParamsRequest{})
	suite.Require().Equal(&expParams, res.Params)
}

func (suite *KeeperTestSuite) TestQueryDenomHash() {
	reqDenom := types.NewDenom("uatom", types.NewHop("transfer", "channelToA"), types.NewHop("transfer", "channelToB"))

	var (
		req     *types.QueryDenomHashRequest
		expHash = reqDenom.Hash().String()
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"invalid trace",
			func() {
				req = &types.QueryDenomHashRequest{
					Trace: "transfer%%/channel-1/transfer/channel-1/uatom",
				}
			},
			errors.New("invalid trace"),
		},
		{
			"not found denom trace",
			func() {
				req = &types.QueryDenomHashRequest{
					Trace: "transfer/channelToC/uatom",
				}
			},
			errors.New("denomination not found"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			req = &types.QueryDenomHashRequest{
				Trace: reqDenom.Path(),
			}
			suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), reqDenom)

			tc.malleate()
			ctx := suite.chainA.GetContext()

			res, err := suite.chainA.GetSimApp().TransferKeeper.DenomHash(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expHash, res.Hash)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr, err.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestEscrowAddress() {
	var req *types.QueryEscrowAddressRequest
	var path *ibctesting.Path

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				req = &types.QueryEscrowAddressRequest{
					PortId:    ibctesting.TransferPort,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			nil,
		},
		{
			"failure - channel not found",
			func() {
				req = &types.QueryEscrowAddressRequest{
					PortId:    ibctesting.InvalidID,
					ChannelId: ibctesting.FirstChannelID,
				}
			},
			errors.New("channel not found"),
		},
		{
			"failure - empty channelID",
			func() {
				req = &types.QueryEscrowAddressRequest{
					PortId:    ibctesting.TransferPort,
					ChannelId: "",
				}
			},
			errors.New("identifier cannot be blank"),
		},
		{
			"failure - empty portID",
			func() {
				req = &types.QueryEscrowAddressRequest{
					PortId:    "",
					ChannelId: ibctesting.FirstChannelID,
				}
			},
			errors.New("identifier cannot be blank"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			tc.malleate()
			ctx := suite.chainA.GetContext()

			res, err := suite.chainA.GetSimApp().TransferKeeper.EscrowAddress(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				expected := types.GetEscrowAddress(ibctesting.TransferPort, path.EndpointA.ChannelID).String()
				suite.Require().Equal(expected, res.EscrowAddress)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr, err.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestTotalEscrowForDenom() {
	var (
		req             *types.QueryTotalEscrowForDenomRequest
		expEscrowAmount sdkmath.Int
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"valid native denom with escrow amount < 2^63",
			func() {
				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: sdk.DefaultBondDenom,
				}

				expEscrowAmount = sdkmath.NewInt(100)
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, expEscrowAmount))
			},
			nil,
		},
		{
			"valid ibc denom with escrow amount > 2^63",
			func() {
				denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop("transfer", "channel-0"))

				suite.chainA.GetSimApp().TransferKeeper.SetDenom(suite.chainA.GetContext(), denom)
				expEscrowAmount, ok := sdkmath.NewIntFromString("100000000000000000000")
				suite.Require().True(ok)
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, expEscrowAmount))

				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: denom.IBCDenom(),
				}
			},
			nil,
		},
		{
			"valid ibc denom treated as native denom",
			func() {
				denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop("transfer", "channel-0"))

				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: denom.IBCDenom(),
				}
			},
			nil, // denom trace is not found, thus the denom is considered a native token
		},
		{
			"invalid ibc denom treated as valid native denom",
			func() {
				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: "ibc/123",
				}
			},
			nil, // the ibc denom does not contain a valid hash, thus the denom is considered a native token
		},
		{
			"invalid denom",
			func() {
				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: "??𓃠🐾??",
				}
			},
			errors.New("invalid denom"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			expEscrowAmount = sdkmath.ZeroInt()
			tc.malleate()
			ctx := suite.chainA.GetContext()

			res, err := suite.chainA.GetSimApp().TransferKeeper.TotalEscrowForDenom(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().Equal(expEscrowAmount, res.Amount.Amount)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr, err.Error())
				suite.Require().Error(err)
			}
		})
	}
}
