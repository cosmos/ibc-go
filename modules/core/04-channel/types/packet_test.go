package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
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
		packet   types.Packet
		expError error
	}{
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), nil},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, disabledTimeout, timeoutTimestamp), nil},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, 0), nil},
		{types.NewPacket(unknownPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), nil},
		{types.NewPacket(validPacketData, 0, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), errors.New("packet sequence cannot be 0: invalid packet")},
		{types.NewPacket(validPacketData, 1, invalidPort, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), errors.New("invalid source port")},
		{types.NewPacket(validPacketData, 1, portid, invalidChannel, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), errors.New("invalid source channel")},
		{types.NewPacket(validPacketData, 1, portid, chanid, invalidPort, cpchanid, timeoutHeight, timeoutTimestamp), errors.New("invalid destination port")},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, invalidChannel, timeoutHeight, timeoutTimestamp), errors.New("invalid destination channel")},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, disabledTimeout, 0), errors.New("packet timeout height and packet timeout timestamp cannot both be 0: invalid packet")},
		{types.NewPacket(make([]byte, types.MaximumPayloadsSize+1), 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), errors.New("packet data bytes cannot exceed 262144 bytes: invalid packet")},
	}

	for _, tc := range testCases {
		err := tc.packet.ValidateBasic()
		if tc.expError == nil {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, tc.expError.Error())
		}
	}
}
