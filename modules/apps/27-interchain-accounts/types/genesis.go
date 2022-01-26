package types

import (
	controllertypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/host/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// DefaultGenesis creates and returns the interchain accounts GenesisState
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ControllerGenesisState: DefaultControllerGenesis(),
		HostGenesisState:       DefaultHostGenesis(),
	}
}

// NewGenesisState creates and returns a new GenesisState instance from the provided controller and host genesis state types
func NewGenesisState(controllerGenesisState controllertypes.ControllerGenesisState, hostGenesisState hosttypes.HostGenesisState) *GenesisState {
	return &GenesisState{
		ControllerGenesisState: controllerGenesisState,
		HostGenesisState:       hostGenesisState,
	}
}

// Validate performs basic validation of the interchain accounts GenesisState
func (gs GenesisState) Validate() error {
	if err := ValidateControllerGenesis(gs.ControllerGenesisState); err != nil {
		return err
	}

	if err := ValidateHostGenesis(gs.HostGenesisState); err != nil {
		return err
	}

	return nil
}

// DefaultControllerGenesis creates and returns the default interchain accounts ControllerGenesisState
func DefaultControllerGenesis() controllertypes.ControllerGenesisState {
	return controllertypes.ControllerGenesisState{
		Params: controllertypes.DefaultParams(),
	}
}

// NewControllerGenesisState creates a returns a new ControllerGenesisState instance
func NewControllerGenesisState(channels []controllertypes.ActiveChannel, accounts []controllertypes.RegisteredInterchainAccount, ports []string, controllerParams controllertypes.Params) controllertypes.ControllerGenesisState {
	return controllertypes.ControllerGenesisState{
		ActiveChannels:     channels,
		InterchainAccounts: accounts,
		Ports:              ports,
		Params:             controllerParams,
	}
}

// ValidateControllerGenesis performs basic validation of the ControllerGenesisState
func ValidateControllerGenesis(gs controllertypes.ControllerGenesisState) error {
	for _, ch := range gs.ActiveChannels {
		if err := host.ChannelIdentifierValidator(ch.ChannelId); err != nil {
			return err
		}

		if err := host.PortIdentifierValidator(ch.PortId); err != nil {
			return err
		}
	}

	for _, acc := range gs.InterchainAccounts {
		if err := host.PortIdentifierValidator(acc.PortId); err != nil {
			return err
		}

		if err := ValidateAccountAddress(acc.AccountAddress); err != nil {
			return err
		}
	}

	for _, port := range gs.Ports {
		if err := host.PortIdentifierValidator(port); err != nil {
			return err
		}
	}

	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return nil
}

// DefaultHostGenesis creates and returns the default interchain accounts HostGenesisState
func DefaultHostGenesis() hosttypes.HostGenesisState {
	return hosttypes.HostGenesisState{
		Port:   PortID,
		Params: hosttypes.DefaultParams(),
	}
}

// NewHostGenesisState creates a returns a new HostGenesisState instance
func NewHostGenesisState(channels []hosttypes.ActiveChannel, accounts []hosttypes.RegisteredInterchainAccount, port string, hostParams hosttypes.Params) hosttypes.HostGenesisState {
	return hosttypes.HostGenesisState{
		ActiveChannels:     channels,
		InterchainAccounts: accounts,
		Port:               port,
		Params:             hostParams,
	}
}

// ValidateHostGenesis performs basic validation of the HostGenesisState
func ValidateHostGenesis(gs hosttypes.HostGenesisState) error {
	for _, ch := range gs.ActiveChannels {
		if err := host.ChannelIdentifierValidator(ch.ChannelId); err != nil {
			return err
		}

		if err := host.PortIdentifierValidator(ch.PortId); err != nil {
			return err
		}
	}

	for _, acc := range gs.InterchainAccounts {
		if err := host.PortIdentifierValidator(acc.PortId); err != nil {
			return err
		}

		if err := ValidateAccountAddress(acc.AccountAddress); err != nil {
			return err
		}
	}

	if err := host.PortIdentifierValidator(gs.Port); err != nil {
		return err
	}

	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return nil
}
