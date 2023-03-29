package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

// NewGenesisState creates a new ibc-transfer GenesisState instance.
func NewGenesisState(portID string, denomTraces Traces, params Params, denomEscrows sdk.Coins) *GenesisState {
	return &GenesisState{
		PortId:       portID,
		DenomTraces:  denomTraces,
		Params:       params,
		DenomEscrows: denomEscrows,
	}
}

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PortId:       PortID,
		DenomTraces:  Traces{},
		Params:       DefaultParams(),
		DenomEscrows: sdk.Coins{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}
	if err := gs.DenomTraces.Validate(); err != nil {
		return err
	}
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	return gs.DenomEscrows.Validate()
}
