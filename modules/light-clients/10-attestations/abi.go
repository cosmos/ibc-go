package attestations

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	errorsmod "cosmossdk.io/errors"
)

const (
	nanosPerSecond = 1_000_000_000
)

var (
	uint64Type, _  = abi.NewType("uint64", "", nil)
	bytes32Type, _ = abi.NewType("bytes32", "", nil)
	tupleArrayType = abi.Arguments{
		{Name: "path", Type: bytes32Type},
		{Name: "commitment", Type: bytes32Type},
	}

	stateAttestationArgs = abi.Arguments{
		{Name: "height", Type: uint64Type},
		{Name: "timestamp", Type: uint64Type},
	}

	packetCompactTupleType, _ = abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "path", Type: "bytes32"},
		{Name: "commitment", Type: "bytes32"},
	})

	packetAttestationArgs = abi.Arguments{
		{Name: "height", Type: uint64Type},
		{Name: "packets", Type: packetCompactTupleType},
	}
)

// ABIPacketCompact is the ABI-compatible representation with fixed-size arrays.
type ABIPacketCompact struct {
	Path       [32]byte
	Commitment [32]byte
}

// StateAttestation is used by client updates.
// This type uses ABI encoding (not Protobuf) for cross-platform compatibility.
type StateAttestation struct {
	Height    uint64
	Timestamp uint64
}

// PacketAttestation is used by membership queries.
// This type uses ABI encoding (not Protobuf) for cross-platform compatibility.
type PacketAttestation struct {
	Height  uint64
	Packets []PacketCompact
}

// PacketCompact represents a packet commitment.
// This type uses ABI encoding (not Protobuf) for cross-platform compatibility.
type PacketCompact struct {
	Path       []byte
	Commitment []byte
}

func (sa *StateAttestation) ABIEncode() ([]byte, error) {
	timestampSeconds := sa.Timestamp / nanosPerSecond
	return stateAttestationArgs.Pack(sa.Height, timestampSeconds)
}

func (pa *PacketAttestation) ABIEncode() ([]byte, error) {
	packets := make([]ABIPacketCompact, len(pa.Packets))
	for i, p := range pa.Packets {
		packets[i] = ABIPacketCompact{
			Path:       bytesToBytes32(p.Path),
			Commitment: bytesToBytes32(p.Commitment),
		}
	}
	return packetAttestationArgs.Pack(pa.Height, packets)
}

func ABIDecodePacketAttestation(data []byte) (*PacketAttestation, error) {
	unpacked, err := packetAttestationArgs.Unpack(data)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrInvalidAttestationData, "failed to ABI decode packet attestation: %v", err)
	}

	if len(unpacked) != 2 {
		return nil, errorsmod.Wrap(ErrInvalidAttestationData, "invalid packet attestation: expected 2 fields")
	}

	height, ok := unpacked[0].(uint64)
	if !ok {
		return nil, errorsmod.Wrap(ErrInvalidAttestationData, "invalid height type")
	}

	abiPackets, ok := unpacked[1].([]struct {
		Path       [32]byte `json:"path"`
		Commitment [32]byte `json:"commitment"`
	})
	if !ok {
		return nil, errorsmod.Wrap(ErrInvalidAttestationData, "invalid packets type")
	}

	packets := make([]PacketCompact, len(abiPackets))
	for i, p := range abiPackets {
		packets[i] = PacketCompact{
			Path:       p.Path[:],
			Commitment: p.Commitment[:],
		}
	}

	return &PacketAttestation{
		Height:  height,
		Packets: packets,
	}, nil
}

func (pc *PacketCompact) ABIEncode() ([]byte, error) {
	return tupleArrayType.Pack(bytesToBytes32(pc.Path), bytesToBytes32(pc.Commitment))
}

func ABIDecodeStateAttestation(data []byte) (*StateAttestation, error) {
	unpacked, err := stateAttestationArgs.Unpack(data)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrInvalidAttestationData, "failed to ABI decode state attestation: %v", err)
	}

	if len(unpacked) != 2 {
		return nil, errorsmod.Wrap(ErrInvalidAttestationData, "invalid state attestation: expected 2 fields")
	}

	height, ok := unpacked[0].(uint64)
	if !ok {
		return nil, errorsmod.Wrap(ErrInvalidAttestationData, "invalid height type")
	}

	timestampSeconds, ok := unpacked[1].(uint64)
	if !ok {
		return nil, errorsmod.Wrap(ErrInvalidAttestationData, "invalid timestamp type")
	}

	return &StateAttestation{
		Height:    height,
		Timestamp: timestampSeconds * nanosPerSecond,
	}, nil
}

func bytesToBytes32(b []byte) [32]byte {
	var result [32]byte
	copy(result[:], b)
	return result
}

func Uint64ToPaddedBytes(v uint64) []byte {
	return common.LeftPadBytes(new(big.Int).SetUint64(v).Bytes(), 32)
}
