package v3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/gogoproto/jsonpb"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

const (
	denom              = "atom/pool"
	amount             = "1000"
	largeAmount        = "18446744073709551616"                                                           // one greater than largest uint64 (^uint64(0))
	invalidLargeAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
)

var (
	sender   = secp256k1.GenPrivKey().PubKey().Address().String()
	receiver = sdk.AccAddress("testaddr2").String()
)

func TestUnmarshalPacketData(t *testing.T) {
	packetDataV1 := types.FungibleTokenPacketData{
		Sender:   "sender",
		Receiver: "recv",
		Memo:     "memo",
		Amount:   "10000",
		Denom:    "uatom",
	}

	bz, err := types.ModuleCdc.Marshal(&packetDataV1)
	require.NoError(t, err)

	var packetDataV2 FungibleTokenPacketData
	err = types.ModuleCdc.Unmarshal(bz, &packetDataV2)
	require.NoError(t, err)

	t.Logf("successfully unmarshalled to packet data v2: %+v", packetDataV2)

	packetDataV2 = FungibleTokenPacketData{
		Sender:   "sender",
		Receiver: "recv",
		Memo:     "memo",
		Tokens: []*Token{
			{
				Denom:  "uatom",
				Amount: "10000",
				Trace:  []string{"transfer/channel-100"},
			},
		},
	}

	bz, err = types.ModuleCdc.Marshal(&packetDataV2)
	require.NoError(t, err)

	packetDataV1 = types.FungibleTokenPacketData{}
	err = types.ModuleCdc.Unmarshal(bz, &packetDataV1)
	require.NoError(t, err)

	t.Logf("successfully unmarshalled to packet data v1: %+v", packetDataV1)
}

func TestUnmarshalPacketDataJSON(t *testing.T) {
	packetDataV1 := types.FungibleTokenPacketData{
		Sender:   "sender",
		Receiver: "recv",
		Memo:     "memo",
		Amount:   "10000",
		Denom:    "uatom",
	}

	bz, err := json.Marshal(&packetDataV1)
	require.NoError(t, err)

	var packetDataV2 FungibleTokenPacketData
	err = json.Unmarshal(bz, &packetDataV2)
	require.NoError(t, err)

	t.Logf("successfully unmarshalled to packet data v2: %+v", packetDataV2)

	packetDataV2 = FungibleTokenPacketData{
		Sender:   "sender",
		Receiver: "recv",
		Memo:     "memo",
		Tokens: []*Token{
			{
				Denom:  "uatom",
				Amount: "10000",
				Trace:  []string{"transfer/channel-100"},
			},
		},
	}

	bz, err = json.Marshal(&packetDataV2)
	require.NoError(t, err)

	packetDataV1 = types.FungibleTokenPacketData{}
	err = json.Unmarshal(bz, &packetDataV1)
	require.NoError(t, err)

	t.Logf("successfully unmarshalled to packet data v1: %+v", packetDataV1)
}

func TestUnmarshalPacketDataProtoJSON(t *testing.T) {
	// NOTE: copied from versinos of ibc-go where we use a custom json marshal
	mustProtoMarshalJSON := func(msg proto.Message) []byte {
		anyResolver := codectypes.NewInterfaceRegistry()

		// EmitDefaults is set to false to prevent marshalling of unpopulated fields (memo)
		// OrigName and the anyResovler match the fields the original SDK function would expect
		// in order to minimize changes.

		// OrigName is true since there is no particular reason to use camel case
		// The any resolver is empty, but provided anyways.
		jm := &jsonpb.Marshaler{OrigName: true, EmitDefaults: false, AnyResolver: anyResolver}

		err := codectypes.UnpackInterfaces(msg, codectypes.ProtoJSONPacker{JSONPBMarshaler: jm})
		if err != nil {
			panic(err)
		}

		buf := new(bytes.Buffer)
		if err := jm.Marshal(buf, msg); err != nil {
			panic(err)
		}

		// sort JSON used in packetData.GetBytes()
		return sdk.MustSortJSON(buf.Bytes())
	}

	packetDataV1 := types.FungibleTokenPacketData{
		Sender:   "sender",
		Receiver: "recv",
		Memo:     "memo",
		// Amount:   "10000", // fails if amount is filled
		// Denom: "uatom", // fails if denom is filled
	}

	bz := mustProtoMarshalJSON(&packetDataV1)

	var packetDataV2 FungibleTokenPacketData
	err := types.ModuleCdc.UnmarshalJSON(bz, &packetDataV2)
	require.NoError(t, err)

	t.Logf("successfully unmarshalled to packet data v2: %+v", packetDataV2)

	packetDataV2 = FungibleTokenPacketData{
		Sender:   "sender",
		Receiver: "recv",
		Memo:     "memo",
		// fails if tokens are filled
		// Tokens: []*Token{
		// 	{
		// 		Denom:  "uatom",
		// 		Amount: "10000",
		// 		Trace:  []string{"transfer/channel-100"},
		// 	},
		// },
	}

	bz = mustProtoMarshalJSON(&packetDataV2)
	require.NoError(t, err)

	packetDataV1 = types.FungibleTokenPacketData{}
	err = types.ModuleCdc.UnmarshalJSON(bz, &packetDataV1)
	require.NoError(t, err)

	t.Logf("successfully unmarshalled to packet data v1: %+v", packetDataV1)
}

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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
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
						Amount: amount,
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
						Denom:  denom,
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
						Denom:  denom,
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
						Denom:  denom,
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
						Denom:  denom,
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
		t.Run(tc.name, func(t *testing.T) {
			err := tc.packetData.ValidateBasic()

			expPass := tc.expErr == nil
			if expPass {
				require.NoError(t, err, tc.name)
			} else {
				require.ErrorContains(t, err, tc.expErr.Error(), tc.name)
			}
		})
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expSender, tc.packetData.GetPacketSender(types.PortID))
		})
	}
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
		t.Run(tc.name, func(t *testing.T) {
			customData := tc.packetData.GetCustomPacketData("src_callback")
			require.Equal(t, tc.expCustomData, customData)
		})
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
						Denom:  denom,
						Amount: amount,
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
						Denom:  denom,
						Amount: amount,
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
		t.Run(tc.name, func(t *testing.T) {
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
		})
	}
}
