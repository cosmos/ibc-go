package solomachine_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name    string
		typeURL string
		expPass bool
	}{
		{
			"success: ClientState",
			sdk.MsgTypeURL(&solomachine.ClientState{}),
			true,
		},
		{
			"success: ConsensusState",
			sdk.MsgTypeURL(&solomachine.ConsensusState{}),
			true,
		},
		{
			"success: Header",
			sdk.MsgTypeURL(&solomachine.Header{}),
			true,
		},
		{
			"success: Misbehaviour",
			sdk.MsgTypeURL(&solomachine.Misbehaviour{}),
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
			encodingCfg := moduletestutil.MakeTestEncodingConfig(solomachine.AppModuleBasic{})
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
