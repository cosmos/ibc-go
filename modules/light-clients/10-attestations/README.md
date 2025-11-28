# 10-attestations Light Client

An attestor-based IBC light client that verifies state using quorum-signed ECDSA attestations from a fixed set of trusted signers.

## Trust Model

- A fixed set of ECDSA attestors (EOA addresses) is configured at client creation
- Updates require `minRequiredSigs` unique signatures from the attestor set

## State

**Client State**
- `attestorAddresses` — trusted attestor set
- `minRequiredSigs` — quorum threshold
- `latestHeight` — highest trusted height
- `isFrozen` — halts all operations when true

**Consensus State** (per height)
- `timestamp` — trusted UNIX timestamp in nanoseconds

## Proofs

All proofs contain an `AttestationProof` with:
- `attestationData` — the attested payload (encoded `StateAttestation` or `PacketAttestation`)
- `signatures` — 65-byte ECDSA signatures over `sha256(attestationData)`

## Limitations

- No client recovery supported
- No client upgrades supported
- No attestor updates or rotation supported
