package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

const (
	tokenDenom  = "atom/pool"
	tokenAmount = "100"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name     string
		token    types.Token
		expError error
	}{
		{
			"success: multiple port channel pair denom",
			types.Token{
				Denom: types.Denom{
					Base: "atom",
					Trace: []types.Hop{
						types.NewHop("transfer", "channel-0"),
						types.NewHop("transfer", "channel-1"),
					},
				},
				Amount: tokenAmount,
			},
			nil,
		},
		{
			"success: one port channel pair denom",
			types.Token{
				Denom: types.Denom{
					Base: "uatom",
					Trace: []types.Hop{
						types.NewHop("transfer", "channel-1"),
					},
				},
				Amount: tokenAmount,
			},
			nil,
		},
		{
			"success: non transfer port trace",
			types.Token{
				Denom: types.Denom{
					Base: "uatom",
					Trace: []types.Hop{
						types.NewHop("transfer", "channel-0"),
						types.NewHop("transfer", "channel-1"),
						types.NewHop("transfer-custom", "channel-2"),
					},
				},
				Amount: tokenAmount,
			},
			nil,
		},
		{
			"failure: empty denom",
			types.Token{
				Denom:  types.Denom{},
				Amount: tokenAmount,
			},
			types.ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid amount string",
			types.Token{
				Denom: types.Denom{
					Base: "atom",
					Trace: []types.Hop{
						types.NewHop("transfer", "channel-0"),
						types.NewHop("transfer", "channel-1"),
					},
				},
				Amount: "value",
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: amount is zero",
			types.Token{
				Denom: types.Denom{
					Base: "atom",
					Trace: []types.Hop{
						types.NewHop("transfer", "channel-0"),
						types.NewHop("transfer", "channel-1"),
					},
				},
				Amount: "0",
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: amount is negative",
			types.Token{
				Denom: types.Denom{
					Base: "atom",
					Trace: []types.Hop{
						types.NewHop("transfer", "channel-0"),
						types.NewHop("transfer", "channel-1"),
					},
				},
				Amount: "-1",
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid identifier in trace",
			types.Token{
				Denom: types.Denom{
					Base: "uatom",
					Trace: []types.Hop{
						types.NewHop("transfer", "channel-1"),
						types.NewHop("randomport", ""),
					},
				},
				Amount: tokenAmount,
			},
			errors.New("invalid token denom: invalid trace: invalid hop source channel ID : identifier cannot be blank: invalid identifier"),
		},
		{
			"failure: empty identifier in trace",
			types.Token{
				Denom: types.Denom{
					Base:  "uatom",
					Trace: []types.Hop{{}},
				},
				Amount: tokenAmount,
			},
			errors.New("invalid token denom: invalid trace: invalid hop source port ID : identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.token.Validate()
			if tc.expError == nil {
				require.NoError(t, err, tc.name)
			} else {
				require.ErrorContains(t, err, tc.expError.Error(), tc.name)
			}
		})
	}
}

func TestToCoin(t *testing.T) {
	testCases := []struct {
		name     string
		token    types.Token
		expCoin  sdk.Coin
		expError error
	}{
		{
			"success: convert token to coin",
			types.Token{
				Denom: types.Denom{
					Base:  tokenDenom,
					Trace: []types.Hop{},
				},
				Amount: tokenAmount,
			},
			sdk.NewCoin(tokenDenom, sdkmath.NewInt(100)),
			nil,
		},
		{
			"failure: invalid amount string",
			types.Token{
				Denom: types.Denom{
					Base:  tokenDenom,
					Trace: []types.Hop{},
				},
				Amount: "value",
			},
			sdk.Coin{},
			types.ErrInvalidAmount,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			coin, err := tc.token.ToCoin()

			require.Equal(t, tc.expCoin, coin, tc.name)

			if tc.expError == nil {
				require.NoError(t, err, tc.name)
			} else {
				require.ErrorContains(t, err, tc.expError.Error(), tc.name)
			}
		})
	}
}
