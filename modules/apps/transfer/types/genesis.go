package types

import (
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// NewGenesisState creates a new ibc-transfer GenesisState instance.
func NewGenesisState(portIDs []string, denomTraces Traces, params Params) *GenesisState {
	return &GenesisState{
		PortIds:     portIDs,
		DenomTraces: denomTraces,
		Params:      params,
	}
}

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PortIds:     []string{PortID, FeePortID},
		DenomTraces: Traces{},
		Params:      DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	for _, port := range gs.PortIds {
		if err := host.PortIdentifierValidator(port); err != nil {
			return err
		}
	}
	if err := gs.DenomTraces.Validate(); err != nil {
		return err
	}
	return gs.Params.Validate()
}
