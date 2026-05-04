package types_test

import (
	"testing"

	"github.com/cosmos/sandbox-ledger/testutil"
	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	testAddr = "cosmos1y6xz2ggfc0pcsmyjlekh0j9pxh6hk87yfwcjct"
)

func TestMsgCreateDenom_ValidateBasic(t *testing.T) {
	testutil.SafeSetAddressPrefixes()
	tests := []struct {
		name      string
		msg       types.MsgCreateDenom
		expectErr error
	}{
		{
			name: "valid message",
			msg: types.MsgCreateDenom{
				Sender: testAddr,
				Denom:  testDenom,
			},
			expectErr: nil,
		},
		{
			name: "invalid sender",
			msg: types.MsgCreateDenom{
				Sender: "invalid",
				Denom:  testDenom,
			},
			expectErr: types.ErrInvalidCreator,
		},
		{
			name: "empty sender",
			msg: types.MsgCreateDenom{
				Sender: "",
				Denom:  testDenom,
			},
			expectErr: types.ErrInvalidCreator,
		},
		{
			name: "empty denom",
			msg: types.MsgCreateDenom{
				Sender: testAddr,
				Denom:  "",
			},
			expectErr: types.ErrInvalidDenom,
		},
		{
			name: "invalid denom",
			msg: types.MsgCreateDenom{
				Sender: testAddr,
				Denom:  "not-so-alphanumeric",
			},
			expectErr: types.ErrInvalidDenom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			require.ErrorIs(t, err, tt.expectErr, "got: %v want: %v", err, tt.expectErr)
		})
	}
}

func TestMsgMint_ValidateBasic(t *testing.T) {
	testutil.SafeSetAddressPrefixes()
	tests := []struct {
		name      string
		msg       types.MsgMint
		expectErr error
	}{
		{
			name: "valid message",
			msg: types.MsgMint{
				From:    testAddr,
				Address: testAddr,
				Amount:  sdk.NewCoin(testDenom, math.NewInt(1000000)),
			},
			expectErr: nil,
		},
		{
			name: "invalid sender",
			msg: types.MsgMint{
				From:    "invalid",
				Address: testAddr,
				Amount:  sdk.NewCoin(testDenom, math.NewInt(1000000)),
			},
			expectErr: types.ErrInvalidAddress,
		},
		{
			name: "invalid mint to address",
			msg: types.MsgMint{
				From:    testAddr,
				Address: "invalid",
				Amount:  sdk.NewCoin(testDenom, math.NewInt(1000000)),
			},
			expectErr: types.ErrInvalidAddress,
		},
		{
			name: "zero amount",
			msg: types.MsgMint{
				From:    testAddr,
				Address: testAddr,
				Amount:  sdk.NewCoin(testDenom, math.ZeroInt()),
			},
			expectErr: types.ErrInvalidAmount,
		},
		{
			name: "negative amount",
			msg: types.MsgMint{
				From:    testAddr,
				Address: testAddr,
				Amount:  sdk.Coin{Denom: testDenom, Amount: math.NewInt(-1000)},
			},
			expectErr: types.ErrInvalidAmount,
		},
		{
			name:      "invalid denom",
			msg:       types.MsgMint{From: testAddr, Address: testAddr, Amount: sdk.NewCoin("not-so-alphanumeric", math.NewInt(1))},
			expectErr: types.ErrInvalidDenom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			require.ErrorIs(t, err, tt.expectErr)
		})
	}
}

func TestMsgBurn_ValidateBasic(t *testing.T) {
	testutil.SafeSetAddressPrefixes()
	tests := []struct {
		name      string
		msg       types.MsgBurn
		expectErr error
	}{
		{
			name: "valid message",
			msg: types.MsgBurn{
				From:   testAddr,
				Amount: sdk.NewCoin(testDenom, math.NewInt(1000000)),
			},
			expectErr: nil,
		},
		{
			name: "invalid sender",
			msg: types.MsgBurn{
				From:   "invalid",
				Amount: sdk.NewCoin(testDenom, math.NewInt(1000000)),
			},
			expectErr: types.ErrInvalidAddress,
		},
		{
			name: "zero amount",
			msg: types.MsgBurn{
				From:   testAddr,
				Amount: sdk.NewCoin(testDenom, math.ZeroInt()),
			},
			expectErr: types.ErrInvalidAmount,
		},
		{
			name: "negative amount",
			msg: types.MsgBurn{
				From:   testAddr,
				Amount: sdk.Coin{Denom: testDenom, Amount: math.NewInt(-1000)},
			},
			expectErr: types.ErrInvalidAmount,
		},
		{
			name:      "invalid denom",
			msg:       types.MsgBurn{From: testAddr, Amount: sdk.NewCoin("not-so-alphanumeric", math.NewInt(1))},
			expectErr: types.ErrInvalidDenom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			require.ErrorIs(t, err, tt.expectErr)
		})
	}
}
