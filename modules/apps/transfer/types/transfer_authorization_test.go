package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/simapp"
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

func TestTransferAuthorization(t *testing.T) {
	app := simapp.Setup(t, false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	authorization := types.NewTransferAuthorization([]string{sourcePort}, []string{sourceChannel}, []sdk.Coins{coins1000}, [][]string{{toAddr.String()}})

	t.Log("verify authorization returns valid method name")
	require.Equal(t, authorization.MsgTypeURL(), "/ibc.applications.transfer.v1.MsgTransfer")
	require.NoError(t, authorization.ValidateBasic())
	transfer := types.NewMsgTransfer(sourcePort, sourceChannel, coin1000, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	require.NoError(t, authorization.ValidateBasic())

	t.Log("verify updated authorization returns nil")
	resp, err := authorization.Accept(ctx, transfer)
	require.NoError(t, err)
	require.True(t, resp.Delete)
	require.Nil(t, resp.Updated)

	t.Log("verify updated authorization returns remaining spent limit")
	authorization = types.NewTransferAuthorization([]string{sourcePort}, []string{sourceChannel}, []sdk.Coins{coins1000}, [][]string{{toAddr.String()}})
	require.Equal(t, authorization.MsgTypeURL(), "/ibc.applications.transfer.v1.MsgTransfer")
	require.NoError(t, authorization.ValidateBasic())
	transfer = types.NewMsgTransfer(sourcePort, sourceChannel, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	require.NoError(t, authorization.ValidateBasic())
	resp, err = authorization.Accept(ctx, transfer)
	require.NoError(t, err)
	require.False(t, resp.Delete)
	require.NotNil(t, resp.Updated)
	sendAuth := types.NewTransferAuthorization([]string{sourcePort}, []string{sourceChannel}, []sdk.Coins{coins500}, [][]string{{toAddr.String()}})
	require.Equal(t, sendAuth.String(), resp.Updated.String())

	t.Log("expect updated authorization nil after spending remaining amount")
	resp, err = resp.Updated.Accept(ctx, transfer)
	require.NoError(t, err)
	require.True(t, resp.Delete)
	require.Nil(t, resp.Updated)

	t.Log("expect error when spend limit for specific port and channel is not set")
	authorization = types.NewTransferAuthorization([]string{sourcePort}, []string{sourceChannel}, []sdk.Coins{coins1000}, [][]string{{toAddr.String()}})
	transfer = types.NewMsgTransfer(sourcePort2, sourceChannel2, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	_, err = authorization.Accept(ctx, transfer)
	require.Error(t, err)

	t.Log("expect removing only 1 allocation if spend limit is finalized for the port")
	authorization = types.NewTransferAuthorization(
		[]string{sourcePort, sourcePort2},
		[]string{sourceChannel, sourceChannel2},
		[]sdk.Coins{coins1000, coins1000},
		[][]string{{toAddr.String()}, {toAddr.String()}})
	transfer = types.NewMsgTransfer(sourcePort, sourceChannel, coin1000, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	resp, err = authorization.Accept(ctx, transfer)
	require.NoError(t, err)
	require.NotNil(t, resp.Updated)
	require.Equal(t, resp.Updated, types.NewTransferAuthorization([]string{sourcePort2}, []string{sourceChannel2}, []sdk.Coins{coins1000}, [][]string{{toAddr.String()}}))
	require.False(t, resp.Delete)

	t.Log("expect error when transferring to not allowed address")
	authorization = types.NewTransferAuthorization([]string{sourcePort}, []string{sourceChannel}, []sdk.Coins{coins1000}, [][]string{{fromAddr.String()}})
	transfer = types.NewMsgTransfer(sourcePort, sourceChannel, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	_, err = authorization.Accept(ctx, transfer)
	require.Error(t, err)
}

func TestTransferAuthorizationValidateBasic(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			transferAuthz = types.TransferAuthorization{
				Allocations: []types.Allocation{
					{
						SourcePort:    mock.PortID,
						SourceChannel: ibctesting.FirstChannelID,
						SpendLimit:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
						AllowList: []string{
							ibctesting.TestAccAddress,
						},
					},
				},
			}

			tc.malleate()

			err := transferAuthz.ValidateBasic()

			if tc.expPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
