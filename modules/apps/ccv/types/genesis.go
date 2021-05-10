package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
)

// NewInitialChildGenesisState returns a child GenesisState for a completely new child chain.
// TODO: Include chain status
func NewInitialChildGenesisState(chainID string, cs *ibctmtypes.ClientState, consState *ibctmtypes.ConsensusState) ChildGenesisState {
	return ChildGenesisState{
		ParentChainId:        chainID,
		NewChain:             true,
		ParentClientState:    cs,
		ParentConsensusState: consState,
	}
}

// NewRestartChildGenesisState returns a child GenesisState that has already been established.
func NewRestartChildGenesisState(chainID, channelID string, unbondingSequences []*UnbondingSequence) ChildGenesisState {
	return ChildGenesisState{
		ParentChainId:      chainID,
		ParentChannelId:    channelID,
		UnbondingSequences: unbondingSequences,
		NewChain:           false,
	}
}

// DefaultGenesisState returns a default new child chain genesis state with blank clientstate and consensus states for testing.
func DefaultChildGenesisState() ChildGenesisState {
	return ChildGenesisState{
		ParentChainId:        "testparentchainid",
		NewChain:             true,
		ParentClientState:    &ibctmtypes.ClientState{},
		ParentConsensusState: &ibctmtypes.ConsensusState{},
	}
}

// Validate performs basic genesis state validation returning an error upon any failure.
// TODO: Validate UnbondingSequences
func (gs ChildGenesisState) Validate() error {
	if gs.ParentChainId == "" {
		return sdkerrors.Wrap(ErrInvalidGenesis, "parent chain id cannot be blank")
	}
	if gs.NewChain {
		if gs.ParentClientState == nil {
			return sdkerrors.Wrap(ErrInvalidGenesis, "parent client state cannot be nil for new chain")
		}
		if err := gs.ParentClientState.Validate(); err != nil {
			return sdkerrors.Wrapf(ErrInvalidGenesis, "parent client state cannot be nil for new chain %s", err.Error())
		}
		if gs.ParentConsensusState == nil {
			return sdkerrors.Wrap(ErrInvalidGenesis, "parent consensus state cannot be nil for new chain")
		}
		if err := gs.ParentConsensusState.ValidateBasic(); err != nil {
			return sdkerrors.Wrapf(ErrInvalidGenesis, "parent client state cannot be nil for new chain %s", err.Error())
		}
		if gs.ParentChannelId != "" {
			return sdkerrors.Wrap(ErrInvalidGenesis, "parent channel id cannot be set for new chain. must be established on handshake")
		}
	} else {
		if gs.ParentChannelId == "" {
			return sdkerrors.Wrap(ErrInvalidGenesis, "parent channel id must be set for a restarting child genesis state")
		}
		if gs.ParentClientState != nil || gs.ParentConsensusState != nil {
			return sdkerrors.Wrap(ErrInvalidGenesis, "parent client state and consensus states must be nil for a restarting genesis state")
		}
	}
	return nil
}
