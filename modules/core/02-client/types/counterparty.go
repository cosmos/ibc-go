package types

func NewCounterpartyInfo(merklePrefix [][]byte, clientID string) CounterpartyInfo {
	return CounterpartyInfo{
		MerklePrefix: merklePrefix,
		ClientId:     clientID,
	}
}
