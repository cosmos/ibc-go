package types

import (
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// NewGenesisState creates a new ibc nft-transfer GenesisState instance.
func NewGenesisState(portID string, traces Traces) *GenesisState {
	return &GenesisState{
		PortId: portID,
		Traces: traces,
	}
}

// DefaultGenesisState returns a GenesisState with "nft-transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PortId: PortID,
		Traces: Traces{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}
	return gs.Traces.Validate()
}
