package types_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *TypesTestSuite) TestTransferAuthorizationAccept() {
	var (
		msgTransfer   types.MsgTransfer
		transferAuthz types.TransferAuthorization
	)

	testCases := []struct {
		name         string
		malleate     func()
		assertResult func(res authz.AcceptResponse, err error)
	}{
		{
			"success",
			func() {},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				s.Require().True(res.Accept)
				s.Require().True(res.Delete)
				s.Require().Nil(res.Updated)
			},
		},
		{
			"success: with spend limit updated",
			func() {
				msgTransfer.Token = sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(50))
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				s.Require().True(res.Accept)
				s.Require().False(res.Delete)

				updatedAuthz, ok := res.Updated.(*types.TransferAuthorization)
				s.Require().True(ok)

				isEqual := updatedAuthz.Allocations[0].SpendLimit.IsEqual(sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(50))))
				s.Require().True(isEqual)
			},
		},
		{
			"success: with empty allow list",
			func() {
				transferAuthz.Allocations[0].AllowList = []string{}
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				s.Require().True(res.Accept)
				s.Require().True(res.Delete)
				s.Require().Nil(res.Updated)
			},
		},
		{
			"success: with multiple allocations",
			func() {
				alloc := types.Allocation{
					SourcePort:    ibctesting.MockPort,
					SourceChannel: "channel-9",
					SpendLimit:    ibctesting.TestCoins,
				}

				transferAuthz.Allocations = append(transferAuthz.Allocations, alloc)
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				s.Require().True(res.Accept)
				s.Require().False(res.Delete)

				updatedAuthz, ok := res.Updated.(*types.TransferAuthorization)
				s.Require().True(ok)

				// assert spent spendlimit is removed from the list
				s.Require().Len(updatedAuthz.Allocations, 1)
			},
		},
		{
			"success: with unlimited spend limit of max uint256",
			func() {
				transferAuthz.Allocations[0].SpendLimit = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, types.UnboundedSpendLimit()))
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				s.Require().True(res.Accept)
				s.Require().False(res.Delete)
				s.Require().Nil(res.Updated)
			},
		},
		{
			"test multiple coins does not overspend",
			func() {
				transferAuthz.Allocations[0].SpendLimit = transferAuthz.Allocations[0].SpendLimit.Add(
					sdk.NewCoins(
						sdk.NewCoin("test-denom", sdkmath.NewInt(100)),
						sdk.NewCoin("test-denom2", sdkmath.NewInt(100)),
					)...,
				)
				msgTransfer.Token = sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(50))
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				updatedTransferAuthz, ok := res.Updated.(*types.TransferAuthorization)
				s.Require().True(ok)

				remainder := updatedTransferAuthz.Allocations[0].SpendLimit.AmountOf(sdk.DefaultBondDenom)
				s.Require().True(sdkmath.NewInt(50).Equal(remainder))

				remainder = updatedTransferAuthz.Allocations[0].SpendLimit.AmountOf("test-denom")
				s.Require().True(sdkmath.NewInt(100).Equal(remainder))

				remainder = updatedTransferAuthz.Allocations[0].SpendLimit.AmountOf("test-denom2")
				s.Require().True(sdkmath.NewInt(100).Equal(remainder))
			},
		},
		{
			"no spend limit set for MsgTransfer port/channel",
			func() {
				msgTransfer.SourcePort = ibctesting.MockPort
				msgTransfer.SourceChannel = "channel-9"
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().Error(err)
			},
		},
		{
			"requested transfer amount is more than the spend limit",
			func() {
				msgTransfer.Token = sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1000))
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().Error(err)
			},
		},
		{
			"receiver address not permitted via allow list",
			func() {
				msgTransfer.Receiver = s.chainB.SenderAccount.GetAddress().String()
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().Error(err)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path := NewTransferPath(s.chainA, s.chainB)
			s.coordinator.Setup(path)

			transferAuthz = types.TransferAuthorization{
				Allocations: []types.Allocation{
					{
						SourcePort:    path.EndpointA.ChannelConfig.PortID,
						SourceChannel: path.EndpointA.ChannelID,
						SpendLimit:    ibctesting.TestCoins,
						AllowList:     []string{ibctesting.TestAccAddress},
					},
				},
			}

			msgTransfer = types.MsgTransfer{
				SourcePort:    path.EndpointA.ChannelConfig.PortID,
				SourceChannel: path.EndpointA.ChannelID,
				Token:         ibctesting.TestCoin,
				Sender:        s.chainA.SenderAccount.GetAddress().String(),
				Receiver:      ibctesting.TestAccAddress,
				TimeoutHeight: s.chainB.GetTimeoutHeight(),
			}

			tc.malleate()

			res, err := transferAuthz.Accept(s.chainA.GetContext(), &msgTransfer)
			tc.assertResult(res, err)
		})
	}
}

func (s *TypesTestSuite) TestTransferAuthorizationMsgTypeURL() {
	var transferAuthz types.TransferAuthorization
	s.Require().Equal(sdk.MsgTypeURL(&types.MsgTransfer{}), transferAuthz.MsgTypeURL(), "invalid type url for transfer authorization")
}

func (s *TypesTestSuite) TestTransferAuthorizationValidateBasic() {
	var transferAuthz types.TransferAuthorization

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
			"success: empty allow list",
			func() {
				transferAuthz.Allocations[0].AllowList = []string{}
			},
			true,
		},
		{
			"success: with multiple allocations",
			func() {
				allocation := types.Allocation{
					SourcePort:    types.PortID,
					SourceChannel: "channel-1",
					SpendLimit:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
					AllowList:     []string{},
				}

				transferAuthz.Allocations = append(transferAuthz.Allocations, allocation)
			},
			true,
		},
		{
			"success: with unlimited spend limit of max uint256",
			func() {
				transferAuthz.Allocations[0].SpendLimit = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, types.UnboundedSpendLimit()))
			},
			true,
		},
		{
			"empty allocations",
			func() {
				transferAuthz = types.TransferAuthorization{Allocations: []types.Allocation{}}
			},
			false,
		},
		{
			"nil allocations",
			func() {
				transferAuthz = types.TransferAuthorization{}
			},
			false,
		},
		{
			"nil spend limit coins",
			func() {
				transferAuthz.Allocations[0].SpendLimit = nil
			},
			false,
		},
		{
			"invalid spend limit coins",
			func() {
				transferAuthz.Allocations[0].SpendLimit = sdk.Coins{sdk.Coin{Denom: ""}}
			},
			false,
		},
		{
			"duplicate entry in allow list",
			func() {
				transferAuthz.Allocations[0].AllowList = []string{ibctesting.TestAccAddress, ibctesting.TestAccAddress}
			},
			false,
		},
		{
			"invalid port identifier",
			func() {
				transferAuthz.Allocations[0].SourcePort = ""
			},
			false,
		},
		{
			"invalid channel identifier",
			func() {
				transferAuthz.Allocations[0].SourceChannel = ""
			},
			false,
		},
		{
			name: "duplicate channel ID",
			malleate: func() {
				allocation := types.Allocation{
					SourcePort:    mock.PortID,
					SourceChannel: transferAuthz.Allocations[0].SourceChannel,
					SpendLimit:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
					AllowList:     []string{ibctesting.TestAccAddress},
				}

				transferAuthz.Allocations = append(transferAuthz.Allocations, allocation)
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			transferAuthz = types.TransferAuthorization{
				Allocations: []types.Allocation{
					{
						SourcePort:    mock.PortID,
						SourceChannel: ibctesting.FirstChannelID,
						SpendLimit:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
						AllowList:     []string{ibctesting.TestAccAddress},
					},
				},
			}

			tc.malleate()

			err := transferAuthz.ValidateBasic()

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
