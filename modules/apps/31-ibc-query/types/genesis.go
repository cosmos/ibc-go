package types

func NewGenesisState(crossChainQueries []*CrossChainQuery) *GenesisState {
	return &GenesisState{
		Queries: crossChainQueries,
	}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Queries: []*CrossChainQuery{},
	}
}

func (gs GenesisState) Validate() error {
	return nil
}
