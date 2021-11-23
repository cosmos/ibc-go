package types

// NewGenesisState creates a 29-fee GenesisState instance.
func NewGenesisState(identifiedFees []*IdentifiedPacketFee) *GenesisState {
	return &GenesisState{
		IdentifiedFees: identifiedFees,
	}
}

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		IdentifiedFees: []*IdentifiedPacketFee{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return nil
}
