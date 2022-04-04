package types

import (
	"bytes"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmtypes "github.com/tendermint/tendermint/types"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
)

// verifyMisbehaviour determines whether or not two conflicting
// headers at the same height would have convinced the light client.
//
// NOTE: consensusState1 is the trusted consensus state that corresponds to the TrustedHeight
// of misbehaviour.Header1
// Similarly, consensusState2 is the trusted consensus state that corresponds
// to misbehaviour.Header2
// Misbehaviour sets frozen height to {0, 1} since it is only used as a boolean value (zero or non-zero).
func (cs *ClientState) verifyMisbehaviour(ctx sdk.Context, clientStore sdk.KVStore, cdc codec.BinaryCodec, misbehaviour *Misbehaviour) error {

	// if heights are equal check that this is valid misbehaviour of a fork
	// otherwise if heights are unequal check that this is valid misbehavior of BFT time violation
	if misbehaviour.Header1.GetHeight().EQ(misbehaviour.Header2.GetHeight()) {
		blockID1, err := tmtypes.BlockIDFromProto(&misbehaviour.Header1.SignedHeader.Commit.BlockID)
		if err != nil {
			return sdkerrors.Wrap(err, "invalid block ID from header 1 in misbehaviour")
		}

		blockID2, err := tmtypes.BlockIDFromProto(&misbehaviour.Header2.SignedHeader.Commit.BlockID)
		if err != nil {
			return sdkerrors.Wrap(err, "invalid block ID from header 2 in misbehaviour")
		}

		// Ensure that Commit Hashes are different
		if bytes.Equal(blockID1.Hash, blockID2.Hash) {
			return sdkerrors.Wrap(clienttypes.ErrInvalidMisbehaviour, "headers block hashes are equal")
		}

	} else {
		// Header1 is at greater height than Header2, therefore Header1 time must be less than or equal to
		// Header2 time in order to be valid misbehaviour (violation of monotonic time).
		if misbehaviour.Header1.SignedHeader.Header.Time.After(misbehaviour.Header2.SignedHeader.Header.Time) {
			return sdkerrors.Wrap(clienttypes.ErrInvalidMisbehaviour, "headers are not at same height and are monotonically increasing")
		}
	}

	// Regardless of the type of misbehaviour, ensure that both headers are valid and would have been accepted by light-client

	// Retrieve trusted consensus states for each Header in misbehaviour
	tmConsensusState1, found := GetConsensusState(clientStore, cdc, misbehaviour.Header1.TrustedHeight)
	if !found {
		return sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header1 at TrustedHeight: %s", misbehaviour.Header1.TrustedHeight)
	}

	tmConsensusState2, found := GetConsensusState(clientStore, cdc, misbehaviour.Header2.TrustedHeight)
	if !found {
		return sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header2 at TrustedHeight: %s", misbehaviour.Header2.TrustedHeight)
	}

	// Check the validity of the two conflicting headers against their respective
	// trusted consensus states
	// NOTE: header height and commitment root assertions are checked in
	// misbehaviour.ValidateBasic by the client keeper and msg.ValidateBasic
	// by the base application.
	if err := checkMisbehaviourHeader(
		cs, tmConsensusState1, misbehaviour.Header1, ctx.BlockTime(),
	); err != nil {
		return sdkerrors.Wrap(err, "verifying Header1 in Misbehaviour failed")
	}
	if err := checkMisbehaviourHeader(
		cs, tmConsensusState2, misbehaviour.Header2, ctx.BlockTime(),
	); err != nil {
		return sdkerrors.Wrap(err, "verifying Header2 in Misbehaviour failed")
	}

	return nil
}

// checkMisbehaviourHeader checks that a Header in Misbehaviour is valid misbehaviour given
// a trusted ConsensusState
func checkMisbehaviourHeader(
	clientState *ClientState, consState *ConsensusState, header *Header, currentTimestamp time.Time,
) error {
	tmTrustedValset, err := tmtypes.ValidatorSetFromProto(header.TrustedValidators)
	if err != nil {
		return sdkerrors.Wrap(err, "trusted validator set is not tendermint validator set type")
	}

	tmCommit, err := tmtypes.CommitFromProto(header.Commit)
	if err != nil {
		return sdkerrors.Wrap(err, "commit is not tendermint commit type")
	}

	// check the trusted fields for the header against ConsensusState
	if err := checkTrustedHeader(header, consState); err != nil {
		return err
	}

	// assert that the age of the trusted consensus state is not older than the trusting period
	if currentTimestamp.Sub(consState.Timestamp) >= clientState.TrustingPeriod {
		return sdkerrors.Wrapf(
			ErrTrustingPeriodExpired,
			"current timestamp minus the latest consensus state timestamp is greater than or equal to the trusting period (%d >= %d)",
			currentTimestamp.Sub(consState.Timestamp), clientState.TrustingPeriod,
		)
	}

	chainID := clientState.GetChainID()
	// If chainID is in revision format, then set revision number of chainID with the revision number
	// of the misbehaviour header
	if clienttypes.IsRevisionFormat(chainID) {
		chainID, _ = clienttypes.SetRevisionNumber(chainID, header.GetHeight().GetRevisionNumber())
	}

	// - ValidatorSet must have TrustLevel similarity with trusted FromValidatorSet
	// - ValidatorSets on both headers are valid given the last trusted ValidatorSet
	if err := tmTrustedValset.VerifyCommitLightTrusting(
		chainID, tmCommit, clientState.TrustLevel.ToTendermint(),
	); err != nil {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidMisbehaviour, "validator set in header has too much change from trusted validator set: %v", err)
	}
	return nil
}
