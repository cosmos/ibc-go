package types

import (
	controllertypes "github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/host/types"
)

// DefaultGenesis creates and returns the interchain accounts GenesisState
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ControllerGenesisState: DefaultControllerGenesis(),
		HostGenesisState:       DefaultHostGenesis(),
	}
}

// NewGenesisState creates and returns a new GenesisState instance from the provided controller and host genesis state types
func NewGenesisState(controllerGenesisState ControllerGenesisState, hostGenesisState HostGenesisState) *GenesisState {
	return &GenesisState{
		ControllerGenesisState: controllerGenesisState,
		HostGenesisState:       hostGenesisState,
	}
}

// DefaultControllerGenesis creates and returns the default interchain accounts ControllerGenesisState
func DefaultControllerGenesis() ControllerGenesisState {
	return ControllerGenesisState{
		Params: controllertypes.DefaultParams(),
	}
}

// NewControllerGenesisState creates a returns a new ControllerGenesisState instance
func NewControllerGenesisState(channels []ActiveChannel, accounts []RegisteredInterchainAccount, ports []string, controllerParams controllertypes.Params) ControllerGenesisState {
	return ControllerGenesisState{
		ActiveChannels:     channels,
		InterchainAccounts: accounts,
		Ports:              ports,
		Params:             controllerParams,
	}
}

// DefaultHostGenesis creates and returns the default interchain accounts HostGenesisState
func DefaultHostGenesis() HostGenesisState {
	return HostGenesisState{
		Port:   PortID,
		Params: hosttypes.DefaultParams(),
	}
}

// NewHostGenesisState creates a returns a new HostGenesisState instance
func NewHostGenesisState(channels []ActiveChannel, accounts []RegisteredInterchainAccount, port string, hostParams hosttypes.Params) HostGenesisState {
	return HostGenesisState{
		ActiveChannels:     channels,
		InterchainAccounts: accounts,
		Port:               port,
		Params:             hostParams,
	}
}
