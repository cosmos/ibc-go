package types

import (
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// NewGenesisState creates a 29-fee GenesisState instance.
func NewGenesisState(identifiedFees []*IdentifiedPacketFee) *GenesisState {
	return &GenesisState{
		PortId:         PortID,
		IdentifiedFees: identifiedFees,
	}
}

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PortId:         PortID,
		IdentifiedFees: []*IdentifiedPacketFee{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}
	return nil
}
