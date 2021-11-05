package types

// DefaultGenesis creates and returns the default interchain accounts GenesisState
// The default GenesisState includes the standard port identifier to which all host chains must bind
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Ports: []string{PortID},
	}
}

// NewGenesisState creates a returns a new GenesisState instance
func NewGenesisState(ports []string, channels []*ActiveChannel, accounts []*RegisteredInterchainAccount) *GenesisState {
	return &GenesisState{
		ActiveChannels:     channels,
		InterchainAccounts: accounts,
		Ports:              ports,
	}
}
