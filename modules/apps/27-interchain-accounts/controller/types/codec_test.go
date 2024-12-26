package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ica "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name    string
		typeURL string
		expErr  error
	}{
		{
			"success: MsgRegisterInterchainAccount",
			sdk.MsgTypeURL(&types.MsgRegisterInterchainAccount{}),
			nil,
		},
		{
			"success: MsgSendTx",
			sdk.MsgTypeURL(&types.MsgSendTx{}),
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
			fmt.Errorf("unable to resolve type URL"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, ica.AppModule{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			fmt.Printf("%+v\n", err)

			if tc.expErr == nil {
				require.NotNil(t, msg)
				require.NoError(t, err)
			} else {
				require.Nil(t, msg)
				require.ErrorContains(t, err, tc.expErr.Error())
			}
		})
	}
}
