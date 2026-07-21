package keeper_test

import (
	"errors"
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

func (s *KeeperTestSuite) TestQueryDenom() {
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
				s.chainA.GetSimApp().TransferKeeper.SetDenom(s.chainA.GetContext(), expDenom)

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
				s.chainA.GetSimApp().TransferKeeper.SetDenom(s.chainA.GetContext(), expDenom)

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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			tc := tc
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			res, err := s.chainA.GetSimApp().TransferKeeper.Denom(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(&expDenom, res.Denom)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expErr, err.Error())
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryDenoms() {
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
					s.chainA.GetSimApp().TransferKeeper.SetDenom(s.chainA.GetContext(), trace)
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
		s.Run(tc.msg, func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			res, err := s.chainA.GetSimApp().TransferKeeper.Denoms(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expDenoms.Sort(), res.Denoms)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expErr, err.Error())
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryParams() {
	ctx := s.chainA.GetContext()
	expParams := types.DefaultParams()
	res, _ := s.chainA.GetSimApp().TransferKeeper.Params(ctx, &types.QueryParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}

func (s *KeeperTestSuite) TestQueryDenomHash() {
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
		s.Run(tc.msg, func() {
			s.SetupTest() // reset

			req = &types.QueryDenomHashRequest{
				Trace: reqDenom.Path(),
			}
			s.chainA.GetSimApp().TransferKeeper.SetDenom(s.chainA.GetContext(), reqDenom)

			tc.malleate()
			ctx := s.chainA.GetContext()

			res, err := s.chainA.GetSimApp().TransferKeeper.DenomHash(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expHash, res.Hash)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expErr, err.Error())
			}
		})
	}
}

func (s *KeeperTestSuite) TestEscrowAddress() {
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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset
			path = ibctesting.NewTransferPath(s.chainA, s.chainB)
			path.Setup()

			tc.malleate()
			ctx := s.chainA.GetContext()

			res, err := s.chainA.GetSimApp().TransferKeeper.EscrowAddress(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				expected := types.GetEscrowAddress(ibctesting.TransferPort, path.EndpointA.ChannelID).String()
				s.Require().Equal(expected, res.EscrowAddress)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expErr, err.Error())
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
		expErr   error
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
			nil,
		},
		{
			"valid ibc denom with escrow amount > 2^63",
			func() {
				denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop("transfer", "channel-0"))

				s.chainA.GetSimApp().TransferKeeper.SetDenom(s.chainA.GetContext(), denom)
				expEscrowAmount, ok := sdkmath.NewIntFromString("100000000000000000000")
				s.Require().True(ok)
				s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(s.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, expEscrowAmount))

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
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			expEscrowAmount = sdkmath.ZeroInt()
			tc.malleate()
			ctx := s.chainA.GetContext()

			res, err := s.chainA.GetSimApp().TransferKeeper.TotalEscrowForDenom(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(expEscrowAmount, res.Amount.Amount)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expErr, err.Error())
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestChannelEscrowForDenom() {
	const channelID = "channel-0"

	ctx := s.chainA.GetContext()
	coin := sdk.NewInt64Coin(sdk.DefaultBondDenom, 100)
	s.chainA.GetSimApp().TransferKeeper.SetChannelEscrowForDenom(ctx, channelID, coin)

	testCases := []struct {
		name      string
		req       *types.QueryChannelEscrowForDenomRequest
		expected  sdk.Coin
		errString string
	}{
		{
			name: "channel identifier",
			req: &types.QueryChannelEscrowForDenomRequest{
				ChannelOrClientId: channelID,
				Denom:             coin.Denom,
			},
			expected: coin,
		},
		{
			name: "client identifier without escrow",
			req: &types.QueryChannelEscrowForDenomRequest{
				ChannelOrClientId: "07-tendermint-0",
				Denom:             coin.Denom,
			},
			expected: sdk.NewInt64Coin(coin.Denom, 0),
		},
		{
			name:      "nil request",
			req:       nil,
			errString: "empty request",
		},
		{
			name: "invalid identifier",
			req: &types.QueryChannelEscrowForDenomRequest{
				ChannelOrClientId: "invalid",
				Denom:             coin.Denom,
			},
			errString: "invalid channel or client identifier",
		},
		{
			name: "invalid denomination",
			req: &types.QueryChannelEscrowForDenomRequest{
				ChannelOrClientId: channelID,
				Denom:             "invalid denom",
			},
			errString: "invalid denom",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			res, err := s.chainA.GetSimApp().TransferKeeper.ChannelEscrowForDenom(ctx, tc.req)
			if tc.errString != "" {
				s.Require().ErrorContains(err, tc.errString)
				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tc.expected, res.Amount)
		})
	}
}

func (s *KeeperTestSuite) TestAllChannelEscrows() {
	ctx := s.chainA.GetContext()
	transferKeeper := s.chainA.GetSimApp().TransferKeeper
	channel0Coins := sdk.NewCoins(sdk.NewInt64Coin("atom", 10), sdk.NewInt64Coin("stake", 20))
	channel1Coin := sdk.NewInt64Coin("stake", 30)
	for _, coin := range channel0Coins {
		transferKeeper.SetChannelEscrowForDenom(ctx, "channel-0", coin)
	}
	transferKeeper.SetChannelEscrowForDenom(ctx, "channel-1", channel1Coin)

	firstPage, err := transferKeeper.AllChannelEscrows(ctx, &types.QueryAllChannelEscrowsRequest{
		Pagination: &query.PageRequest{Limit: 1, CountTotal: true},
	})
	s.Require().NoError(err)
	s.Require().Equal([]types.ChannelEscrowAmount{{ChannelOrClientId: "channel-0", Amount: channel0Coins[0]}}, firstPage.ChannelEscrows)
	s.Require().Equal(uint64(3), firstPage.Pagination.Total)
	s.Require().NotEmpty(firstPage.Pagination.NextKey)

	secondPage, err := transferKeeper.AllChannelEscrows(ctx, &types.QueryAllChannelEscrowsRequest{
		Pagination: &query.PageRequest{Key: firstPage.Pagination.NextKey, Limit: 1},
	})
	s.Require().NoError(err)
	s.Require().Equal([]types.ChannelEscrowAmount{{ChannelOrClientId: "channel-0", Amount: channel0Coins[1]}}, secondPage.ChannelEscrows)
	s.Require().NotEmpty(secondPage.Pagination.NextKey)

	reversePage, err := transferKeeper.AllChannelEscrows(ctx, &types.QueryAllChannelEscrowsRequest{
		Pagination: &query.PageRequest{Limit: 1, Reverse: true},
	})
	s.Require().NoError(err)
	s.Require().Equal([]types.ChannelEscrowAmount{{ChannelOrClientId: "channel-1", Amount: channel1Coin}}, reversePage.ChannelEscrows)

	_, err = transferKeeper.AllChannelEscrows(ctx, nil)
	s.Require().ErrorContains(err, "empty request")
}
