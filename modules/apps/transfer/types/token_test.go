package types

import (
	fmt "fmt"
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
				[]*Token{
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
				[]*Token{
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
		{
			"empty string trace",
			NewFungibleTokenPacketDataV2(
				[]*Token{
					{
						Denom:  denom,
						Amount: amount,
						Trace:  []string{""},
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
			"failure: invalid identifier in trace",
			Token{
				Denom:  "uatom",
				Amount: amount,
				Trace:  []string{"transfer/channel-1", "randomport"},
			},
			fmt.Errorf("trace info must come in pairs of port and channel identifiers '{portID}/{channelID}', got the identifiers: [transfer channel-1 randomport]"),
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
