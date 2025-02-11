package types

const (
	// SubModuleName defines the IBC client name
	SubModuleName string = "clientv2"
	// KeyCounterparty is the key for the counterpartyInfo in the client-specific store
	KeyCounterparty = "counterparty"
)

// CounterpartyKey returns the key under which the counterparty is stored in the client store
func CounterpartyKey() []byte {
	return []byte(KeyCounterparty)
}
