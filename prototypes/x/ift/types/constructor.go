package types

import (
	"encoding/json"
	"strings"
)

const (
	ConstructorEVM    = "evm"
	ConstructorCosmos = "cosmos"
	ConstructorSolana = "solana"
)

// SolanaOptions represents the Solana-specific options in the constructor string.
// Format: {"solana":{"gmp_program_id":"...","mint_pubkey":"..."}}
type SolanaOptions struct {
	GMPProgramID string `json:"gmp_program_id"`
	MintPubKey   string `json:"mint_pubkey"` // SPL token mint public key
}

// ValidateCounterpartyAddress validates the counterparty IFT contract address format
// based on the constructor type.
func ValidateCounterpartyAddress(constructorType, address string) error {
	baseType := ParseConstructorType(constructorType)
	switch baseType {
	case ConstructorEVM:
		return EVMConstructor{}.ValidateCounterpartyAddress(address)
	case ConstructorCosmos:
		return CosmosTxConstructor{}.ValidateCounterpartyAddress(address)
	case ConstructorSolana:
		return ValidateSolanaAddress(address)
	default:
		return ErrInvalidConstructorType.Wrapf("unknown: %s", constructorType)
	}
}

// ParseConstructorType extracts the constructor type from the constructor string.
// For JSON format (e.g., {"solana":{...}}), returns the top-level key.
// For plain strings (e.g., "evm", "cosmos"), returns the string as-is.
func ParseConstructorType(constructorStr string) string {
	if strings.HasPrefix(constructorStr, "{") {
		var wrapper map[string]json.RawMessage
		if json.Unmarshal([]byte(constructorStr), &wrapper) == nil {
			for key := range wrapper {
				return key
			}
		}
	}
	return constructorStr
}

// ParseSolanaConfig extracts the Solana options from a constructor string.
// Expects JSON format: {"solana":{"gmp_program_id":"...","mint_pubkey":"..."}}
// Returns ErrNotSolanaConstructor if the constructor is not a Solana constructor.
func ParseSolanaConfig(constructorStr string) (*SolanaOptions, error) {
	var wrapper map[string]*SolanaOptions
	if err := json.Unmarshal([]byte(constructorStr), &wrapper); err != nil {
		return nil, ErrInvalidConstructorType.Wrapf("invalid JSON: %s", err)
	}

	cfg, ok := wrapper[ConstructorSolana]
	if !ok {
		return nil, ErrNotSolanaConstructor
	}

	return cfg, nil
}

// ValidConstructorTypes returns all valid constructor type identifiers
func ValidConstructorTypes() []string {
	return []string{ConstructorEVM, ConstructorCosmos, ConstructorSolana}
}

// ValidateConstructorString validates a constructor string.
// For EVM and Cosmos constructors, it just validates the type is known.
// For Solana constructors, it also validates the embedded configuration.
func ValidateConstructorString(constructorStr string) error {
	constructorType := ParseConstructorType(constructorStr)
	switch constructorType {
	case ConstructorEVM, ConstructorCosmos:
		return nil
	case ConstructorSolana:
		_, err := ValidateSolanaConstructorString(constructorStr)
		return err
	default:
		return ErrInvalidConstructorType.Wrapf("unknown: %s", constructorType)
	}
}

// ValidateSolanaConstructorString validates and parses a Solana constructor string.
// Returns the parsed config or error if invalid.
func ValidateSolanaConstructorString(constructorStr string) (*SolanaOptions, error) {
	cfg, err := ParseSolanaConfig(constructorStr)
	if err != nil {
		return nil, err
	}

	if err := ValidateSolanaAddress(cfg.GMPProgramID); err != nil {
		return nil, ErrInvalidSolanaAddress.Wrapf("invalid gmp_program_id: %s", err)
	}
	if err := ValidateSolanaAddress(cfg.MintPubKey); err != nil {
		return nil, ErrInvalidSolanaAddress.Wrapf("invalid mint_pubkey: %s", err)
	}

	return cfg, nil
}
