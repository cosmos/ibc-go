package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
)

func TestTransferAuthorization(t *testing.T) {
	app := simapp.Setup(t, false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	allocation := Allocation{
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		SpendLimit:    coins1000,
		AllowList:     []string{toAddr.String()},
	}
	authorization := NewTransferAuthorization(allocation)

	t.Log("verify authorization returns valid method name")
	require.Equal(t, authorization.MsgTypeURL(), "/ibc.applications.transfer.v1.MsgTransfer")
	require.NoError(t, authorization.ValidateBasic())
	transfer := NewMsgTransfer(sourcePort, sourceChannel, coin1000, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	require.NoError(t, authorization.ValidateBasic())

	t.Log("verify updated authorization returns nil")
	resp, err := authorization.Accept(ctx, transfer)
	require.NoError(t, err)
	require.True(t, resp.Delete)
	require.Nil(t, resp.Updated)

	t.Log("verify updated authorization returns remaining spent limit")
	authorization = NewTransferAuthorization(allocation)
	require.Equal(t, authorization.MsgTypeURL(), "/ibc.applications.transfer.v1.MsgTransfer")
	require.NoError(t, authorization.ValidateBasic())
	transfer = NewMsgTransfer(sourcePort, sourceChannel, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	require.NoError(t, authorization.ValidateBasic())
	resp, err = authorization.Accept(ctx, transfer)
	require.NoError(t, err)
	require.False(t, resp.Delete)
	require.NotNil(t, resp.Updated)

	allocation = Allocation{
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		SpendLimit:    coins500,
		AllowList:     []string{toAddr.String()},
	}
	sendAuth := NewTransferAuthorization(allocation)
	require.Equal(t, sendAuth.String(), resp.Updated.String())

	t.Log("expect updated authorization nil after spending remaining amount")
	resp, err = resp.Updated.Accept(ctx, transfer)
	require.NoError(t, err)
	require.True(t, resp.Delete)
	require.Nil(t, resp.Updated)

	t.Log("expect error when spend limit for specific port and channel is not set")
	allocation = Allocation{
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		SpendLimit:    coins1000,
		AllowList:     []string{toAddr.String()},
	}
	authorization = NewTransferAuthorization(allocation)
	transfer = NewMsgTransfer(sourcePort2, sourceChannel2, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	_, err = authorization.Accept(ctx, transfer)
	require.Error(t, err)

	t.Log("expect removing only 1 allocation if spend limit is finalized for the port")

	allocations := []Allocation{
		{
			SourcePort:    sourcePort,
			SourceChannel: sourceChannel,
			SpendLimit:    coins1000,
			AllowList:     []string{toAddr.String()},
		},
		{
			SourcePort:    sourcePort2,
			SourceChannel: sourceChannel2,
			SpendLimit:    coins1000,
			AllowList:     []string{toAddr.String()},
		},
	}
	authorization = NewTransferAuthorization(allocations...)
	transfer = NewMsgTransfer(sourcePort, sourceChannel, coin1000, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	resp, err = authorization.Accept(ctx, transfer)
	require.NoError(t, err)
	require.NotNil(t, resp.Updated)
	require.Equal(t, resp.Updated, NewTransferAuthorization(allocations[1]))
	require.False(t, resp.Delete)

	t.Log("expect error when transferring to not allowed address")
	allocation = Allocation{
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		SpendLimit:    coins1000,
		AllowList:     []string{fromAddr.String()},
	}
	authorization = NewTransferAuthorization(allocation)
	transfer = NewMsgTransfer(sourcePort, sourceChannel, coin500, fromAddr.String(), toAddr.String(), timeoutHeight, 0, "")
	_, err = authorization.Accept(ctx, transfer)
	require.Error(t, err)
}
