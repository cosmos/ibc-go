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

func TestGetFullDenomPath(t *testing.T) {
	testCases := []struct {
		name       string
		packetData FungibleTokenPacketDataV2
		expPath    string
	}{
		{
			"denom path with trace",
			NewFungibleTokenPacketDataV2(
				[]Token{
					{
						Denom:  denom,
						Amount: amount,
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"",
			),
			"transfer/channel-0/transfer/channel-1/atom/pool",
		},
		{
			"nil trace",
			NewFungibleTokenPacketDataV2(
				[]Token{
					{
						Denom:  denom,
						Amount: amount,
						Trace:  []string{},
					},
				},
				sender,
				receiver,
				"",
			),
			denom,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.packetData.Tokens[0].GetFullDenomPath()
			require.Equal(t, tc.expPath, path)
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name     string
		token    Token
		expError error
	}{
		{
			"success: multiple port channel pair denom",
			Token{
				Denom:  "atom",
				Amount: amount,
				Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
			},
			nil,
		},
		{
			"success: one port channel pair denom",
			Token{
				Denom:  "uatom",
				Amount: amount,
				Trace:  []string{"transfer/channel-1"},
			},
			nil,
		},
		{
			"success: non transfer port trace",
			Token{
				Denom:  "uatom",
				Amount: amount,
				Trace:  []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
			},
			nil,
		},
		{
			"failure: empty denom",
			Token{
				Denom:  "",
				Amount: amount,
				Trace:  nil,
			},
			ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid amount string",
			Token{
				Denom:  "atom",
				Amount: "value",
				Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
			},
			ErrInvalidAmount,
		},
		{
			"failure: amount is zero",
			Token{
				Denom:  "atom",
				Amount: "0",
				Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
			},
			ErrInvalidAmount,
		},
		{
			"failure: amount is negative",
			Token{
				Denom:  "atom",
				Amount: "-1",
				Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
			},
			ErrInvalidAmount,
		},
		{
			"failure: invalid identifier in trace",
			Token{
				Denom:  "uatom",
				Amount: amount,
				Trace:  []string{"transfer/channel-1", "randomport"},
			},
			fmt.Errorf("trace info must come in pairs of port and channel identifiers '{portID}/{channelID}', got the identifiers: [transfer channel-1 randomport]"),
		},
		{
			"failure: empty identifier in trace",
			Token{
				Denom:  "uatom",
				Amount: amount,
				Trace:  []string{""},
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
					Denom:  "tree",
					Amount: "1",
					Trace:  []string{},
				},
			},
			`denom:"tree" amount:"1" `,
		},
		{
			"single token with trace",
			Tokens{
				Token{
					Denom:  "tree",
					Amount: "1",
					Trace:  []string{"portid/channelid"},
				},
			},
			`denom:"tree" amount:"1" trace:"portid/channelid" `,
		},
		{
			"multiple tokens, no trace",
			Tokens{
				Token{
					Denom:  "tree",
					Amount: "1",
					Trace:  []string{},
				},
				Token{
					Denom:  "gas",
					Amount: "2",
					Trace:  []string{},
				},
				Token{
					Denom:  "mineral",
					Amount: "3",
					Trace:  []string{},
				},
			},
			`denom:"tree" amount:"1" ,denom:"gas" amount:"2" ,denom:"mineral" amount:"3" `,
		},
		{
			"multiple tokens, trace and no trace",
			Tokens{
				Token{
					Denom:  "tree",
					Amount: "1",
					Trace:  []string{},
				},
				Token{
					Denom:  "gas",
					Amount: "2",
					Trace:  []string{"portid/channelid"},
				},
				Token{
					Denom:  "mineral",
					Amount: "3",
					Trace:  []string{"portid/channelid", "transfer/channel-52"},
				},
			},
			`denom:"tree" amount:"1" ,denom:"gas" amount:"2" trace:"portid/channelid" ,denom:"mineral" amount:"3" trace:"portid/channelid" trace:"transfer/channel-52" `,
		},
	}

	for _, tt := range cases {
		require.Equal(t, tt.expected, tt.input.String())
	}
}
