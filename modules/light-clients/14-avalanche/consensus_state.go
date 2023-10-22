package avalanche

import (
	"time"

	errorsmod "cosmossdk.io/errors"

	"github.com/ava-labs/avalanchego/utils/crypto/bls"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// SentinelRoot is used as a stand-in root value for the consensus state set at the upgrade height
const SentinelRoot = "sentinel_root"

// NewConsensusState creates a new ConsensusState instance.
func NewConsensusState(
	timestamp time.Time, vdrs []*Validator,
	storageRoot, signedStorageRoot, validatorSet, signedValidatorSet []byte,
	signersInput []byte,
) *ConsensusState {
	return &ConsensusState{
		Timestamp:          timestamp,
		StorageRoot:        storageRoot,
		SignedStorageRoot:  signedStorageRoot,
		ValidatorSet:       validatorSet,
		SignedValidatorSet: signedValidatorSet,
		Vdrs:               vdrs,
		SignersInput:       signersInput,
	}
}

// ClientType returns Tendermint
func (ConsensusState) ClientType() string {
	return exported.Avalanche
}

// GetTimestamp returns block time in nanoseconds of the header that created consensus state
func (cs ConsensusState) GetTimestamp() uint64 {
	return uint64(cs.Timestamp.UnixNano())
}

// ValidateBasic defines a basic validation for the tendermint consensus state.
// NOTE: ProcessedTimestamp may be zero if this is an initial consensus state passed in by relayer
// as opposed to a consensus state constructed by the chain.
func (cs ConsensusState) ValidateBasic() error {
	if len(cs.StorageRoot) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "root cannot be empty")
	}
	if len(cs.SignedStorageRoot) != bls.SignatureLen {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "root signature length not equal bls.SignatureLen")
	}
	if len(cs.ValidatorSet) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "validator set cannot be empty")
	}
	if len(cs.SignedValidatorSet) != bls.SignatureLen {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "validator set signature length not equal bls.SignatureLen")
	}
	if len(cs.SignersInput) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "SignersInput cannot be empty")
	}
	if cs.Timestamp.Unix() <= 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "timestamp must be a positive Unix time")
	}
	return nil
}
