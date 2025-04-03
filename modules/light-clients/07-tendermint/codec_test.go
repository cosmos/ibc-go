package tendermint_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	tendermint "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name     string
		typeURL  string
		expError error
	}{
		{
			"success: ClientState",
			sdk.MsgTypeURL(&tendermint.ClientState{}),
			nil,
		},
		{
			"success: ConsensusState",
			sdk.MsgTypeURL(&tendermint.ConsensusState{}),
			nil,
		},
		{
			"success: Header",
			sdk.MsgTypeURL(&tendermint.Header{}),
			nil,
		},
		{
			"success: Misbehaviour",
			sdk.MsgTypeURL(&tendermint.Misbehaviour{}),
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
			encodingCfg := moduletestutil.MakeTestEncodingConfig(tendermint.AppModuleBasic{})
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
