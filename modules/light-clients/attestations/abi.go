package attestations

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

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

	packetAttestationType, _ = abi.NewType("tuple", "PacketAttestation", []abi.ArgumentMarshaling{
		{Name: "height", Type: "uint64"},
		{Name: "packets", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
			{Name: "path", Type: "bytes32"},
			{Name: "commitment", Type: "bytes32"},
		}},
	})

	packetAttestationArgs = abi.Arguments{
		{Name: "attestation", Type: packetAttestationType},
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

// ABIPacketAttestation is the ABI-compatible representation for tuple-wrapped encoding.
type ABIPacketAttestation struct {
	Height  uint64
	Packets []ABIPacketCompact
}

func (pa *PacketAttestation) ABIEncode() ([]byte, error) {
	packets := make([]ABIPacketCompact, len(pa.Packets))
	for i, p := range pa.Packets {
		packets[i] = ABIPacketCompact{
			Path:       bytesToBytes32(p.Path),
			Commitment: bytesToBytes32(p.Commitment),
		}
	}
	// Pack as tuple-wrapped struct to match Solidity's abi.encode(PacketAttestation)
	abiAttestation := ABIPacketAttestation{
		Height:  pa.Height,
		Packets: packets,
	}
	return packetAttestationArgs.Pack(abiAttestation)
}

func ABIDecodePacketAttestation(data []byte) (*PacketAttestation, error) {
	unpacked, err := packetAttestationArgs.Unpack(data)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrInvalidAttestationData, "failed to ABI decode packet attestation: %v", err)
	}

	// Tuple-wrapped format: single element containing the struct
	if len(unpacked) != 1 {
		return nil, errorsmod.Wrap(ErrInvalidAttestationData, "invalid packet attestation: expected 1 tuple element")
	}

	//nolint:revive // go-ethereum returns anonymous struct, cannot use named type
	abiAttestation, ok := unpacked[0].(struct {
		Height  uint64 `json:"height"`
		Packets []struct {
			Path       [32]byte `json:"path"`
			Commitment [32]byte `json:"commitment"`
		} `json:"packets"`
	})
	if !ok {
		return nil, errorsmod.Wrapf(ErrInvalidAttestationData, "invalid packet attestation type, got %T", unpacked[0])
	}

	packets := make([]PacketCompact, len(abiAttestation.Packets))
	for i, p := range abiAttestation.Packets {
		packets[i] = PacketCompact{
			Path:       p.Path[:],
			Commitment: p.Commitment[:],
		}
	}

	return &PacketAttestation{
		Height:  abiAttestation.Height,
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
