package cosmosevm

import "github.com/cosmos/ibc-go/v10/modules/core/exported"

var _ exported.ConsensusState = (*ConsensusState)(nil)

// NewConsensusState creates a new ConsensusState instance.
func NewConsensusState() *ConsensusState {
	return &ConsensusState{}
}

// ClientType implements the exported.ConsensusState interface.
func (ConsensusState) ClientType() string {
	return exported.CosmosEvm
}

// GetTimestamp implements the exported.ConsensusState interface and is deprecated.
func (cs ConsensusState) GetTimestamp() uint64 {
	panic("unreachable: deprecated function")
}

// ValidateBasic implements the exported.ConsensusState interface.
func (cs ConsensusState) ValidateBasic() error {
	return nil
}
