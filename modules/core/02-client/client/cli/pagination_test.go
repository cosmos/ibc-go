package cli_test

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// TestPageKeyRoundTrip pins the assumption every paginated IBC CLI query relies on:
// the codec behind clientCtx.PrintProto emits next_key as standard padded base64, and
// feeding that printed string back through --page-key yields the original cursor bytes.
// If the CLI output encoding ever changes, this fails instead of silently returning
// wrong pages.
func TestPageKeyRoundTrip(t *testing.T) {
	// not valid UTF-8, and exercises the +, / and padding edges of the base64 alphabet
	raw := []byte{0x01, 0xfb, 0xff, 0x61, 0x2f, 0x2b, 0x00, 0x7f}

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bz, err := cdc.MarshalJSON(&query.PageResponse{NextKey: raw})
	require.NoError(t, err)

	var printed struct {
		NextKey string `json:"next_key"`
	}
	require.NoError(t, json.Unmarshal(bz, &printed))
	require.Equal(t, "Afv/YS8rAH8=", printed.NextKey)

	cmd := &cobra.Command{}
	flags.AddPaginationFlagsToCmd(cmd, "test")
	require.NoError(t, cmd.Flags().Set(flags.FlagPageKey, printed.NextKey))

	flagSet, err := client.FlagSetWithPageKeyDecoded(cmd.Flags())
	require.NoError(t, err)

	pageReq, err := client.ReadPageRequest(flagSet)
	require.NoError(t, err)
	require.Equal(t, raw, pageReq.Key)
}
