# 10-attestations Light Client

## Overview

The attestations light client uses a fixed set of ECDSA attestors (EOA addresses) with a quorum-based signature verification model. It supports:

- Client state updates via attested state attestations
- Packet membership verification via attested packet commitments
- Misbehaviour detection (conflicting timestamps at the same height)
- Frozen state handling

## Key Features

- **Quorum-based verification**: Requires a minimum number of unique attestor signatures
- **ECDSA signature recovery**: Uses Ethereum-style ECDSA signature recovery
- **Per-height consensus states**: Stores trusted timestamps per height
- **Misbehaviour detection**: Freezes client on conflicting timestamps

## Limitations

- Non-membership verification is not supported
- Client recovery is not supported  
- Client upgrades are not supported

## Spec

This document defines a minimal attestor‑based IBC light client suitable for implementation in ibc‑go. It omits any repository‑specific details and focuses on externally observable behavior, state, and message formats.

### 1. Trust model
- A fixed set of ECDSA attestors (EOA addresses) is configured at client creation.
- A quorum parameter `minRequiredSigs` defines the minimum number of **unique** attestor signatures required to accept data.
- Attestor keys and quorum are immutable; key rotation and quorum changes are out of scope.

### 2. Core state
#### Client State
- `attestorAddresses: address[]` — trusted attestor set.
- `minRequiredSigs: uint8` — quorum threshold (1 ≤ threshold ≤ attestor count).
- `latestHeight: uint64` — highest height that has been trusted.
- `isFrozen: bool` — when true, all verification and updates MUST fail.

#### Consensus State (per height)
- `timestamp: uint64` — trusted UNIX timestamp (nanoseconds) for the height.
- Stored in a map keyed by `height`; at least one initial `(height, timestamp)` pair MUST be present at instantiation.

### 3. Proof payloads
All attestor signatures cover `sha256(attestationData)` (no domain prefix). Signatures are 65‑byte `(r||s||v)` and MUST be unique per proof.

- **AttestationProof**
  - `attestationData: bytes`
  - `signatures: bytes[]`

- **StateAttestation** (used by client updates)
  - `height: uint64`
  - `timestamp: uint64`
  - Encoded into `attestationData`.

- **PacketAttestation** (used by membership queries)
  - `height: uint64`
  - `packets: PacketCompact[]`

- **PacketCompact**
  - `path: bytes32`
  - `commitment: bytes32`

### 4. ICS‑02
#### updateClient(updateMsg)
1. Decode `updateMsg` as `AttestationProof`.
2. Verify signatures:
   - Each signature length is 65 bytes.
   - Recover signer from `sha256(attestationData)`; signer MUST belong to `attestorAddresses`.
   - Signers MUST be unique.
   - Count of valid signatures MUST be ≥ `minRequiredSigs`; otherwise reject.
3. Decode `attestationData` as `StateAttestation`.
4. `height` and `timestamp` MUST be non‑zero.
5. If a consensus timestamp already exists for `height`:
   - If it differs from the provided timestamp, set `isFrozen = true` and return `UpdateResult.Misbehaviour`.
   - If it matches, return `UpdateResult.NoOp`.
6. Otherwise store the timestamp for `height` and set `latestHeight = max(latestHeight, height)`.
7. Return `UpdateResult.Update`.

#### verifyMembership(msg)
Inputs: `proof`, `proofHeight.revisionHeight` (revision number is always 0), `value` (expected packet commitment).

1. `value` MUST be non‑empty.
2. A trusted timestamp for `proofHeight` MUST already exist; otherwise reject.
3. Decode `proof` as `AttestationProof` and verify signatures as in `updateClient`.
4. Decode `attestationData` as `PacketAttestation`.
5. `PacketAttestation.height` MUST equal `proofHeight`; `packets` MUST be non‑empty.
6. If any `PacketCompact` has both `path` and `commitment` equal to the requested `path`/`value`, return the trusted timestamp (nanoseconds) for `proofHeight`; otherwise reject (`NotMember`).

#### verifyNonMembership
Not supported in this version; MUST reject.

#### misbehaviour / upgradeClient
Not supported in this version; MUST reject.

#### Frozen state
If `isFrozen` is true, every call to the above entry points MUST reject without altering state.

### 5. Signature validation rules
- Only 65‑byte ECDSA signatures are accepted.
- Duplicate signers invalidate the proof.
- Any signer outside the attestor set invalidates the proof.
- Invalid or malleable signatures MUST be rejected.

### 6. Timestamp rules
- The first trusted timestamp is provided at instantiation.
- For an existing height, conflicting timestamps constitute misbehaviour (client frozen).
- For new heights, no monotonicity requirement beyond what the relayer enforces; however, implementers SHOULD optionally check monotonic increase to detect out‑of‑order or inconsistent data.

### 7. Access control (optional)
- Implementations MAY restrict `updateClient` and verification entry points to a configured submitter role. If access control is enabled, only addresses with that role may invoke state‑changing or proof‑consuming methods; otherwise they are permissionless.

### 8. Error conditions (non‑exhaustive)
- Empty attestor set or threshold invalid at creation.
- Empty signatures; invalid signature length; duplicate signer; unknown signer; signature recovery failure.
- Threshold not met.
- Invalid or zero height/timestamp in `StateAttestation`.
- Missing trusted timestamp for `proofHeight` in membership checks.
- Empty packet list in `PacketAttestation`.
- Height mismatch between `proofHeight` and `PacketAttestation.height`.
- Membership value not found in attested packet commitments.
- Client frozen.

### 9. Relayer guidance
- Always send `updateClient` with a valid `StateAttestation` before submitting membership proofs that rely on newer heights.
- Ensure aggregated signatures reach quorum and are unique.
- Use the same attestor set and quorum that were embedded at client creation; rotation is not allowed.

### 10. Compliance checklist
- Implements ICS‑02 entry points with the behaviors above.
- Enforces quorumed, unique ECDSA signatures over `sha256(attestationData)`.
- Maintains per‑height trusted timestamps and detects conflicting timestamps as misbehaviour.
- Supports packet‑membership verification via attested packet commitment lists.
- Explicitly rejects non‑membership, misbehaviour handling, and upgrades (unless a future version adds support).
