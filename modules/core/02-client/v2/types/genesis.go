package types

import "errors"

// DefaultGenesisState returns the ibc client submodule's default genesis state.
func DefaultGenesisState() GenesisState {
	return GenesisState{
		CounterpartyInfos: []CounterpartyInfo{},
	}
}

// Validate checks the CounterpartyInfos added to the genesis for validity.
func (gs GenesisState) Validate() error {
	seenIDs := make(map[string]struct{})

	for _, counterparty := range gs.CounterpartyInfos {
		if len(counterparty.ClientId) == 0 {
			return errors.New("counterparty client id cannot be empty")
		}

		if len(counterparty.MerklePrefix) == 0 {
			return errors.New("counterparty merkle prefix cannot be empty")
		}

		if _, ok := seenIDs[counterparty.ClientId]; ok {
			return errors.New("duplicate counterparty client id %s found")
		}
		seenIDs[counterparty.ClientId] = struct{}{}
	}

	return nil
}
