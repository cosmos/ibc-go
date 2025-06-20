package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ratelimiting "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name    string
		typeURL string
		errMsg  string
	}{
		{
			"success: MsgAddRateLimit",
			sdk.MsgTypeURL(&types.MsgAddRateLimit{}),
			"",
		},
		{
			"success: MsgUpdateRateLimit",
			sdk.MsgTypeURL(&types.MsgUpdateRateLimit{}),
			"",
		},
		{
			"success: MsgRemoveRateLimit",
			sdk.MsgTypeURL(&types.MsgRemoveRateLimit{}),
			"",
		},
		{
			"success: MsgResetRateLimit",
			sdk.MsgTypeURL(&types.MsgResetRateLimit{}),
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
			encodingCfg := moduletestutil.MakeTestEncodingConfig(ratelimiting.AppModuleBasic{})
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
