package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	denom  = "atom/pool"
	amount = "100"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name     string
		token    Token
		expError error
	}{
		{
			"success: multiple port channel pair denom",
			Token{
				Denom: Denom{
					Base: "atom",
					Trace: []Hop{
						NewHop("transfer", "channel-0"),
						NewHop("transfer", "channel-1"),
					},
				},
				Amount: amount,
			},
			nil,
		},
		{
			"success: one port channel pair denom",
			Token{
				Denom: Denom{
					Base: "uatom",
					Trace: []Hop{
						NewHop("transfer", "channel-1"),
					},
				},
				Amount: amount,
			},
			nil,
		},
		{
			"success: non transfer port trace",
			Token{
				Denom: Denom{
					Base: "uatom",
					Trace: []Hop{
						NewHop("transfer", "channel-0"),
						NewHop("transfer", "channel-1"),
						NewHop("transfer-custom", "channel-2"),
					},
				},
				Amount: amount,
			},
			nil,
		},
		{
			"failure: empty denom",
			Token{
				Denom:  Denom{},
				Amount: amount,
			},
			ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid amount string",
			Token{
				Denom: Denom{
					Base: "atom",
					Trace: []Hop{
						NewHop("transfer", "channel-0"),
						NewHop("transfer", "channel-1"),
					},
				},
				Amount: "value",
			},
			ErrInvalidAmount,
		},
		{
			"failure: amount is zero",
			Token{
				Denom: Denom{
					Base: "atom",
					Trace: []Hop{
						NewHop("transfer", "channel-0"),
						NewHop("transfer", "channel-1"),
					},
				},
				Amount: "0",
			},
			ErrInvalidAmount,
		},
		{
			"failure: amount is negative",
			Token{
				Denom: Denom{
					Base: "atom",
					Trace: []Hop{
						NewHop("transfer", "channel-0"),
						NewHop("transfer", "channel-1"),
					},
				},
				Amount: "-1",
			},
			ErrInvalidAmount,
		},
		{
			"failure: invalid identifier in trace",
			Token{
				Denom: Denom{
					Base: "uatom",
					Trace: []Hop{
						NewHop("transfer", "channel-1"),
						NewHop("randomport", ""),
					},
				},
				Amount: amount,
			},
			errors.New("invalid token denom: invalid trace: invalid hop source channel ID : identifier cannot be blank: invalid identifier"),
		},
		{
			"failure: empty identifier in trace",
			Token{
				Denom: Denom{
					Base:  "uatom",
					Trace: []Hop{{}},
				},
				Amount: amount,
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
		token    Token
		expCoin  sdk.Coin
		expError error
	}{
		{
			"success: convert token to coin",
			Token{
				Denom: Denom{
					Base:  denom,
					Trace: []Hop{},
				},
				Amount: amount,
			},
			sdk.NewCoin(denom, sdkmath.NewInt(100)),
			nil,
		},
		{
			"failure: invalid amount string",
			Token{
				Denom: Denom{
					Base:  denom,
					Trace: []Hop{},
				},
				Amount: "value",
			},
			sdk.Coin{},
			ErrInvalidAmount,
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
