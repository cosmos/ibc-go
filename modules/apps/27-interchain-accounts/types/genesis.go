package types

// DefaultGenesis creates and returns the default interchain accounts GenesisState
func DefaultControllerGenesis() *ControllerGenesisState {
	return &ControllerGenesisState{}
}

// NewControllerGenesisState creates a returns a new ControllerGenesisState instance
func NewControllerGenesisState(channels []*ActiveChannel, accounts []*RegisteredInterchainAccount, ports []string) *ControllerGenesisState {
	return &ControllerGenesisState{
		ActiveChannels:     channels,
		InterchainAccounts: accounts,
		Ports:              ports,
	}
}

// DefaultHostGenesis creates and returns the default interchain accounts GenesisState
func DefaultHostGenesis() *HostGenesisState {
	return &HostGenesisState{
		Port: PortID,
	}
}

// NewHostGenesisState creates a returns a new ControllerGenesisState instance
func NewHostGenesisState(channels []*ActiveChannel, accounts []*RegisteredInterchainAccount, port string) *HostGenesisState {
	return &HostGenesisState{
		ActiveChannels:     channels,
		InterchainAccounts: accounts,
		Port:               port,
	}
}
