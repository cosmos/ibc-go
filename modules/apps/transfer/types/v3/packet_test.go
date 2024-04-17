package v3

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

const (
	denom              = "transfer/gaiachannel/atom"
	amount             = "100"
	largeAmount        = "18446744073709551616"                                                           // one greater than largest uint64 (^uint64(0))
	invalidLargeAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
)

var (
	sender   = secp256k1.GenPrivKey().PubKey().Address().String()
	receiver = sdk.AccAddress("testaddr2").String()
)

// TestFungibleTokenPacketDataValidateBasic tests ValidateBasic for FungibleTokenPacketData
func TestFungibleTokenPacketDataValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		packetData FungibleTokenPacketData
		expErr     error
	}{
		{
			"success: valid packet",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"",
			),
			nil,
		},
		{
			"success: valid packet with memo",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"memo",
			),
			nil,
		},
		{
			"success: valid packet with large amount",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: largeAmount,
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"memo",
			),
			nil,
		},
		{
			"failure: invalid denom",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid empty amount",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid empty token array",
			NewFungibleTokenPacketData(
				[]*Token{},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid zero amount",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "0",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid negative amount",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "-100",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid large amount",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: invalidLargeAmount,
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"memo",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: missing sender address",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				"",
				receiver,
				"memo",
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: missing recipient address",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				"",
				"",
			),
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		err := tc.packetData.ValidateBasic()

		expPass := tc.expErr == nil
		if expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorContains(t, err, tc.expErr.Error(), tc.name)
		}
	}
}

func TestGetPacketSender(t *testing.T) {
	testCases := []struct {
		name       string
		packetData FungibleTokenPacketData
		expSender  string
	}{
		{
			"non-empty sender field",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"",
			),
			sender,
		},
		{
			"empty sender field",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				"",
				receiver,
				"abc",
			),
			"",
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expSender, tc.packetData.GetPacketSender(types.PortID))
	}

	packetData := NewFungibleTokenPacketData(
		[]*Token{
			{
				Denom:  "atom/pool",
				Amount: "1000",
				Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
			},
		},
		sender,
		receiver,
		"",
	)
	require.Equal(t, sender, packetData.GetPacketSender(types.PortID))
}

func TestPacketDataProvider(t *testing.T) {
	testCases := []struct {
		name          string
		packetData    FungibleTokenPacketData
		expCustomData interface{}
	}{
		{
			"success: src_callback key in memo",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, receiver)),

			map[string]interface{}{
				"address": receiver,
			},
		},
		{
			"success: src_callback key in memo with additional fields",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, receiver)),
			map[string]interface{}{
				"address":   receiver,
				"gas_limit": "200000",
			},
		},
		{
			"success: src_callback has string value",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				`{"src_callback": "string"}`),
			"string",
		},
		{
			"failure: src_callback key not found memo",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"dest_callback": {"address": "%s", "min_gas": "200000"}}`, receiver)),
			nil,
		},
		{
			"failure: empty memo",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				""),
			nil,
		},
		{
			"failure: non-json memo",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"invalid"),
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc

		customData := tc.packetData.GetCustomPacketData("src_callback")
		require.Equal(t, tc.expCustomData, customData)
	}
}

func TestFungibleTokenPacketDataOmitEmpty(t *testing.T) {
	testCases := []struct {
		name       string
		packetData FungibleTokenPacketData
		expMemo    bool
	}{
		{
			"empty memo field, resulting marshalled bytes should not contain the memo field",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"",
			),
			false,
		},
		{
			"non-empty memo field, resulting marshalled bytes should contain the memo field",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				},
				sender,
				receiver,
				"abc",
			),
			true,
		},
	}

	for _, tc := range testCases {
		bz, err := json.Marshal(tc.packetData)
		if tc.expMemo {
			require.NoError(t, err, tc.name)
			// check that the memo field is present in the marshalled bytes
			require.Contains(t, string(bz), "memo")
		} else {
			require.NoError(t, err, tc.name)
			// check that the memo field is not present in the marshalled bytes
			require.NotContains(t, string(bz), "memo")
		}
	}
}

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
						Denom:  "atom/pool",
						Amount: "1000",
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
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{},
					},
				},
				sender,
				receiver,
				"",
			),
			"atom/pool",
		},
		{
			"empty string trace",
			NewFungibleTokenPacketData(
				[]*Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{""},
					},
				},
				sender,
				receiver,
				"",
			),
			"atom/pool",
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
				Amount: "1000",
				Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
			},
			nil,
		},
		{
			"success: one port channel pair denom",
			Token{
				Denom:  "uatom",
				Amount: "1000",
				Trace:  []string{"transfer/channel-1"},
			},
			nil,
		},
		{
			"success: non transfer port trace",
			Token{
				Denom:  "uatom",
				Amount: "1000",
				Trace:  []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
			},
			nil,
		},
		{
			"failure: empty denom",
			Token{
				Denom:  "",
				Amount: "1000",
				Trace:  nil,
			},
			types.ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid identifier in trace",
			Token{
				Denom:  "uatom",
				Amount: "1000",
				Trace:  []string{"transfer/channel-1", "randomport"},
			},
			fmt.Errorf("trace info must come in pairs of port and channel identifiers '{portID}/{channelID}', got the identifiers: randomport"),
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
