package types

import (
	"bytes"
	fmt "fmt"
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmtypes "github.com/tendermint/tendermint/types"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

var _ exported.Header = &Header{}

// ConsensusState returns the updated consensus state associated with the header
func (h Header) ConsensusState() *ConsensusState {
	return &ConsensusState{
		Timestamp: h.GetTime(),
		Root:      commitmenttypes.NewMerkleRoot(h.Header.GetAppHash()),
	}
}

// ClientType defines that the Header is a Dymint rollapp
func (h Header) ClientType() string {
	return exported.Dymint
}

// GetChainID returns the chain-id
func (h Header) GetChainID() string {
	return h.Header.ChainID
}

// GetHeight returns the current height. It returns 0 if the dymint
// header is nil.
// NOTE: the header.Header is checked to be non nil in ValidateBasic.
func (h Header) GetHeight() exported.Height {
	revision := clienttypes.ParseChainID(h.Header.ChainID)
	return clienttypes.NewHeight(revision, uint64(h.Header.Height))
}

// GetTime returns the current block timestamp. It returns a zero time if
// the dymint header is nil.
// NOTE: the header.Header is checked to be non nil in ValidateBasic.
func (h Header) GetTime() time.Time {
	return h.Header.Time
}

// ValidateBasic calls the SignedHeader ValidateBasic function and checks
// that validatorsets are not nil.
// NOTE: TrustedHeight may be empty when creating client
// with MsgCreateClient
func (h Header) ValidateBasic() error {
	if h.SignedHeader == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidHeader, "dymint signed header cannot be nil")
	}
	if h.Header == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidHeader, "dymint header cannot be nil")
	}
	tmSignedHeader, err := tmtypes.SignedHeaderFromProto(h.SignedHeader)
	if err != nil {
		return sdkerrors.Wrap(err, "header is not a dymint header")
	}
	if err := tmSignedHeader.ValidateBasic(h.Header.GetChainID()); err != nil {
		return sdkerrors.Wrap(err, "header failed basic validation")
	}

	// TrustedHeight is less than Header for updates and misbehaviour
	if h.TrustedHeight.GTE(h.GetHeight()) {
		return sdkerrors.Wrapf(ErrInvalidHeaderHeight, "TrustedHeight %d must be less than header height %d",
			h.TrustedHeight, h.GetHeight())
	}

	if h.ValidatorSet == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidHeader, "validator set is nil")
	}
	tmValset, err := tmtypes.ValidatorSetFromProto(h.ValidatorSet)
	if err != nil {
		return sdkerrors.Wrap(err, "validator set is not dymint validator set")
	}
	if !bytes.Equal(h.Header.ValidatorsHash, tmValset.Hash()) {
		return sdkerrors.Wrap(clienttypes.ErrInvalidHeader, "validator set does not match hash")
	}
	return nil
}

// ValidateCommit checks if the given commit is a valid commit from the passed-in validatorset
func (h Header) ValidateCommit() (err error) {
	blockID, err := tmtypes.BlockIDFromProto(&h.SignedHeader.Commit.BlockID)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid block ID from header SignedHeader.Commit")
	}
	tmCommit, err := tmtypes.CommitFromProto(h.Commit)
	if err != nil {
		return sdkerrors.Wrap(err, "commit is not dymint commit type")
	}
	tmValset, err := tmtypes.ValidatorSetFromProto(h.ValidatorSet)
	if err != nil {
		return sdkerrors.Wrap(err, "validator set is not dymint validator set type")
	}

	if tmValset.Size() != len(tmCommit.Signatures) {
		return tmtypes.NewErrInvalidCommitSignatures(tmValset.Size(), len(tmCommit.Signatures))
	}

	if !blockID.Equals(tmCommit.BlockID) {
		return fmt.Errorf("invalid commit -- wrong block ID: want %v, got %v",
			blockID, tmCommit.BlockID)
	}

	// We don't know the validators that committed this block, so we have to
	// check for each vote if its validator is already known.
	valIdx, val := tmValset.GetByAddress(h.Header.ProposerAddress)
	if val != nil {
		commitSig := tmCommit.Signatures[valIdx]
		if !commitSig.ForBlock() {
			return sdkerrors.Wrap(clienttypes.ErrInvalidHeader, "validator set did not commit to header")
		}
		// Validate signature.
		if !bytes.Equal(commitSig.ValidatorAddress, h.Header.ProposerAddress) {
			return fmt.Errorf("wrong proposer address in commit, got %X) but expected %X", valIdx, h.Header.ProposerAddress)
		}
		headerBytes, err := h.SignedHeader.Header.Marshal()
		if err != nil {
			return err
		}
		if !val.PubKey.VerifySignature(headerBytes, commitSig.Signature) {
			return fmt.Errorf("wrong signature (#%d): %X", valIdx, commitSig.Signature)
		}
	} else {
		return fmt.Errorf("proposer is not in the validator set (proposer: %x)", h.Header.ProposerAddress)

	}

	return nil
}
