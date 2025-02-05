package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v9/modules/core"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
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
			"success: MsgChannelUpgradeInit",
			sdk.MsgTypeURL(&types.MsgChannelUpgradeInit{}),
			nil,
		},
		{
			"success: MsgChannelUpgradeTry",
			sdk.MsgTypeURL(&types.MsgChannelUpgradeTry{}),
			nil,
		},
		{
			"success: MsgChannelUpgradeAck",
			sdk.MsgTypeURL(&types.MsgChannelUpgradeAck{}),
			nil,
		},
		{
			"success: MsgChannelUpgradeConfirm",
			sdk.MsgTypeURL(&types.MsgChannelUpgradeConfirm{}),
			nil,
		},
		{
			"success: MsgChannelUpgradeOpen",
			sdk.MsgTypeURL(&types.MsgChannelUpgradeOpen{}),
			nil,
		},
		{
			"success: MsgChannelUpgradeTimeout",
			sdk.MsgTypeURL(&types.MsgChannelUpgradeTimeout{}),
			nil,
		},
		{
			"success: MsgChannelUpgradeCancel",
			sdk.MsgTypeURL(&types.MsgChannelUpgradeCancel{}),
			nil,
		},
		{
			"success: MsgPruneAcknowledgements",
			sdk.MsgTypeURL(&types.MsgPruneAcknowledgements{}),
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
			fmt.Errorf("unable to resolve type URL ibc.invalid.MsgTypeURL"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, ibc.AppModule{})
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
