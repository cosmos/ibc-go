package types

import (
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// DefaultGenesis creates and returns the default interchain query GenesisState
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		HostPort: PortID,
		Params:   DefaultParams(),
	}
}

// NewHostGenesisState creates a returns a new GenesisState instance
func NewHostGenesisState(hostPort string, params Params) *GenesisState {
	return &GenesisState{
		HostPort: hostPort,
		Params:   params,
	}
}

// Validate performs basic validation of the GenesisState
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.HostPort); err != nil {
		return err
	}

	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return nil
}
