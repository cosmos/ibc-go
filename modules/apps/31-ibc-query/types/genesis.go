package types

func NewGenesisState(queries []*CrossChainQuery, results []*CrossChainQueryResult) *GenesisState {
	return &GenesisState{
		Queries: queries,
		Results: results,
	}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Queries: []*CrossChainQuery{},
		Results: []*CrossChainQueryResult{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
