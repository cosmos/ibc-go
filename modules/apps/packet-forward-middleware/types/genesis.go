package types

import "errors"

// DefaultGenesisState returns a GenesisState with an empty map of in-flight packets.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		InFlightPackets: make(map[string]InFlightPacket),
	}
}

// Validate performs basic genesis state validation returning an error upon any failure.
func (gs GenesisState) Validate() error {
	if gs.InFlightPackets == nil {
		return errors.New("in-flight packets cannot be nil")
	}

	return nil
}
