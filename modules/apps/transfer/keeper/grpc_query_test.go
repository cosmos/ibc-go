package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestQueryDenomTrace() {
	var (
		req      *types.QueryDenomTraceRequest
		expTrace types.DenomTrace
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success: correct ibc denom",
			func() {
				expTrace.Path = "transfer/channelToA/transfer/channelToB" //nolint:goconst
				expTrace.BaseDenom = "uatom"                              //nolint:goconst
				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(s.chainA.GetContext(), expTrace)

				req = &types.QueryDenomTraceRequest{
					Hash: expTrace.IBCDenom(),
				}
			},
			true,
		},
		{
			"success: correct hex hash",
			func() {
				expTrace.Path = "transfer/channelToA/transfer/channelToB"
				expTrace.BaseDenom = "uatom"
				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(s.chainA.GetContext(), expTrace)

				req = &types.QueryDenomTraceRequest{
					Hash: expTrace.Hash().String(),
				}
			},
			true,
		},
		{
			"failure: invalid hash",
			func() {
				req = &types.QueryDenomTraceRequest{
					Hash: "!@#!@#!",
				}
			},
			false,
		},
		{
			"failure: not found denom trace",
			func() {
				expTrace.Path = "transfer/channelToA/transfer/channelToB"
				expTrace.BaseDenom = "uatom"
				req = &types.QueryDenomTraceRequest{
					Hash: expTrace.IBCDenom(),
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.GetSimApp().TransferKeeper.DenomTrace(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(&expTrace, res.DenomTrace)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryDenomTraces() {
	var (
		req       *types.QueryDenomTracesRequest
		expTraces = types.Traces(nil)
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty pagination",
			func() {
				req = &types.QueryDenomTracesRequest{}
			},
			true,
		},
		{
			"success",
			func() {
				expTraces = append(expTraces, types.DenomTrace{Path: "", BaseDenom: "uatom"})
				expTraces = append(expTraces, types.DenomTrace{Path: "transfer/channelToB", BaseDenom: "uatom"})
				expTraces = append(expTraces, types.DenomTrace{Path: "transfer/channelToA/transfer/channelToB", BaseDenom: "uatom"})

				for _, trace := range expTraces {
					s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(s.chainA.GetContext(), trace)
				}

				req = &types.QueryDenomTracesRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.GetSimApp().TransferKeeper.DenomTraces(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expTraces.Sort(), res.DenomTraces)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryParams() {
	ctx := sdk.WrapSDKContext(s.chainA.GetContext())
	expParams := types.DefaultParams()
	res, _ := s.chainA.GetSimApp().TransferKeeper.Params(ctx, &types.QueryParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}

func (s *KeeperTestSuite) TestQueryDenomHash() {
	reqTrace := types.DenomTrace{
		Path:      "transfer/channelToA/transfer/channelToB",
		BaseDenom: "uatom",
	}

	var (
		req     *types.QueryDenomHashRequest
		expHash = reqTrace.Hash().String()
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"invalid trace",
			func() {
				req = &types.QueryDenomHashRequest{
					Trace: "transfer/channelToA/transfer/",
				}
			},
			false,
		},
		{
			"not found denom trace",
			func() {
				req = &types.QueryDenomHashRequest{
					Trace: "transfer/channelToC/uatom",
				}
			},
			false,
		},
		{
			"success",
			func() {},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			req = &types.QueryDenomHashRequest{
				Trace: reqTrace.GetFullDenomPath(),
			}
			s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(s.chainA.GetContext(), reqTrace)

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.GetSimApp().TransferKeeper.DenomHash(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expHash, res.Hash)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestEscrowAddress() {
	var req *types.QueryEscrowAddressRequest

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				req = &types.QueryEscrowAddressRequest{
					PortId:    ibctesting.TransferPort,
					ChannelId: ibctesting.FirstChannelID,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.GetSimApp().TransferKeeper.EscrowAddress(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				expected := types.GetEscrowAddress(ibctesting.TransferPort, ibctesting.FirstChannelID).String()
				s.Require().Equal(expected, res.EscrowAddress)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestTotalEscrowForDenom() {
	var (
		req             *types.QueryTotalEscrowForDenomRequest
		expEscrowAmount sdkmath.Int
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"valid native denom with escrow amount < 2^63",
			func() {
				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: sdk.DefaultBondDenom,
				}

				expEscrowAmount = sdkmath.NewInt(100)
				s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(s.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, expEscrowAmount))
			},
			true,
		},
		{
			"valid ibc denom with escrow amount > 2^63",
			func() {
				denomTrace := types.DenomTrace{
					Path:      "transfer/channel-0",
					BaseDenom: sdk.DefaultBondDenom,
				}

				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(s.chainA.GetContext(), denomTrace)
				expEscrowAmount, ok := sdkmath.NewIntFromString("100000000000000000000")
				s.Require().True(ok)
				s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(s.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, expEscrowAmount))

				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: denomTrace.IBCDenom(),
				}
			},
			true,
		},
		{
			"valid ibc denom treated as native denom",
			func() {
				denomTrace := types.DenomTrace{
					Path:      "transfer/channel-0",
					BaseDenom: sdk.DefaultBondDenom,
				}

				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: denomTrace.IBCDenom(),
				}
			},
			true, // denom trace is not found, thus the denom is considered a native token
		},
		{
			"invalid ibc denom treated as valid native denom",
			func() {
				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: "ibc/123",
				}
			},
			true, // the ibc denom does not contain a valid hash, thus the denom is considered a native token
		},
		{
			"invalid denom",
			func() {
				req = &types.QueryTotalEscrowForDenomRequest{
					Denom: "??𓃠🐾??",
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			expEscrowAmount = sdkmath.ZeroInt()
			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.GetSimApp().TransferKeeper.TotalEscrowForDenom(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(expEscrowAmount, res.Amount.Amount)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
