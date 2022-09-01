package v2

import (
	controllertypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// DefaultGenesis creates and returns the interchain accounts GenesisState
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ControllerGenesisState: DefaultControllerGenesis(),
		HostGenesisState:       genesistypes.DefaultHostGenesis(),
	}
}

// NewGenesisState creates and returns a new GenesisState instance from the provided controller and host genesis state types
func NewGenesisState(controllerGenesisState ControllerGenesisState, hostGenesisState genesistypes.HostGenesisState) *GenesisState {
	return &GenesisState{
		ControllerGenesisState: controllerGenesisState,
		HostGenesisState:       hostGenesisState,
	}
}

// Validate performs basic validation of the interchain accounts GenesisState
func (gs GenesisState) Validate() error {
	if err := gs.ControllerGenesisState.Validate(); err != nil {
		return err
	}

	if err := gs.HostGenesisState.Validate(); err != nil {
		return err
	}

	return nil
}

// DefaultControllerGenesis creates and returns the default interchain accounts ControllerGenesisState
func DefaultControllerGenesis() ControllerGenesisState {
	return ControllerGenesisState{
		Params: controllertypes.DefaultParams(),
	}
}

// NewControllerGenesisState creates a returns a new ControllerGenesisState instance
func NewControllerGenesisState(
	channels []genesistypes.ActiveChannel,
	accounts []genesistypes.RegisteredInterchainAccount,
	ports []string,
	controllerParams controllertypes.Params,
	middlewareEnabledChannels []MiddlewareEnabled,
) ControllerGenesisState {
	return ControllerGenesisState{
		ActiveChannels:     channels,
		InterchainAccounts: accounts,
		Ports:              ports,
		Params:             controllerParams,
		MiddlewareEnabled:  middlewareEnabledChannels,
	}
}

// Validate performs basic validation of the ControllerGenesisState
func (gs ControllerGenesisState) Validate() error {
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

		if err := icatypes.ValidateAccountAddress(acc.AccountAddress); err != nil {
			return err
		}
	}

	for _, port := range gs.Ports {
		if err := host.PortIdentifierValidator(port); err != nil {
			return err
		}
	}

	for _, mw := range gs.MiddlewareEnabled {
		if err := host.PortIdentifierValidator(mw.PortId); err != nil {
			return err
		}

		if err := host.ChannelIdentifierValidator(mw.ChannelId); err != nil {
			return err
		}
	}

	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return nil
}
