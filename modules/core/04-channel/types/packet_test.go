package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

func TestCommitPacket(t *testing.T) {
	basePacket := types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp)

	registry := codectypes.NewInterfaceRegistry()
	clienttypes.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)

	baseCommitment := types.CommitPacket(cdc, basePacket)
	require.NotNil(t, baseCommitment)

	diffValidPackedData := []byte("anotherpackeddata")

	diffHeightRevisionNumber := timeoutHeight
	diffHeightRevisionNumber.RevisionNumber++

	diffHeightRevisionHeight := timeoutHeight
	diffHeightRevisionHeight.RevisionHeight++

	diffTimeout := uint64(101)

	diffHeightRevision := timeoutHeight
	diffHeightRevision.RevisionHeight++
	diffHeightRevision.RevisionNumber++

	testCases := []struct {
		name   string
		packet types.Packet
	}{
		{
			name:   "diff data",
			packet: types.NewPacket(diffValidPackedData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp),
		},
		{
			name:   "diff timeout revision number",
			packet: types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, diffHeightRevisionNumber, timeoutTimestamp),
		},
		{
			name:   "diff timeout revision height",
			packet: types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, diffHeightRevisionHeight, timeoutTimestamp),
		},
		{
			name:   "diff timeout timestamp",
			packet: types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, diffTimeout),
		},
		{
			name:   "diff every field",
			packet: types.NewPacket(diffValidPackedData, 1, portid, chanid, cpportid, cpchanid, diffHeightRevision, diffTimeout),
		},
	}

	for _, tc := range testCases {
		commitment := types.CommitPacket(cdc, tc.packet)
		require.NotNil(t, commitment)
		require.False(t, string(commitment) == string(baseCommitment), tc.name)
	}
}

func TestPacketValidateBasic(t *testing.T) {
	testCases := []struct {
		packet  types.Packet
		expPass bool
		errMsg  string
	}{
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), true, ""},
		{types.NewPacket(validPacketData, 0, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), false, "invalid sequence"},
		{types.NewPacket(validPacketData, 1, invalidPort, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), false, "invalid source port"},
		{types.NewPacket(validPacketData, 1, portid, invalidChannel, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), false, "invalid source channel"},
		{types.NewPacket(validPacketData, 1, portid, chanid, invalidPort, cpchanid, timeoutHeight, timeoutTimestamp), false, "invalid destination port"},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, invalidChannel, timeoutHeight, timeoutTimestamp), false, "invalid destination channel"},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, disabledTimeout, 0), false, "disabled both timeout height and timestamp"},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, disabledTimeout, timeoutTimestamp), true, "disabled timeout height, valid timeout timestamp"},
		{types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, 0), true, "disabled timeout timestamp, valid timeout height"},
		{types.NewPacket(unknownPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp), true, ""},
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
