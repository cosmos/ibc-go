package types

// NewGenesisState creates a 29-fee GenesisState instance.
func NewGenesisState(identifiedFees []*IdentifiedPacketFee, feeEnabledChannels []*FeeEnabledChannel, registeredRelayers []*RegisteredRelayerAddress) *GenesisState {
	return &GenesisState{
		IdentifiedFees:     identifiedFees,
		FeeEnabledChannels: feeEnabledChannels,
		RegisteredRelayers: registeredRelayers,
	}
}

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		IdentifiedFees:     []*IdentifiedPacketFee{},
		FeeEnabledChannels: []*FeeEnabledChannel{},
		RegisteredRelayers: []*RegisteredRelayerAddress{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return nil
}
