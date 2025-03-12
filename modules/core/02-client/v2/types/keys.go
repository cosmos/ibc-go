package types

const (
	// SubModuleName defines the IBC client name
	SubModuleName string = "clientv2"
	// KeyCounterparty is the key for the counterpartyInfo in the client-specific store
	KeyCounterparty = "counterparty"
	// KeyV2Params is the key for the v2 params of the client
	// NOTE: v1 params were global parameters, whereas this is the parameters per clientID
	KeyV2Params = "v2params"
)

// CounterpartyKey returns the key under which the counterparty is stored in the client store
func CounterpartyKey() []byte {
	return []byte(KeyCounterparty)
}

// V2ParamsKey returns the key under which the v2 parameters are stored in the client store
func V2ParamsKey() []byte {
	return []byte(KeyV2Params)
}
