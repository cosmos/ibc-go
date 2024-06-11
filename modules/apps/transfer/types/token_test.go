package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
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
					Trace: []Trace{
						NewTrace("transfer", "channel-0"),
						NewTrace("transfer", "channel-1"),
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
					Trace: []Trace{
						NewTrace("transfer", "channel-1"),
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
					Trace: []Trace{
						NewTrace("transfer", "channel-0"),
						NewTrace("transfer", "channel-1"),
						NewTrace("transfer-custom", "channel-2"),
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
					Trace: []Trace{
						NewTrace("transfer", "channel-0"),
						NewTrace("transfer", "channel-1"),
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
					Trace: []Trace{
						NewTrace("transfer", "channel-0"),
						NewTrace("transfer", "channel-1"),
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
					Trace: []Trace{
						NewTrace("transfer", "channel-0"),
						NewTrace("transfer", "channel-1"),
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
					Trace: []Trace{
						NewTrace("transfer", "channel-1"),
						NewTrace("randomport", ""),
					},
				},
				Amount: amount,
			},
			fmt.Errorf("invalid token denom: invalid trace: invalid channelID: identifier cannot be blank: invalid identifier"),
		},
		{
			"failure: empty identifier in trace",
			Token{
				Denom: Denom{
					Base:  "uatom",
					Trace: []Trace{{}},
				},
				Amount: amount,
			},
			fmt.Errorf("invalid token denom: invalid trace: invalid portID: identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.token.Validate()
			expPass := tc.expError == nil
			if expPass {
				require.NoError(t, err, tc.name)
			} else {
				require.ErrorContains(t, err, tc.expError.Error(), tc.name)
			}
		})
	}
}
