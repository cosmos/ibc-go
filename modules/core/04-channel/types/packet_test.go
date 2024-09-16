package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

func TestCommitPacket(t *testing.T) {
	// V1 packet1 commitment
	packet1 := types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp)
	commitment1 := types.CommitPacket(packet1)
	require.NotNil(t, commitment1)

	// V2 packet commitment with empty app version
	packet2 := types.NewPacketWithVersion(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp, "")
	commitment2 := types.CommitPacket(packet2)
	require.NotNil(t, commitment2)

	// even though app version is empty for both packet1 and packet2
	// the commitment is different because we use Eureka protocol for packet2
	require.NotEqual(t, commitment1, commitment2)

	// V2 packet commitment with non-empty app version
	packet3 := types.NewPacketWithVersion(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp, validVersion)
	commitment3 := types.CommitPacket(packet3)
	require.NotNil(t, commitment3)

	require.NotEqual(t, commitment1, commitment3)
	require.NotEqual(t, commitment2, commitment3)

	// V2 packet commitment with non-empty app version and zero timeout height
	packet4 := types.NewPacketWithVersion(validPacketData, 1, portid, chanid, cpportid, cpchanid, clienttypes.ZeroHeight(), timeoutTimestamp, validVersion)
	commitment4 := types.CommitPacket(packet4)
	require.NotNil(t, commitment4)

	require.NotEqual(t, commitment1, commitment4)
	require.NotEqual(t, commitment2, commitment4)
	require.NotEqual(t, commitment3, commitment4)
}

func TestPacketValidateBasic(t *testing.T) {
	testCases := []struct {
		packet  types.Packet
		expPass bool
		errMsg  string
	}{
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), true, ""},
		{types.NewPacket(unknownPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), true, ""},
		{types.NewPacket(validPacketData, 0, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), false, "invalid sequence"},
		{types.NewPacket(validPacketData, 1, invalidPort, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), false, "invalid source port"},
		{types.NewPacket(validPacketData, 1, portid, invalidChannel, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), false, "invalid source channel"},
		{types.NewPacket(validPacketData, 1, portid, chanid, invalidPort, cpchanid, timeoutHeight, timeoutTimestamp), false, "invalid destination port"},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, invalidChannel, timeoutHeight, timeoutTimestamp), false, "invalid destination channel"},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, disabledTimeout, 0), false, "disabled both timeout height and timestamp"},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, disabledTimeout, timeoutTimestamp), true, "disabled timeout height, valid timeout timestamp"},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, 0), true, "disabled timeout timestamp, valid timeout height"},
		{types.NewPacketWithVersion(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp, "version"), true, "valid v2 packet"},
		{types.Packet{1, portid, chanid, cpportid, cpchanid, validPacketData, timeoutHeight, timeoutTimestamp, types.IBC_VERSION_1, "version", "json"}, false, "invalid specifying of app version with protocol version 1"},
		{types.Packet{1, portid, chanid, cpportid, cpchanid, validPacketData, timeoutHeight, timeoutTimestamp, types.IBC_VERSION_UNSPECIFIED, "version", "json"}, false, "invalid specifying of app version with unspecified protocol version"},
		{types.NewPacketWithVersion(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp, ""), false, "app version must be specified when packet uses protocol version 2"},
		{types.NewPacketWithVersion(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp, "      "), false, "app version must be specified when packet uses protocol version 2"},
	}

	for i, tc := range testCases {
		tc := tc

		err := tc.packet.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "Msg %d failed: %s", i, tc.errMsg)
		} else {
			require.Error(t, err, "Invalid Msg %d passed: %s", i, tc.errMsg)
		}
	}
}
