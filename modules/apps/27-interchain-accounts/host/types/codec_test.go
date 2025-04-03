package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ica "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
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
		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(ica.AppModuleBasic{})
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
