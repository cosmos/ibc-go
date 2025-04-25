package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name     string
		typeURL  string
		expError error
	}{
		{
			"success: Packet",
			sdk.MsgTypeURL(&types.Packet{}),
			nil,
		},
		{
			"success: MsgChannelOpenInit",
			sdk.MsgTypeURL(&types.MsgChannelOpenInit{}),
			nil,
		},
		{
			"success: MsgChannelOpenTry",
			sdk.MsgTypeURL(&types.MsgChannelOpenTry{}),
			nil,
		},
		{
			"success: MsgChannelOpenAck",
			sdk.MsgTypeURL(&types.MsgChannelOpenAck{}),
			nil,
		},
		{
			"success: MsgChannelOpenConfirm",
			sdk.MsgTypeURL(&types.MsgChannelOpenConfirm{}),
			nil,
		},
		{
			"success: MsgChannelCloseInit",
			sdk.MsgTypeURL(&types.MsgChannelCloseInit{}),
			nil,
		},
		{
			"success: MsgChannelCloseConfirm",
			sdk.MsgTypeURL(&types.MsgChannelCloseConfirm{}),
			nil,
		},
		{
			"success: MsgRecvPacket",
			sdk.MsgTypeURL(&types.MsgRecvPacket{}),
			nil,
		},
		{
			"success: MsgAcknowledgement",
			sdk.MsgTypeURL(&types.MsgAcknowledgement{}),
			nil,
		},
		{
			"success: MsgTimeout",
			sdk.MsgTypeURL(&types.MsgTimeout{}),
			nil,
		},
		{
			"success: MsgTimeoutOnClose",
			sdk.MsgTypeURL(&types.MsgTimeoutOnClose{}),
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
			encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
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
