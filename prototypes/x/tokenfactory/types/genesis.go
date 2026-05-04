package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:        DefaultParams(),
		FactoryDenoms: []GenesisDenom{},
	}
}

// DefaultParams returns the default parameters
func DefaultParams() Params {
	return Params{}
}

// ValidateGenesis validates the tokenfactory genesis parameters
func ValidateGenesis(data GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	for _, denom := range data.FactoryDenoms {
		if err := ValidateTokenFactoryDenom(denom.Denom); err != nil {
			return err
		}

		if _, err := sdk.AccAddressFromBech32(denom.AuthorityMetadata.Admin); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the parameters
func (Params) Validate() error {
	return nil
}
