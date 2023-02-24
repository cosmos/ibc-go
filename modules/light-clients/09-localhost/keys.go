package localhost

const (
	// ModuleName defines the 09-localhost light client module name
	ModuleName = "09-localhost"
)

// SentinelProof defines the 09-localhost sentinel proof.
// Core IBC disallows submission of empty or nil proofs in messaging.
// This serves as a placeholder value for relayers to leverage as the proof field in various message types.
// Localhost client state verification will fail if the sentintel proof value is not provided.
var SentinelProof = []byte("proof_localhost")
