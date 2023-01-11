package types_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	"github.com/cosmos/ibc-go/v6/testing/mock"
)

var (
	sourcePort     = "port"
	sourceChannel  = "channel-100"
	sourcePort2    = "port2"
	sourceChannel2 = "channel-101"
	coins1000      = sdk.Coins{sdk.NewCoin("stake", sdk.NewInt(1000))}
	coins500       = sdk.Coins{sdk.NewCoin("stake", sdk.NewInt(500))}
	coin1000       = sdk.NewCoin("stake", sdk.NewInt(1000))
	coin500        = sdk.NewCoin("stake", sdk.NewInt(500))
	fromAddr       = sdk.AccAddress("_____from _____")
	toAddr         = sdk.AccAddress("_______to________")
	timeoutHeight  = clienttypes.NewHeight(0, 10)
)

func (suite *TypesTestSuite) TestTransferAuthorizationAccept() {
	var (
		authorization types.TransferAuthorization
		msgTransfer   types.MsgTransfer
	)

	testCases := []struct {
		name      string
		malleate  func()
		expPass   bool
		expResult func()
	}{
		{
			"success",
			func() {},
			true,
			func() {},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path := NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			authorization = types.TransferAuthorization{
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

			_, err := authorization.Accept(suite.chainA.GetContext(), &msgTransfer)
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}

	// app := simapp.Setup(t, false)
	// ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	// allocation := types.Allocation{
	// 	SourcePort:    sourcePort,
	// 	SourceChannel: sourceChannel,
	// 	SpendLimit:    coins1000,
	// 	AllowList:     []string{toAddr.String()},
	// }
	// authorization := types.NewTransferAuthorization(allocation)

	// t.Log("verify authorization returns valid method name")
	// require.Equal(t, authorization.MsgTypeURL(), "/ibc.applications.transfer.v1.MsgTransfer")
	// require.NoError(t, authorization.ValidateBasic())
	// transfer := types.NewMsgTransfer(sourcePort, sourceChannel, coin1000, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	// require.NoError(t, authorization.ValidateBasic())

	// t.Log("verify updated authorization returns nil")
	// resp, err := authorization.Accept(ctx, transfer)
	// require.NoError(t, err)
	// require.True(t, resp.Delete)
	// require.Nil(t, resp.Updated)

	// t.Log("verify updated authorization returns remaining spent limit")
	// authorization = types.NewTransferAuthorization(allocation)
	// require.Equal(t, authorization.MsgTypeURL(), "/ibc.applications.transfer.v1.MsgTransfer")
	// require.NoError(t, authorization.ValidateBasic())
	// transfer = types.NewMsgTransfer(sourcePort, sourceChannel, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	// require.NoError(t, authorization.ValidateBasic())
	// resp, err = authorization.Accept(ctx, transfer)
	// require.NoError(t, err)
	// require.False(t, resp.Delete)
	// require.NotNil(t, resp.Updated)

	// allocation = types.Allocation{
	// 	SourcePort:    sourcePort,
	// 	SourceChannel: sourceChannel,
	// 	SpendLimit:    coins500,
	// 	AllowList:     []string{toAddr.String()},
	// }
	// sendAuth := types.NewTransferAuthorization(allocation)
	// require.Equal(t, sendAuth.String(), resp.Updated.String())

	// t.Log("expect updated authorization nil after spending remaining amount")
	// resp, err = resp.Updated.Accept(ctx, transfer)
	// require.NoError(t, err)
	// require.True(t, resp.Delete)
	// require.Nil(t, resp.Updated)

	// t.Log("expect error when spend limit for specific port and channel is not set")
	// allocation = types.Allocation{
	// 	SourcePort:    sourcePort,
	// 	SourceChannel: sourceChannel,
	// 	SpendLimit:    coins1000,
	// 	AllowList:     []string{toAddr.String()},
	// }
	// authorization = types.NewTransferAuthorization(allocation)
	// transfer = types.NewMsgTransfer(sourcePort2, sourceChannel2, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	// _, err = authorization.Accept(ctx, transfer)
	// require.Error(t, err)

	// t.Log("expect removing only 1 allocation if spend limit is finalized for the port")

	// allocations := []types.Allocation{
	// 	{
	// 		SourcePort:    sourcePort,
	// 		SourceChannel: sourceChannel,
	// 		SpendLimit:    coins1000,
	// 		AllowList:     []string{toAddr.String()},
	// 	},
	// 	{
	// 		SourcePort:    sourcePort2,
	// 		SourceChannel: sourceChannel2,
	// 		SpendLimit:    coins1000,
	// 		AllowList:     []string{toAddr.String()},
	// 	},
	// }
	// authorization = types.NewTransferAuthorization(allocations...)
	// transfer = types.NewMsgTransfer(sourcePort, sourceChannel, coin1000, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	// resp, err = authorization.Accept(ctx, transfer)
	// require.NoError(t, err)
	// require.NotNil(t, resp.Updated)
	// require.Equal(t, resp.Updated, types.NewTransferAuthorization(allocations[1]))
	// require.False(t, resp.Delete)

	// t.Log("expect error when transferring to not allowed address")
	// allocation = types.Allocation{
	// 	SourcePort:    sourcePort,
	// 	SourceChannel: sourceChannel,
	// 	SpendLimit:    coins1000,
	// 	AllowList:     []string{fromAddr.String()},
	// }
	// authorization = types.NewTransferAuthorization(allocation)
	// transfer = types.NewMsgTransfer(sourcePort, sourceChannel, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	// _, err = authorization.Accept(ctx, transfer)
	// require.Error(t, err)
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
					SourceChannel: ibctesting.FirstChannelID,
					SpendLimit:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
					AllowList:     []string{},
				}

				transferAuthz.Allocations = append(transferAuthz.Allocations, allocation)
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
