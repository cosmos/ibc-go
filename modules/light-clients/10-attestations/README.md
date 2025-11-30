# 10-attestations Light Client

An attestor-based IBC light client that verifies state using quorum-signed ECDSA attestations from a fixed set of trusted signers.

## Overview

The attestations light client provides a trust model based on a quorum of trusted attestors rather than cryptographic verification of block headers. This design suits scenarios where a set of known, trusted parties can attest to the state of a counterparty chain.

## Wire Compatibility

The attestation data format is ABI-encoded for wire compatibility with:
- Solidity attestor light client (`solidity-ibc-eureka`)
- CosmWasm attestor light client (`cw-ics08-wasm-attestor`)

This allows the same attestor infrastructure to generate proofs that work across all three platforms (Cosmos SDK, EVM, CosmWasm).

## Trust Model

- A fixed set of ECDSA attestors (Ethereum-style EOA addresses) is configured at client creation
- Updates and proofs require `minRequiredSigs` unique valid signatures from the attestor set
- Signatures are standard 65-byte ECDSA signatures (`r || s || v`) over `sha256(abiEncodedAttestationData)`
- Each signer can only sign once per proof (duplicates are rejected)
- Only addresses in the attestor set are accepted (unknown signers are rejected)

## State

### Client State

| Field               | Type       | Description                                       |
|---------------------|------------|---------------------------------------------------|
| `attestorAddresses` | `[]string` | Fixed set of trusted attestor EOA addresses       |
| `minRequiredSigs`   | `uint32`   | Minimum unique signatures required (quorum)       |
| `latestHeight`      | `uint64`   | Highest trusted height (revision number is 0)     |
| `isFrozen`          | `bool`     | When true, all operations are halted              |

### Consensus State

Stored per height with the following field:

| Field       | Type     | Description                                                   |
|-------------|----------|---------------------------------------------------------------|
| `timestamp` | `uint64` | Trusted UNIX timestamp (stored in nanoseconds internally)     |

## Client Updates

Client updates use `AttestationProof` as the client message, containing an ABI-encoded `StateAttestation`:

```solidity
// ABI-encoded StateAttestation
struct StateAttestation {
    uint64 height;     // consensus state height
    uint64 timestamp;  // timestamp in seconds
}
// Encoding: abi.encode(height, timestamp) = 64 bytes (two padded uint64s)
```

When a valid update is received:
1. Signatures are verified against the attestor set
2. A new consensus state is created at the specified height
3. `latestHeight` is updated if the new height exceeds it

Updates can also set consensus states for heights lower than or equal to `latestHeight`, enabling flexible state attestation.

**Note**: Timestamps in ABI encoding use seconds for compatibility with Solidity. They are converted to nanoseconds internally.

## Proof Verification

Both membership and non-membership proofs use `AttestationProof` containing an ABI-encoded `PacketAttestation`:

```solidity
// ABI-encoded PacketAttestation
struct PacketCompact {
    bytes32 path;       // keccak256-hashed path
    bytes32 commitment; // commitment value
}

struct PacketAttestation {
    uint64 height;
    PacketCompact[] packets;
}
// Encoding: abi.encode(height, packets) with dynamic array
```

### Path and Value Normalization

All paths and values are normalized to 32 bytes using `keccak256` hashing:
- If already 32 bytes: used as-is (assumed pre-hashed)
- Otherwise: `keccak256(data)` is computed

This ensures consistent verification across implementations (ibc-go, Solidity, CosmWasm).

**Note**: The `keccak256` function is used (matching the Solidity implementation) for cross-platform compatibility.

### Membership Verification

Verifies that a value exists at a given path:
1. Validates the proof has sufficient valid signatures
2. Confirms a consensus state exists for the claimed height
3. Matches the path and commitment in the attested packets
4. Paths are keccak256-hashed if not already 32 bytes
5. Values longer than 32 bytes are keccak256-hashed before comparison

### Non-Membership Verification

Verifies that a path has no value (was deleted or never existed):
1. Validates the proof has sufficient valid signatures
2. Confirms a consensus state exists for the claimed height
3. Finds the path in attested packets with a zero commitment (32 zero bytes)

## Signature Verification

The signature verification process:
1. Computes `sha256(abiEncodedAttestationData)` as the message hash
2. Recovers the signer address from each 65-byte ECDSA signature
3. Verifies each signer is in the attestor set
4. Ensures no duplicate signers
5. Confirms the quorum threshold (`minRequiredSigs`) is met

## Client Status

| Status    | Condition                         |
|-----------|-----------------------------------|
| `Active`  | `isFrozen` is false               |
| `Frozen`  | `isFrozen` is true                |
| `Unknown` | Client state not found            |

## Limitations

- **No client recovery**: `RecoverClient` is not supported
- **No client upgrades**: `VerifyUpgradeAndUpdateState` returns an error
- **No attestor rotation**: The attestor set is fixed at client creation
- **No misbehaviour handling**: `CheckForMisbehaviour` and `UpdateStateOnMisbehaviour` are not implemented
- **Revision number is always 0**: Heights use only the revision height component
- **ABI encoding required**: Attestation data must be ABI-encoded (not Protobuf)
