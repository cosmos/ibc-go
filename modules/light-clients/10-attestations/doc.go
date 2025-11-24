/*
Package attestations implements a concrete LightClientModule, ClientState, ConsensusState,
and AttestationProof types for the Attestor light client.
This implementation is based on the ATTESTOR_SPEC.md specification.

Note that client identifiers are expected to be in the form: 10-attestations-{N}.
Client identifiers are generated and validated by core IBC, unexpected client identifiers will result in errors.
*/
package attestations


