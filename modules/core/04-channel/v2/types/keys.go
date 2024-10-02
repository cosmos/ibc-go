package types

const (
	// SubModuleName defines the channelv2 module name.
	SubModuleName = "channelv2"

	// CounterpartyKey is the key used to store counterparty in the client store.
	// the counterparty key is imported from types instead of host because
	// the counterparty key is not a part of the ics-24 host specification
	CounterpartyKey = "counterparty"
)
