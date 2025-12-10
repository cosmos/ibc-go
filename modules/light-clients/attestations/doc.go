// Package attestations implements an attestor-based IBC light client that verifies
// IBC packets using quorum-signed ECDSA attestations from a fixed set of trusted
// signers.
//
// # Experimental
//
// This package is EXPERIMENTAL and is not yet stable. It may change
// in backwards-incompatible ways without notice.
//
// The attestations light client provides a trust model based on a quorum of
// trusted attestors rather than cryptographic verification of block headers.
// This design suits scenarios where a set of known, trusted parties can attest
// to IBC packets on a counterparty chain, rather than attesting to the chain's
// state root.
//
// Attestation data is ABI-encoded for wire compatibility with:
//   - Solidity attestor light client (solidity-ibc-eureka)
//   - CosmWasm attestor light client (cw-ics08-wasm-attestor)
//
// The client state tracks:
//   - attestorAddresses: fixed set of ECDSA attestor addresses
//   - minRequiredSigs: quorum threshold
//   - latestHeight: highest trusted height
//   - isFrozen: whether operations are halted
//
// Consensus states are stored per height and contain a trusted timestamp. Proof
// verification relies on quorum-signed attestations over ABI-encoded packet data
// (paths and commitments hashed with keccak256).
//
// Limitations:
//   - No client recovery or upgrades
//   - No attestor rotation
//   - No misbehaviour handling
//   - Revision number is always 0 (only revision height is used).
package attestations
