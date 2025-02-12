package types

// NewCounterpartyInfo creates a new counterparty info instance from merlePrefix and clientID
func NewCounterpartyInfo(merklePrefix [][]byte, clientID string) CounterpartyInfo {
	return CounterpartyInfo{
		MerklePrefix: merklePrefix,
		ClientId:     clientID,
	}
}
