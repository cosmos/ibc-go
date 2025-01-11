package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ica "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name    string
		typeURL string
		errMsg  string
	}{
		{
			"success: MsgUpdateParams",
			sdk.MsgTypeURL(&types.MsgUpdateParams{}),
			"",
		},
		{
			"success: MsgModuleQuerySafe",
			sdk.MsgTypeURL(&types.MsgModuleQuerySafe{}),
			"",
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			"unable to resolve type URL",
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, ica.AppModule{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			if tc.errMsg == "" {
				require.NotNil(t, msg)
				require.NoError(t, err)
			} else {
				require.Nil(t, msg)
				require.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}
