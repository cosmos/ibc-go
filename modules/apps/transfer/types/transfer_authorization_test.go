package types_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	"github.com/cosmos/ibc-go/v6/testing/mock"
)

func (suite *TypesTestSuite) TestTransferAuthorizationAccept() {
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
				suite.Require().NoError(err)

				suite.Require().True(res.Accept)
				suite.Require().True(res.Delete)
				suite.Require().Nil(res.Updated)
			},
		},
		{
			"success: with spend limit updated",
			func() {
				msgTransfer.Token = sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(50))
			},
			func(res authz.AcceptResponse, err error) {
				suite.Require().NoError(err)

				suite.Require().True(res.Accept)
				suite.Require().False(res.Delete)

				updatedAuthz, ok := res.Updated.(*types.TransferAuthorization)
				suite.Require().True(ok)

				isEqual := updatedAuthz.Allocations[0].SpendLimit.IsEqual(sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(50))))
				suite.Require().True(isEqual)
			},
		},
		{
			"success: with empty allow list",
			func() {
				transferAuthz.Allocations[0].AllowList = []string{}
			},
			func(res authz.AcceptResponse, err error) {
				suite.Require().NoError(err)

				suite.Require().True(res.Accept)
				suite.Require().True(res.Delete)
				suite.Require().Nil(res.Updated)
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
				suite.Require().NoError(err)

				suite.Require().True(res.Accept)
				suite.Require().False(res.Delete)

				updatedAuthz, ok := res.Updated.(*types.TransferAuthorization)
				suite.Require().True(ok)

				// assert spent spendlimit is removed from the list
				suite.Require().Len(updatedAuthz.Allocations, 1)
			},
		},
		{
			"success: with unlimited spend limit of max uint256",
			func() {
				transferAuthz.Allocations[0].SpendLimit = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, types.UnboundedSpendLimit()))
			},
			func(res authz.AcceptResponse, err error) {
				suite.Require().NoError(err)

				suite.Require().True(res.Accept)
				suite.Require().False(res.Delete)
				suite.Require().Nil(res.Updated)
			},
		},
		{
			"test multiple coins does not overspend",
			func() {
				transferAuthz.Allocations[0].SpendLimit = transferAuthz.Allocations[0].SpendLimit.Add(
					sdk.NewCoins(
						sdk.NewCoin("test-denom", sdk.NewInt(100)),
						sdk.NewCoin("test-denom2", sdk.NewInt(100)),
					)...,
				)
				msgTransfer.Token = sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(50))
			},
			func(res authz.AcceptResponse, err error) {
				suite.Require().NoError(err)

				updatedTransferAuthz, ok := res.Updated.(*types.TransferAuthorization)
				suite.Require().True(ok)

				remainder := updatedTransferAuthz.Allocations[0].SpendLimit.AmountOf(sdk.DefaultBondDenom)
				suite.Require().True(sdk.NewInt(50).Equal(remainder))

				remainder = updatedTransferAuthz.Allocations[0].SpendLimit.AmountOf("test-denom")
				suite.Require().True(sdk.NewInt(100).Equal(remainder))

				remainder = updatedTransferAuthz.Allocations[0].SpendLimit.AmountOf("test-denom2")
				suite.Require().True(sdk.NewInt(100).Equal(remainder))
			},
		},
		{
			"no spend limit set for MsgTransfer port/channel",
			func() {
				msgTransfer.SourcePort = ibctesting.MockPort
				msgTransfer.SourceChannel = "channel-9"
			},
			func(res authz.AcceptResponse, err error) {
				suite.Require().Error(err)
			},
		},
		{
			"requested transfer amount is more than the spend limit",
			func() {
				msgTransfer.Token = sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1000))
			},
			func(res authz.AcceptResponse, err error) {
				suite.Require().Error(err)
			},
		},
		{
			"receiver address not permitted via allow list",
			func() {
				msgTransfer.Receiver = suite.chainB.SenderAccount.GetAddress().String()
			},
			func(res authz.AcceptResponse, err error) {
				suite.Require().Error(err)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path := NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

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
				Sender:        suite.chainA.SenderAccount.GetAddress().String(),
				Receiver:      ibctesting.TestAccAddress,
				TimeoutHeight: suite.chainB.GetTimeoutHeight(),
			}

			tc.malleate()

			res, err := transferAuthz.Accept(suite.chainA.GetContext(), &msgTransfer)
			tc.assertResult(res, err)
		})
	}
}

func (suite *TypesTestSuite) TestTransferAuthorizationMsgTypeURL() {
	var transferAuthz types.TransferAuthorization
	suite.Require().Equal(sdk.MsgTypeURL(&types.MsgTransfer{}), transferAuthz.MsgTypeURL(), "invalid type url for transfer authorization")
}

func (suite *TypesTestSuite) TestTransferAuthorizationValidateBasic() {
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
					SpendLimit:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
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
					SpendLimit:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
					AllowList:     []string{ibctesting.TestAccAddress},
				}

				transferAuthz.Allocations = append(transferAuthz.Allocations, allocation)
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			transferAuthz = types.TransferAuthorization{
				Allocations: []types.Allocation{
					{
						SourcePort:    mock.PortID,
						SourceChannel: ibctesting.FirstChannelID,
						SpendLimit:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
						AllowList:     []string{ibctesting.TestAccAddress},
					},
				},
			}

			tc.malleate()

			err := transferAuthz.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
