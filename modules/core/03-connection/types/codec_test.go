package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v9/modules/core"
	"github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name    string
		typeURL string
		expPass bool
	}{
		{
			"success: MsgConnectionOpenInit",
			sdk.MsgTypeURL(&types.MsgConnectionOpenInit{}),
			true,
		},
		{
			"success: MsgConnectionOpenTry",
			sdk.MsgTypeURL(&types.MsgConnectionOpenTry{}),
			true,
		},
		{
			"success: MsgConnectionOpenAck",
			sdk.MsgTypeURL(&types.MsgConnectionOpenAck{}),
			true,
		},
		{
			"success: MsgConnectionOpenConfirm",
			sdk.MsgTypeURL(&types.MsgConnectionOpenConfirm{}),
			true,
		},
		{
			"success: MsgUpdateParams",
			sdk.MsgTypeURL(&types.MsgUpdateParams{}),
			true,
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			if tc.expPass {
				require.NotNil(t, msg)
				require.NoError(t, err)
			} else {
				require.Nil(t, msg)
				require.Error(t, err)
			}
		})
	}
}
