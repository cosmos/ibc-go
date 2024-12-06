package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

func TestCommitPacket(t *testing.T) {
	packet := types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp)
	commitment := types.CommitPacket(packet)
	require.NotNil(t, commitment)

	testCases := []struct {
		name   string
		packet types.Packet
	}{
		{
			name:   "diff data",
			packet: types.NewPacket(unknownPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp),
		},
		{
			name:   "diff timeout revision number",
			packet: types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, clienttypes.NewHeight(timeoutHeight.RevisionNumber+1, timeoutHeight.RevisionHeight), timeoutTimestamp),
		},
		{
			name:   "diff timeout revision height",
			packet: types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, clienttypes.NewHeight(timeoutHeight.RevisionNumber, timeoutHeight.RevisionHeight+1), timeoutTimestamp),
		},
		{
			name:   "diff timeout timestamp",
			packet: types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, uint64(1)),
		},
	}

	for _, tc := range testCases {
		testCommitment := types.CommitPacket(tc.packet)
		require.NotNil(t, testCommitment)

		require.NotEqual(t, commitment, testCommitment)
	}
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
	}

	for i, tc := range testCases {
		err := tc.packet.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "Msg %d failed: %s", i, tc.errMsg)
		} else {
			require.Error(t, err, "Invalid Msg %d passed: %s", i, tc.errMsg)
		}
	}
}
