package attestations

import (
	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ exported.ConsensusState = (*ConsensusState)(nil)

// ClientType returns Attestations type.
func (ConsensusState) ClientType() string {
	return exported.Attestations
}

// GetTimestamp is deprecated and will panic if called.
func (ConsensusState) GetTimestamp() uint64 {
	panic("GetTimestamp is deprecated")
}

// ValidateBasic defines basic validation for the attestations consensus state.
func (cs ConsensusState) ValidateBasic() error {
	if cs.Timestamp == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "timestamp cannot be 0")
	}
	return nil
}
