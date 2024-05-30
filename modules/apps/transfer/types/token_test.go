package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	denom  = "atom/pool"
	amount = "100"
)

var (
	sender   = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver = sdk.AccAddress("testaddr2").String()
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
			fmt.Errorf("trace info must come in pairs of port and channel identifiers '{portID}/{channelID}', got the identifiers: [transfer channel-1 randomport]"),
		},
		{
			"failure: empty identifier in trace",
			Token{
				Denom: Denom{
					Base:  "uatom",
					Trace: []Trace{Trace{}},
				},
				Amount: amount,
			},
			fmt.Errorf("trace info must come in pairs of port and channel identifiers '{portID}/{channelID}', got the identifiers: "),
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

func TestTokens_String(t *testing.T) {
	cases := []struct {
		name     string
		input    Tokens
		expected string
	}{
		{
			"empty tokens",
			Tokens{},
			"",
		},
		{
			"single token, no trace",
			Tokens{
				Token{
					Denom: Denom{
						Base:  "tree",
						Trace: []Trace{},
					},
					Amount: "1",
				},
			},
			`denom:<base:"tree" > amount:"1" `,
		},
		{
			"single token with trace",
			Tokens{
				Token{
					Denom: Denom{
						Base: "tree",
						Trace: []Trace{
							NewTrace("portid", "channelid"),
						},
					},
					Amount: "1",
				},
			},
			`denom:<base:"tree" trace:"portid/channelid" > amount:"1" `,
		},
		{
			"multiple tokens, no trace",
			Tokens{
				Token{
					Denom: Denom{
						Base: "tree",
					},
					Amount: "1",
				},
				Token{
					Denom: Denom{
						Base: "gas",
					},
					Amount: "2",
				},
				Token{
					Denom: Denom{
						Base: "mineral",
					},
					Amount: "3",
				},
			},
			`denom:<base:"tree" > amount:"1" ,denom:<base:"gas" > amount:"2" ,denom:<base:"mineral" > amount:"3" `,
		},
		{
			"multiple tokens, trace and no trace",
			Tokens{
				Token{
					Denom: Denom{
						Base: "tree",
					},
					Amount: "1",
				},
				Token{
					Denom: Denom{
						Base: "gas",
						Trace: []Trace{
							NewTrace("portid", "channelid"),
						},
					},
					Amount: "2",
				},
				Token{
					Denom: Denom{
						Base: "mineral",
						Trace: []Trace{
							NewTrace("portid", "channelid"),
							NewTrace("transfer", "channel-52"),
						},
					},
					Amount: "3",
				},
			},
			`denom:<base:"tree" > amount:"1" ,denom:<base:"gas" trace:"portid/channelid" > amount:"2" ,denom:<base:"mineral" trace:"portid/channelid" trace:"transfer/channel-52" > amount:"3" `,
		},
	}

	for _, tt := range cases {
		require.Equal(t, tt.expected, tt.input.String())
	}
}
