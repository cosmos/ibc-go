package types

import (
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

var _ exported.Header = &DuplicateHeaderHeader{}

// FrozenHeight is the same for all DuplicateHeaderHeader.
var FrozenHeight = clienttypes.NewHeight(0, 1)

// NewDuplicateHeaderHeader creates a new DuplicateHeaderHeader instance.
func NewDuplicateHeaderHeader(clientID string, header1, header2 *Header) *DuplicateHeaderHeader {
	return &DuplicateHeaderHeader{
		ClientId: clientID,
		Header1:  header1,
		Header2:  header2,
	}
}

// ClientType is Tendermint light client.
func (h DuplicateHeaderHeader) ClientType() string {
	return exported.Tendermint
}

// GetClientID returns the ID of the client that committed a DuplicateHeaderHeader.
func (h DuplicateHeaderHeader) GetClientID() string {
	return h.ClientId
}

// GetHeight is added for compatibility between PRs.
// TODO: Remove after GetHeight() is removed from Header interface.
func (h DuplicateHeaderHeader) GetHeight() exported.Height {
	return nil
}

// GetTime returns the timestamp at which DuplicateHeaderHeader occurred. It uses the
// maximum value from both headers to prevent producing an invalid header outside
// of the DuplicateHeaderHeader age range.
func (h DuplicateHeaderHeader) GetTime() time.Time {
	t1, t2 := h.Header1.GetTime(), h.Header2.GetTime()
	if t1.After(t2) {
		return t1
	}
	return t2
}

// ValidateBasic implements DuplicateHeaderHeader interface.
func (h DuplicateHeaderHeader) ValidateBasic() error {
	if h.Header1 == nil {
		return sdkerrors.Wrap(ErrInvalidHeader, "DuplicateHeaderHeader Header1 cannot be nil")
	}
	if h.Header2 == nil {
		return sdkerrors.Wrap(ErrInvalidHeader, "DuplicateHeaderHeader Header2 cannot be nil")
	}
	if h.Header1.TrustedHeight.RevisionHeight == 0 {
		return sdkerrors.Wrapf(ErrInvalidHeaderHeight, "DuplicateHeaderHeader Header1 cannot have zero revision height")
	}
	if h.Header2.TrustedHeight.RevisionHeight == 0 {
		return sdkerrors.Wrapf(ErrInvalidHeaderHeight, "DuplicateHeaderHeader Header2 cannot have zero revision height")
	}
	if h.Header1.TrustedValidators == nil {
		return sdkerrors.Wrap(ErrInvalidValidatorSet, "trusted validator set in Header1 cannot be empty")
	}
	if h.Header2.TrustedValidators == nil {
		return sdkerrors.Wrap(ErrInvalidValidatorSet, "trusted validator set in Header2 cannot be empty")
	}
	if h.Header1.Header.ChainID != h.Header2.Header.ChainID {
		return sdkerrors.Wrap(clienttypes.ErrInvalidMisbehaviour, "headers must have identical chainIDs")
	}

	if err := host.ClientIdentifierValidator(h.ClientId); err != nil {
		return sdkerrors.Wrap(err, "DuplicateHeaderHeader client ID is invalid")
	}

	// ValidateBasic on both validators.
	if err := h.Header1.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(
			clienttypes.ErrInvalidMisbehaviour,
			sdkerrors.Wrap(err, "header 1 failed validation").Error(),
		)
	}
	if err := h.Header2.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(
			clienttypes.ErrInvalidMisbehaviour,
			sdkerrors.Wrap(err, "header 2 failed validation").Error(),
		)
	}
	// Ensure that Height1 is greater than or equal to Height2.
	if h.Header1.GetHeight().LT(h.Header2.GetHeight()) {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidMisbehaviour, "Header1 height is less than Header2 height (%s < %s)", h.Header1.GetHeight(), h.Header2.GetHeight())
	}

	blockID1, err := tmtypes.BlockIDFromProto(&h.Header1.SignedHeader.Commit.BlockID)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid block ID from header 1 in DuplicateHeaderHeader")
	}
	blockID2, err := tmtypes.BlockIDFromProto(&h.Header2.SignedHeader.Commit.BlockID)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid block ID from header 2 in DuplicateHeaderHeader")
	}

	if err := validCommit(h.Header1.Header.ChainID, *blockID1,
		h.Header1.Commit, h.Header1.ValidatorSet); err != nil {
		return err
	}
	if err := validCommit(h.Header2.Header.ChainID, *blockID2,
		h.Header2.Commit, h.Header2.ValidatorSet); err != nil {
		return err
	}
	return nil
}

// validCommit checks if the given commit is a valid commit from the passed-in validatorset.
func validCommit(chainID string, blockID tmtypes.BlockID, commit *tmproto.Commit, valSet *tmproto.ValidatorSet) (err error) {
	tmCommit, err := tmtypes.CommitFromProto(commit)
	if err != nil {
		return sdkerrors.Wrap(err, "commit is not tendermint commit type")
	}
	tmValset, err := tmtypes.ValidatorSetFromProto(valSet)
	if err != nil {
		return sdkerrors.Wrap(err, "validator set is not tendermint validator set type")
	}

	if err := tmValset.VerifyCommitLight(chainID, blockID, tmCommit.Height, tmCommit); err != nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidMisbehaviour, "validator set did not commit to header")
	}

	return nil
}
