package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// StakingKeeper defines the contract expected by parent-chain ccv module.
// StakingKeeper is responsible for keeping track of latest validator set of all baby chains
type StakingKeeper interface {
	GetValidatorSetChanges(chainID string) []abci.ValidatorUpdate
	// This method is not required by CCV module explicitly but necessary for init protocol
	GetInitialValidatorSet(chainID string) []sdk.Tx
}

// RegistryKeeper defines the contract expected by parent-chain ccv module from a Registry Module that will keep track
// of chain creators and respective validator sets
// RegistryKeeper is responsible for verifying that chain creator is authorized to create a chain with given chain-id,
// as well as which validators are staking for a given chain.
type CNSKeeper interface {
	AuthorizeChainCreator(chainID, creator string)
	GetValidatorSet(chainID string) []sdk.ValAddress
}

// TODO: Expected interfaces for distribution on parent and baby chains
