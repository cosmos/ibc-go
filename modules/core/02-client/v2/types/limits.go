package types

const (
	// MaxCounterpartyMerklePrefixParts bounds the number of prefix parts which may be persisted
	// for a single client v2 counterparty entry.
	MaxCounterpartyMerklePrefixParts = 32

	// MaxCounterpartyMerklePrefixPartLength bounds the maximum byte length of a single prefix part.
	MaxCounterpartyMerklePrefixPartLength = 256

	// MaxCounterpartyMerklePrefixTotalLength bounds the total bytes across all prefix parts.
	MaxCounterpartyMerklePrefixTotalLength = 2048
)
