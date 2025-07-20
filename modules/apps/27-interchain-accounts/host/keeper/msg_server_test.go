package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

func (s *KeeperTestSuite) TestModuleQuerySafe() {
	var (
		msg          *types.MsgModuleQuerySafe
		expResponses [][]byte
	)
	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				balanceQueryBz, err := banktypes.NewQueryBalanceRequest(s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom).Marshal()
				s.Require().NoError(err)

				queryReq := types.QueryRequest{
					Path: "/cosmos.bank.v1beta1.Query/Balance",
					Data: balanceQueryBz,
				}

				msg = types.NewMsgModuleQuerySafe(s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(), []types.QueryRequest{queryReq})

				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

				expResp := banktypes.QueryBalanceResponse{Balance: &balance}
				expRespBz, err := expResp.Marshal()
				s.Require().NoError(err)

				expResponses = [][]byte{expRespBz}
			},
			nil,
		},
		{
			"success: multiple queries",
			func() {
				balanceQueryBz, err := banktypes.NewQueryBalanceRequest(s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom).Marshal()
				s.Require().NoError(err)

				queryReq := types.QueryRequest{
					Path: "/cosmos.bank.v1beta1.Query/Balance",
					Data: balanceQueryBz,
				}

				paramsQuery := stakingtypes.QueryParamsRequest{}
				paramsQueryBz, err := paramsQuery.Marshal()
				s.Require().NoError(err)

				paramsQueryReq := types.QueryRequest{
					Path: "/cosmos.staking.v1beta1.Query/Params",
					Data: paramsQueryBz,
				}

				msg = types.NewMsgModuleQuerySafe(s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(), []types.QueryRequest{queryReq, paramsQueryReq})

				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

				expResp := banktypes.QueryBalanceResponse{Balance: &balance}
				expRespBz, err := expResp.Marshal()
				s.Require().NoError(err)

				params, err := s.chainA.GetSimApp().StakingKeeper.GetParams(s.chainA.GetContext())
				s.Require().NoError(err)
				expParamsResp := stakingtypes.QueryParamsResponse{Params: params}
				expParamsRespBz, err := expParamsResp.Marshal()
				s.Require().NoError(err)

				expResponses = [][]byte{expRespBz, expParamsRespBz}
			},
			nil,
		},
		{
			"failure: not module query safe",
			func() {
				balanceQueryBz, err := banktypes.NewQueryBalanceRequest(s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom).Marshal()
				s.Require().NoError(err)

				queryReq := types.QueryRequest{
					Path: "/cosmos.bank.v1beta1.Query/Balance",
					Data: balanceQueryBz,
				}

				paramsQuery := transfertypes.QueryParamsRequest{}
				paramsQueryBz, err := paramsQuery.Marshal()
				s.Require().NoError(err)

				paramsQueryReq := types.QueryRequest{
					Path: "/ibc.applications.transfer.v1.Query/Params",
					Data: paramsQueryBz,
				}

				msg = types.NewMsgModuleQuerySafe(s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(), []types.QueryRequest{queryReq, paramsQueryReq})
			},
			ibcerrors.ErrInvalidRequest,
		},
		{
			"failure: invalid query path",
			func() {
				balanceQueryBz, err := banktypes.NewQueryBalanceRequest(s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom).Marshal()
				s.Require().NoError(err)

				queryReq := types.QueryRequest{
					Path: "/cosmos.invalid.Query/Invalid",
					Data: balanceQueryBz,
				}

				msg = types.NewMsgModuleQuerySafe(s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(), []types.QueryRequest{queryReq})
			},
			ibcerrors.ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			// reset
			msg = nil
			expResponses = nil

			tc.malleate()

			ctx := s.chainA.GetContext()
			msgServer := keeper.NewMsgServerImpl(s.chainA.GetSimApp().ICAHostKeeper)
			res, err := msgServer.ModuleQuerySafe(ctx, msg)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)

				s.Require().ElementsMatch(expResponses, res.Responses)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
				s.Require().Nil(res)
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpdateParams() {
	testCases := []struct {
		name   string
		msg    *types.MsgUpdateParams
		expErr error
	}{
		{
			"success",
			types.NewMsgUpdateParams(s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(), types.DefaultParams()),
			nil,
		},
		{
			"invalid signer address",
			types.NewMsgUpdateParams("signer", types.DefaultParams()),
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			ctx := s.chainA.GetContext()
			msgServer := keeper.NewMsgServerImpl(s.chainA.GetSimApp().ICAHostKeeper)
			res, err := msgServer.UpdateParams(ctx, tc.msg)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
				s.Require().Nil(res)
			}
		})
	}
}
