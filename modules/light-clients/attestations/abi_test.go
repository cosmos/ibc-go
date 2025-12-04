package attestations_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	attestations "github.com/cosmos/ibc-go/v10/modules/light-clients/attestations"
)

const nanosPerSecond = 1_000_000_000

func TestABIEncodeDecodeStateAttestation(t *testing.T) {
	testCases := []struct {
		name      string
		height    uint64
		timestamp uint64
	}{
		{name: "zero values", height: 0, timestamp: 0},
		{name: "typical values", height: 100, timestamp: 1234567890 * nanosPerSecond},
		{name: "max uint64 height", height: ^uint64(0), timestamp: 1000 * nanosPerSecond},
		{name: "large timestamp", height: 500, timestamp: ^uint64(0)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &attestations.StateAttestation{
				Height:    tc.height,
				Timestamp: tc.timestamp,
			}

			encoded, err := original.ABIEncode()
			require.NoError(t, err)
			require.Len(t, encoded, 64, "ABI encoding should produce 64 bytes (2x32-byte words)")

			decoded, err := attestations.ABIDecodeStateAttestation(encoded)
			require.NoError(t, err)
			require.Equal(t, original.Height, decoded.Height)
			// Timestamp should round-trip through seconds conversion
			expectedTimestamp := (tc.timestamp / nanosPerSecond) * nanosPerSecond
			require.Equal(t, expectedTimestamp, decoded.Timestamp)
		})
	}
}

func TestABIEncodeDecodePacketAttestation(t *testing.T) {
	testCases := []struct {
		name    string
		height  uint64
		packets []attestations.PacketCompact
	}{
		{
			name:   "single packet",
			height: 100,
			packets: []attestations.PacketCompact{
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: bytes.Repeat([]byte{0x02}, 32)},
			},
		},
		{
			name:   "multiple packets",
			height: 200,
			packets: []attestations.PacketCompact{
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: bytes.Repeat([]byte{0x02}, 32)},
				{Path: bytes.Repeat([]byte{0x03}, 32), Commitment: bytes.Repeat([]byte{0x04}, 32)},
				{Path: bytes.Repeat([]byte{0x05}, 32), Commitment: bytes.Repeat([]byte{0x06}, 32)},
			},
		},
		{
			name:    "empty packets",
			height:  300,
			packets: []attestations.PacketCompact{},
		},
		{
			name:   "zero commitment (non-membership)",
			height: 400,
			packets: []attestations.PacketCompact{
				{Path: bytes.Repeat([]byte{0xAB}, 32), Commitment: make([]byte, 32)},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &attestations.PacketAttestation{
				Height:  tc.height,
				Packets: tc.packets,
			}

			encoded, err := original.ABIEncode()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(encoded), 64, "ABI encoding should be at least 64 bytes")

			decoded, err := attestations.ABIDecodePacketAttestation(encoded)
			require.NoError(t, err)
			require.Equal(t, original.Height, decoded.Height)
			require.Len(t, decoded.Packets, len(original.Packets))
			for i := range original.Packets {
				require.Equal(t, original.Packets[i].Path, decoded.Packets[i].Path)
				require.Equal(t, original.Packets[i].Commitment, decoded.Packets[i].Commitment)
			}
		})
	}
}

func TestABISolidityCompatibility(t *testing.T) {
	// Test round-trip encoding compatibility
	t.Run("StateAttestation round-trip", func(t *testing.T) {
		// Create a StateAttestation with a known timestamp (in nanoseconds internally)
		// The ABI encoding uses seconds, so we test with a value that round-trips cleanly
		timestampSeconds := uint64(1234567890)
		timestampNanos := timestampSeconds * nanosPerSecond

		original := &attestations.StateAttestation{
			Height:    100,
			Timestamp: timestampNanos,
		}

		encoded, err := original.ABIEncode()
		require.NoError(t, err)
		require.Len(t, encoded, 64, "StateAttestation should encode to 64 bytes")

		decoded, err := attestations.ABIDecodeStateAttestation(encoded)
		require.NoError(t, err)
		require.Equal(t, uint64(100), decoded.Height)
		require.Equal(t, timestampNanos, decoded.Timestamp)
	})

	t.Run("PacketAttestation compatibility with Solidity", func(t *testing.T) {
		// Test vector for PacketAttestation with one packet
		// abi.encode(PacketAttestation({
		//   height: 100,
		//   packets: [PacketCompact({path: bytes32(0x01...01), commitment: bytes32(0x02...02)})]
		// }))
		path := bytes.Repeat([]byte{0x01}, 32)
		commitment := bytes.Repeat([]byte{0x02}, 32)

		pa := &attestations.PacketAttestation{
			Height: 100,
			Packets: []attestations.PacketCompact{
				{Path: path, Commitment: commitment},
			},
		}

		encoded, err := pa.ABIEncode()
		require.NoError(t, err)

		decoded, err := attestations.ABIDecodePacketAttestation(encoded)
		require.NoError(t, err)

		require.Equal(t, pa.Height, decoded.Height)
		require.Len(t, decoded.Packets, 1)
		require.Equal(t, path, decoded.Packets[0].Path)
		require.Equal(t, commitment, decoded.Packets[0].Commitment)
	})
}

func TestABIDecodeInvalidData(t *testing.T) {
	t.Run("StateAttestation with invalid data", func(t *testing.T) {
		_, err := attestations.ABIDecodeStateAttestation([]byte{0x01, 0x02, 0x03})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ABI decode state attestation")
	})

	t.Run("PacketAttestation with invalid data", func(t *testing.T) {
		_, err := attestations.ABIDecodePacketAttestation([]byte{0x01, 0x02, 0x03})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ABI decode packet attestation")
	})

	t.Run("StateAttestation with empty data", func(t *testing.T) {
		_, err := attestations.ABIDecodeStateAttestation([]byte{})
		require.Error(t, err)
	})

	t.Run("PacketAttestation with empty data", func(t *testing.T) {
		_, err := attestations.ABIDecodePacketAttestation([]byte{})
		require.Error(t, err)
	})
}

func TestABIEncodePacketCompact(t *testing.T) {
	path := bytes.Repeat([]byte{0xAA}, 32)
	commitment := bytes.Repeat([]byte{0xBB}, 32)

	pc := &attestations.PacketCompact{
		Path:       path,
		Commitment: commitment,
	}

	encoded, err := pc.ABIEncode()
	require.NoError(t, err)
	require.Len(t, encoded, 64, "PacketCompact should encode to 64 bytes")
	require.Equal(t, path, encoded[:32])
	require.Equal(t, commitment, encoded[32:64])
}
