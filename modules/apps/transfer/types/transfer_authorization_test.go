package types_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/mock"
)

const (
	testMemo1 = `{"wasm":{"contract":"osmo1c3ljch9dfw5kf52nfwpxd2zmj2ese7agnx0p9tenkrryasrle5sqf3ftpg","msg":{"osmosis_swap":{"output_denom":"uosmo","slippage":{"twap":{"slippage_percentage":"20","window_seconds":10}},"receiver":"feeabs/feeabs1efd63aw40lxf3n4mhf7dzhjkr453axurwrhrrw","on_failed_delivery":"do_nothing"}}}}`
	testMemo2 = `{"forward":{"channel":"channel-11","port":"transfer","receiver":"stars1twfv52yxcyykx2lcvgl42svw46hsm5dd4ww6xy","retries":2,"timeout":1712146014542131200}}`
)

func (s *TypesTestSuite) TestTransferAuthorizationAccept() {
	var (
		msgTransfer   *types.MsgTransfer
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

				isEqual := updatedAuthz.Allocations[0].SpendLimit.Equal(sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(50))))
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
			"success: empty AllowedPacketData and empty memo",
			func() {
				allowedList := []string{}
				transferAuthz.Allocations[0].AllowedPacketData = allowedList
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				s.Require().True(res.Accept)
				s.Require().True(res.Delete)
				s.Require().Nil(res.Updated)
			},
		},
		{
			"success: AllowedPacketData allows any packet",
			func() {
				allowedList := []string{"*"}
				transferAuthz.Allocations[0].AllowedPacketData = allowedList
				msgTransfer.Memo = testMemo1
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				s.Require().True(res.Accept)
				s.Require().True(res.Delete)
				s.Require().Nil(res.Updated)
			},
		},
		{
			"success: transfer memo allowed",
			func() {
				allowedList := []string{testMemo1, testMemo2}
				transferAuthz.Allocations[0].AllowedPacketData = allowedList
				msgTransfer.Memo = testMemo1
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().NoError(err)

				s.Require().True(res.Accept)
				s.Require().True(res.Delete)
				s.Require().Nil(res.Updated)
			},
		},
		{
			"empty AllowedPacketData but not empty memo",
			func() {
				allowedList := []string{}
				transferAuthz.Allocations[0].AllowedPacketData = allowedList
				msgTransfer.Memo = testMemo1
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().Error(err)
			},
		},
		{
			"memo not allowed",
			func() {
				allowedList := []string{testMemo1}
				transferAuthz.Allocations[0].AllowedPacketData = allowedList
				msgTransfer.Memo = testMemo2
			},
			func(res authz.AcceptResponse, err error) {
				s.Require().Error(err)
				s.Require().ErrorContains(err, fmt.Sprintf("not allowed memo: %s", testMemo2))
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

			path := ibctesting.NewTransferPath(s.chainA, s.chainB)
			path.Setup()

			transferAuthz = types.TransferAuthorization{
				Allocations: []types.Allocation{
					{
						SourcePort:    path.EndpointA.ChannelConfig.PortID,
						SourceChannel: path.EndpointA.ChannelID,
						SpendLimit:    sdk.NewCoins(ibctesting.TestCoin),
						AllowList:     []string{ibctesting.TestAccAddress},
					},
				},
			}

			msgTransfer = types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				ibctesting.TestCoin,
				s.chainA.SenderAccount.GetAddress().String(),
				ibctesting.TestAccAddress,
				s.chainB.GetTimeoutHeight(),
				0,
				"",
			)

			tc.malleate()

			res, err := transferAuthz.Accept(s.chainA.GetContext(), msgTransfer)
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
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"success: empty allow list",
			func() {
				transferAuthz.Allocations[0].AllowList = []string{}
			},
			nil,
		},
		{
			"success: with multiple allocations",
			func() {
				allocation := types.Allocation{
					SourcePort:    types.PortID,
					SourceChannel: "channel-1",
					SpendLimit:    sdk.NewCoins(ibctesting.TestCoin),
					AllowList:     []string{},
				}

				transferAuthz.Allocations = append(transferAuthz.Allocations, allocation)
			},
			nil,
		},
		{
			"success: with unlimited spend limit of max uint256",
			func() {
				transferAuthz.Allocations[0].SpendLimit = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, types.UnboundedSpendLimit()))
			},
			nil,
		},
		{
			"success: wildcard allowed packet data",
			func() {
				transferAuthz.Allocations[0].AllowedPacketData = []string{"*"}
			},
			nil,
		},
		{
			"empty allocations",
			func() {
				transferAuthz = types.TransferAuthorization{Allocations: []types.Allocation{}}
			},
			types.ErrInvalidAuthorization,
		},
		{
			"nil allocations",
			func() {
				transferAuthz = types.TransferAuthorization{}
			},
			types.ErrInvalidAuthorization,
		},
		{
			"nil spend limit coins",
			func() {
				transferAuthz.Allocations[0].SpendLimit = nil
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"invalid spend limit coins",
			func() {
				transferAuthz.Allocations[0].SpendLimit = sdk.Coins{sdk.Coin{Denom: ""}}
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"duplicate entry in allow list",
			func() {
				transferAuthz.Allocations[0].AllowList = []string{ibctesting.TestAccAddress, ibctesting.TestAccAddress}
			},
			types.ErrInvalidAuthorization,
		},
		{
			"invalid port identifier",
			func() {
				transferAuthz.Allocations[0].SourcePort = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid channel identifier",
			func() {
				transferAuthz.Allocations[0].SourceChannel = ""
			},
			host.ErrInvalidID,
		},
		{
			"duplicate channel ID",
			func() {
				allocation := types.Allocation{
					SourcePort:    mock.PortID,
					SourceChannel: transferAuthz.Allocations[0].SourceChannel,
					SpendLimit:    sdk.NewCoins(ibctesting.TestCoin),
					AllowList:     []string{ibctesting.TestAccAddress},
				}

				transferAuthz.Allocations = append(transferAuthz.Allocations, allocation)
			},
			channeltypes.ErrInvalidChannel,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			transferAuthz = types.TransferAuthorization{
				Allocations: []types.Allocation{
					{
						SourcePort:    mock.PortID,
						SourceChannel: ibctesting.FirstChannelID,
						SpendLimit:    sdk.NewCoins(ibctesting.TestCoin),
						AllowList:     []string{ibctesting.TestAccAddress},
					},
				},
			}

			tc.malleate()

			err := transferAuthz.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
