package types

const (
	// AckErrorString defines a string constant included in error acknowledgements
	// NOTE: Changing this const is state machine breaking as acknowledgements are written into state
	AckErrorString = "error handling packet on destination chain: see events for details"
)
