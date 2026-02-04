package types

import (
	errorsmod "cosmossdk.io/errors"
)

// NewCounterpartyInfo creates a new counterparty info instance from merlePrefix and clientID
func NewCounterpartyInfo(merklePrefix [][]byte, clientID string) CounterpartyInfo {
	return CounterpartyInfo{
		MerklePrefix: merklePrefix,
		ClientId:     clientID,
	}
}

// ValidateCounterpartyMerklePrefix enforces size limits on the counterparty merkle prefix.
// This prevents durable state bloat by persisting unbounded byte arrays under the client store.
func ValidateCounterpartyMerklePrefix(prefix [][]byte) error {
	if len(prefix) == 0 {
		return errorsmod.Wrap(ErrInvalidCounterparty, "counterparty merkle prefix cannot be empty")
	}
	if len(prefix) > MaxCounterpartyMerklePrefixParts {
		return errorsmod.Wrapf(
			ErrInvalidCounterparty,
			"counterparty merkle prefix parts must not exceed %d items",
			MaxCounterpartyMerklePrefixParts,
		)
	}

	total := 0
	for i, part := range prefix {
		if len(part) > MaxCounterpartyMerklePrefixPartLength {
			return errorsmod.Wrapf(
				ErrInvalidCounterparty,
				"counterparty merkle prefix part %d exceeds max length %d bytes",
				i,
				MaxCounterpartyMerklePrefixPartLength,
			)
		}
		total += len(part)
		if total > MaxCounterpartyMerklePrefixTotalLength {
			return errorsmod.Wrapf(
				ErrInvalidCounterparty,
				"counterparty merkle prefix exceeds max total length %d bytes",
				MaxCounterpartyMerklePrefixTotalLength,
			)
		}
	}

	return nil
}
