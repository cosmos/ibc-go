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

var (
	_ exported.Misbehaviour = &ConflictingHeaders{}

	// Use the same FrozenHeight for all misbehaviour
	FrozenHeight = clienttypes.NewHeight(0, 1)
)

// NewConflictingHeaders creates a new ConflictingHeaders instance.
func NewConflictingHeaders(clientID string, header1, header2 *Header) *ConflictingHeaders {
	return &ConflictingHeaders{
		ClientId: clientID,
		Header1:  header1,
		Header2:  header2,
	}
}

// ClientType is Tendermint light client
func (ch ConflictingHeaders) ClientType() string {
	return exported.Tendermint
}

// GetClientID returns the ID of the client whom the conflicting headers are related to.
func (ch ConflictingHeaders) GetClientID() string {
	return ch.ClientId
}

// GetTime returns the timestamp at which misbehaviour occurred. It uses the
// maximum value from both headers to prevent producing an invalid header outside
// of the misbehaviour age range.
func (ch ConflictingHeaders) GetTime() time.Time {
	t1, t2 := ch.Header1.GetTime(), ch.Header2.GetTime()
	if t1.After(t2) {
		return t1
	}
	return t2
}

// ValidateBasic implements Header interface
func (ch ConflictingHeaders) ValidateBasic() error {
	if ch.Header1 == nil {
		return sdkerrors.Wrap(ErrInvalidHeader, "conflicting headers Header1 cannot be nil")
	}
	if ch.Header2 == nil {
		return sdkerrors.Wrap(ErrInvalidHeader, "conflicting headers Header2 cannot be nil")
	}
	if ch.Header1.TrustedHeight.RevisionHeight == 0 {
		return sdkerrors.Wrapf(ErrInvalidHeaderHeight, "conflicting headers Header1 cannot have zero revision height")
	}
	if ch.Header2.TrustedHeight.RevisionHeight == 0 {
		return sdkerrors.Wrapf(ErrInvalidHeaderHeight, "conflicting headers Header2 cannot have zero revision height")
	}
	if ch.Header1.TrustedValidators == nil {
		return sdkerrors.Wrap(ErrInvalidValidatorSet, "trusted validator set in Header1 cannot be empty")
	}
	if ch.Header2.TrustedValidators == nil {
		return sdkerrors.Wrap(ErrInvalidValidatorSet, "trusted validator set in Header2 cannot be empty")
	}
	if ch.Header1.Header.ChainID != ch.Header2.Header.ChainID {
		return sdkerrors.Wrap(clienttypes.ErrInvalidMisbehaviour, "headers must have identical chainIDs")
	}

	if err := host.ClientIdentifierValidator(ch.ClientId); err != nil {
		return sdkerrors.Wrap(err, "conflicting headers client ID is invalid")
	}

	// ValidateBasic on both validators
	if err := ch.Header1.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(
			clienttypes.ErrInvalidMisbehaviour,
			sdkerrors.Wrap(err, "header 1 failed validation").Error(),
		)
	}
	if err := ch.Header2.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(
			clienttypes.ErrInvalidMisbehaviour,
			sdkerrors.Wrap(err, "header 2 failed validation").Error(),
		)
	}
	// Ensure that Height1 is greater than or equal to Height2
	if ch.Header1.GetHeight().LT(ch.Header2.GetHeight()) {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidMisbehaviour, "Header1 height is less than Header2 height (%s < %s)", ch.Header1.GetHeight(), ch.Header2.GetHeight())
	}

	blockID1, err := tmtypes.BlockIDFromProto(&ch.Header1.SignedHeader.Commit.BlockID)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid block ID from header 1")
	}
	blockID2, err := tmtypes.BlockIDFromProto(&ch.Header2.SignedHeader.Commit.BlockID)
	if err != nil {
		return sdkerrors.Wrap(err, "invalid block ID from header 2")
	}

	if err := validCommit(ch.Header1.Header.ChainID, *blockID1,
		ch.Header1.Commit, ch.Header1.ValidatorSet); err != nil {
		return err
	}
	if err := validCommit(ch.Header2.Header.ChainID, *blockID2,
		ch.Header2.Commit, ch.Header2.ValidatorSet); err != nil {
		return err
	}
	return nil
}

// validCommit checks if the given commit is a valid commit from the passed-in validatorset
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
