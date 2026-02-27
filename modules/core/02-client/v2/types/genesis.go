package types

import (
	"errors"
	"fmt"
)

// DefaultGenesisState returns the ibc client submodule's default genesis state.
func DefaultGenesisState() GenesisState {
	return GenesisState{
		CounterpartyInfos: []GenesisCounterpartyInfo{},
	}
}

// Validate checks the CounterpartyInfos added to the genesis for validity.
func (gs GenesisState) Validate() error {
	seenIDs := make(map[string]struct{})

	for _, genInfo := range gs.CounterpartyInfos {
		if len(genInfo.ClientId) == 0 {
			return errors.New("counterparty client id cannot be empty")
		}

		if genInfo.ClientId == genInfo.CounterpartyInfo.ClientId {
			return errors.New("counterparty client id and client id cannot be the same")
		}

		if len(genInfo.CounterpartyInfo.MerklePrefix) == 0 {
			return errors.New("counterparty merkle prefix cannot be empty")
		}
		if err := ValidateCounterpartyMerklePrefix(genInfo.CounterpartyInfo.MerklePrefix); err != nil {
			return err
		}

		if _, ok := seenIDs[genInfo.ClientId]; ok {
			return fmt.Errorf("duplicate counterparty client id %s found", genInfo.ClientId)
		}
		seenIDs[genInfo.ClientId] = struct{}{}
	}

	return nil
}
