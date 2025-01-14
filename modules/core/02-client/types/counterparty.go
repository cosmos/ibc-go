package types

func NewCounterpartyInfo(merklePrefix [][]byte, clientId string) CounterpartyInfo {
	return CounterpartyInfo{
		MerklePrefix: merklePrefix,
		ClientId:     clientId,
	}
}
