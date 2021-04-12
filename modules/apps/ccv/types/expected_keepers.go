package types

// StakingKeeper defines the contract expected by parent-chain ccv module.
// StakingKeeper is responsible for keeping track of latest validator set of all baby chains
type StakingKeeper interface {
	GetValidatorSetChanges(chainID string) []abci.ValidatorUpdate
	// This method is not required by CCV module explicitly but necessary for init protocol
	GetInitialValidatorSet(chainID string) []sdk.Tx
}

// CNSKeeper defines the contract expected by parent-chain ccv module
// CNSKeeper is responsible for verifying that chain creator is authorized to create a chain with given chain-id
type CNSKeeper interface {
	AuthorizeChainCreator(chainID, creator string)
}

// TODO: Expected interfaces for distribution on parent and baby chains
