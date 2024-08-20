package types_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	fee "github.com/cosmos/ibc-go/v9/modules/apps/29-fee"
	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name    string
		typeURL string
		expErr  error
	}{
		{
			"success: MsgPayPacketFee",
			sdk.MsgTypeURL(&types.MsgPayPacketFee{}),
			nil,
		},
		{
			"success: MsgPayPacketFeeAsync",
			sdk.MsgTypeURL(&types.MsgPayPacketFeeAsync{}),
			nil,
		},
		{
			"success: MsgRegisterPayee",
			sdk.MsgTypeURL(&types.MsgRegisterPayee{}),
			nil,
		},
		{
			"success: MsgRegisterCounterpartyPayee",
			sdk.MsgTypeURL(&types.MsgRegisterCounterpartyPayee{}),
			nil,
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			errors.New("unable to resolve type URL ibc.invalid.MsgTypeURL"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(fee.AppModuleBasic{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			if tc.expErr == nil {
				require.NotNil(t, msg)
				require.NoError(t, err)
			} else {
				require.Nil(t, msg)
				require.True(t, errors.Is(err, tc.expErr) || strings.Contains(err.Error(), tc.expErr.Error()), err.Error())
			}
		})
	}
}
