package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v9/modules/core"
	"github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name     string
		typeURL  string
		expError error
	}{
		{
			"success: MsgConnectionOpenInit",
			sdk.MsgTypeURL(&types.MsgConnectionOpenInit{}),
			nil,
		},
		{
			"success: MsgConnectionOpenTry",
			sdk.MsgTypeURL(&types.MsgConnectionOpenTry{}),
			nil,
		},
		{
			"success: MsgConnectionOpenAck",
			sdk.MsgTypeURL(&types.MsgConnectionOpenAck{}),
			nil,
		},
		{
			"success: MsgConnectionOpenConfirm",
			sdk.MsgTypeURL(&types.MsgConnectionOpenConfirm{}),
			nil,
		},
		{
			"success: MsgUpdateParams",
			sdk.MsgTypeURL(&types.MsgUpdateParams{}),
			nil,
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			fmt.Errorf("unable to resolve type URL ibc.invalid.MsgTypeURL"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, ibc.AppModule{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			if tc.expError == nil {
				require.NotNil(t, msg)
				require.NoError(t, err)
			} else {
				require.Nil(t, msg)
				require.ErrorContains(t, err, tc.expError.Error())
			}
		})
	}
}
