package v3

import (
	fmt "fmt"
	"testing"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"
)

func TestGetFullDenomPath(t *testing.T) {
	testCases := []struct {
		name       string
		packetData FungibleTokenPacketData
		expPath    string
	}{
		{
			"denom path with trace",
			NewFungibleTokenPacketData(
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
			NewFungibleTokenPacketData(
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
			NewFungibleTokenPacketData(
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
		path := tc.packetData.Tokens[0].GetFullDenomPath()

		require.Equal(t, tc.expPath, path)
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
			types.ErrInvalidDenomForTransfer,
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
		err := tc.token.Validate()
		expPass := tc.expError == nil
		if expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorContains(t, err, tc.expError.Error(), tc.name)
		}
	}
}
