package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

func TestCodecTypeRegistration(t *testing.T) {
	testCases := []struct {
		name    string
		typeURL string
		expPass bool
	}{
		{
			"success: Channel",
			sdk.MsgTypeURL(&types.Channel{}),
			true,
		},
		{
			"success: Counterparty",
			sdk.MsgTypeURL(&types.Counterparty{}),
			true,
		},
		{
			"success: Packet",
			sdk.MsgTypeURL(&types.Packet{}),
			true,
		},
		{
			"success: MsgChannelOpenInit",
			sdk.MsgTypeURL(&types.MsgChannelOpenInit{}),
			true,
		},
		{
			"success: MsgChannelOpenTry",
			sdk.MsgTypeURL(&types.MsgChannelOpenTry{}),
			true,
		},
		{
			"success: MsgChannelOpenAck",
			sdk.MsgTypeURL(&types.MsgChannelOpenAck{}),
			true,
		},
		{
			"success: MsgChannelOpenConfirm",
			sdk.MsgTypeURL(&types.MsgChannelOpenConfirm{}),
			true,
		},
		{
			"success: MsgChannelCloseInit",
			sdk.MsgTypeURL(&types.MsgChannelCloseInit{}),
			true,
		},
		{
			"success: MsgChannelCloseConfirm",
			sdk.MsgTypeURL(&types.MsgChannelCloseConfirm{}),
			true,
		},
		{
			"success: MsgRecvPacket",
			sdk.MsgTypeURL(&types.MsgRecvPacket{}),
			true,
		},
		{
			"success: MsgAcknowledgement",
			sdk.MsgTypeURL(&types.MsgAcknowledgement{}),
			true,
		},
		{
			"success: MsgTimeout",
			sdk.MsgTypeURL(&types.MsgTimeout{}),
			true,
		},
		{
			"success: MsgTimeoutOnClose",
			sdk.MsgTypeURL(&types.MsgTimeoutOnClose{}),
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
