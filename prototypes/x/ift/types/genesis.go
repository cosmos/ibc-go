package types

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		Bridges:          []GenesisBridge{},
		PendingTransfers: []PendingTransfer{},
	}
}

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		Authority: "", // Should be set to gov module address during init
	}
}

// Validate performs basic genesis state validation
func (gs GenesisState) Validate() error {
	for _, bridge := range gs.Bridges {
		if bridge.Denom == "" {
			return ErrInvalidDenom
		}
		if bridge.Bridge.ClientId == "" {
			return ErrInvalidClientID
		}
	}

	for _, pending := range gs.PendingTransfers {
		if pending.Denom == "" {
			return ErrInvalidDenom
		}
		if pending.ClientId == "" {
			return ErrInvalidClientID
		}
	}

	return nil
}
