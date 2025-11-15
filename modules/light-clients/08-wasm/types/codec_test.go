package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	wasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name     string
		typeURL  string
		expError error
	}{
		{
			"success: ClientState",
			sdk.MsgTypeURL(&types.ClientState{}),
			nil,
		},
		{
			"success: ConsensusState",
			sdk.MsgTypeURL(&types.ConsensusState{}),
			nil,
		},
		{
			"success: ClientMessage",
			sdk.MsgTypeURL(&types.ClientMessage{}),
			nil,
		},
		{
			"success: MsgStoreCode",
			sdk.MsgTypeURL(&types.MsgStoreCode{}),
			nil,
		},
		{
			"success: MsgMigrateContract",
			sdk.MsgTypeURL(&types.MsgMigrateContract{}),
			nil,
		},
		{
			"success: MsgRemoveChecksum",
			sdk.MsgTypeURL(&types.MsgRemoveChecksum{}),
			nil,
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			errors.New("unable to resolve type URL ibc.invalid.MsgTypeURL"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(wasm.AppModuleBasic{})
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
