package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// CheckHeaderAndUpdateState checks if the provided header is valid and updates
// the consensus state if appropriate. It returns an error if:
// - the header provided is not parseable to a solo machine header
// - the header sequence does not match the current sequence
// - the header timestamp is less than the consensus state timestamp
// - the currently registered public key did not provide the update signature
func (cs ClientState) CheckHeaderAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore,
	header exported.Header,
) (exported.ClientState, exported.ConsensusState, error) {
	if err := cs.VerifyHeader(ctx, cdc, clientStore, header); err != nil {
		return nil, nil, err
	}

	return cs.UpdateState(ctx, cdc, clientStore, header)
}

// VerifyHeader checks if the Solo Machine update signature is valid.
func (cs ClientState) VerifyHeader(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore,
	header exported.Header,
) error {
	smHeader, ok := header.(*Header)
	if ok {
		return cs.verifyHeader(ctx, cdc, clientStore, smHeader)
	}

	conflictingSignaturesHeader, ok := header.(*ConflictingSignaturesHeader)
	if ok {
		return cs.checkConflictingSignaturesHeader(ctx, cdc, clientStore, conflictingSignaturesHeader)
	}

	return sdkerrors.Wrapf(
		clienttypes.ErrInvalidHeader, "header type %T, expected  %T", header, &Header{},
	)
}

func (cs ClientState) verifyHeader(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore,
	header *Header,
) error {
	// assert update sequence is current sequence
	if header.Sequence != cs.Sequence {
		return sdkerrors.Wrapf(
			clienttypes.ErrInvalidHeader,
			"header sequence does not match the client state sequence (%d != %d)", header.Sequence, cs.Sequence,
		)
	}

	// assert update timestamp is not less than current consensus state timestamp
	if header.Timestamp < cs.ConsensusState.Timestamp {
		return sdkerrors.Wrapf(
			clienttypes.ErrInvalidHeader,
			"header timestamp is less than to the consensus state timestamp (%d < %d)", header.Timestamp, cs.ConsensusState.Timestamp,
		)
	}

	// assert currently registered public key signed over the new public key with correct sequence
	data, err := HeaderSignBytes(cdc, header)
	if err != nil {
		return err
	}
	return verifySignature(cdc, cs, data, header.Signature)
}

// CheckForMisbehaviour returns true.
func (cs ClientState) CheckForMisbehaviour(
	_ sdk.Context, _ codec.BinaryCodec, _ sdk.KVStore,
	_ exported.Header,
) (bool, error) {
	return true, nil
}

// UpdateState updates the consensus state to the new public key and an incremented sequence.
func (cs ClientState) UpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore,
	header exported.Header,
) (exported.ClientState, exported.ConsensusState, error) {
	smHeader := header.(*Header)
	consensusState := &ConsensusState{
		PublicKey:   smHeader.NewPublicKey,
		Diversifier: smHeader.NewDiversifier,
		Timestamp:   smHeader.Timestamp,
	}

	// increment sequence number
	cs.Sequence++
	cs.ConsensusState = consensusState
	return &cs, consensusState, nil
}

// UpdateStateOnMisbehaviour updates state upon misbehaviour. This method should only be called on misbehaviour
// as it does not perform any misbehaviour checks.
func (cs ClientState) UpdateStateOnMisbehaviour(
	_ sdk.Context, _ codec.BinaryCodec, _ sdk.KVStore, // prematurely include args for self storage of consensus state
) *ClientState {
	cs.IsFrozen = true
	return &cs
}
