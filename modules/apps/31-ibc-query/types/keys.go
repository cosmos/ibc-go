package types

import fmt "fmt"

const (
	// ModuleName defines the 31-ibc-query name
	ModuleName = "queryibc"

	// StoreKey is the store key string for IBC query module
	StoreKey = ModuleName

	// RouterKey is the message route for IBC query module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC query module
	QuerierRoute = ModuleName

	KeyNextQuerySequence = "nextQuerySequence"
	QueryPrefix = "query-"
)

var (
	// QueryKey defines the key to store the query in store
	QueryKey = []byte{0x01}
	// QueryResultKey defines the key to store query result in store
	QueryResultKey = []byte{0x02}
)

func FormatQueryIdentifier(sequence uint64) string {
	return fmt.Sprintf("%s%d", QueryPrefix, sequence)
}