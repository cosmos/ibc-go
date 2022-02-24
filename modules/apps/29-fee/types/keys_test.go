package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func TestKeyCounterpartyRelayer(t *testing.T) {
	var (
		relayerAddress = "relayer_address"
		channelID      = "channel-0"
	)

	key := types.KeyCounterpartyRelayer(relayerAddress, channelID)
	require.Equal(t, string(key), fmt.Sprintf("%s/%s/%s", types.CounterpartyRelayerAddressKeyPrefix, relayerAddress, channelID))
}

func TestParseKeyFeesInEscrow(t *testing.T) {
	validPacketID := channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 1)

	testCases := []struct {
		name    string
		key     string
		expPass bool
	}{
		{
			"success",
			string(types.KeyFeesInEscrow(validPacketID)),
			true,
		},
		{
			"incorrect key - key split has incorrect length",
			string(types.FeeEnabledKey(validPacketID.PortId, validPacketID.ChannelId)),
			false,
		},
		{
			"incorrect key - sequence cannot be parsed",
			fmt.Sprintf("%s/%s", types.KeyFeesInEscrowChannelPrefix(validPacketID.PortId, validPacketID.ChannelId), "sequence"),
			false,
		},
	}

	for _, tc := range testCases {
		packetId, err := types.ParseKeyFeesInEscrow(tc.key)

		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, validPacketID, packetId)
		} else {
			require.Error(t, err)
		}
	}
}
