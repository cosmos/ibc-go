package types

const (
	// SubModuleName defines the IBC client name
	SubModuleName string = "clientv2"
	// KeyCounterparty is the key for the counterpartyInfo in the client-specific store
	KeyCounterparty = "counterparty"
	// KeyConfig is the key for the v2 configuration of the client
	// NOTE: v1 params were global parameters, whereas this is a configuration per client
	KeyConfig = "config"
)

// CounterpartyKey returns the key under which the counterparty is stored in the client store
func CounterpartyKey() []byte {
	return []byte(KeyCounterparty)
}

// ConfigKey returns the key under which the v2 configuration are stored in the client store
func ConfigKey() []byte {
	return []byte(KeyConfig)
}
