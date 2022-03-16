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
	msg exported.ClientMessage,
) (exported.ClientState, exported.ConsensusState, error) {
	if err := cs.VerifyClientMessage(ctx, cdc, clientStore, msg); err != nil {
		return nil, nil, err
	}

	return cs.UpdateState(ctx, cdc, clientStore, msg)
}

// VerifyClientMessage introspects the provided ClientMessage and checks its validity
// A Solomachine Header is considered valid if the currently registered public key has signed over the new public key with the correct sequence
// A Solomachine Misbehaviour is considered valid if duplicate signatures of the current public key are found on two different messages at a given sequence
func (cs ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	switch msg := clientMsg.(type) {
	case *Header:
		return cs.verifyHeader(ctx, cdc, clientStore, msg)
	case *Misbehaviour:
		return cs.verifyMisbehaviour(ctx, cdc, clientStore, msg)
	default:
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "expected type of %T or %T, got type %T", Header{}, Misbehaviour{}, msg)
	}
}

func (cs ClientState) verifyHeader(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, header *Header) error {
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

	sigData, err := UnmarshalSignatureData(cdc, header.Signature)
	if err != nil {
		return err
	}

	publicKey, err := cs.ConsensusState.GetPubKey()
	if err != nil {
		return err
	}

	if err := VerifySignature(publicKey, data, sigData); err != nil {
		return sdkerrors.Wrap(ErrInvalidHeader, err.Error())
	}

	return nil
}

func (cs ClientState) verifyMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, misbehaviour *Misbehaviour) error {
	// NOTE: a check that the misbehaviour message data are not equal is done by
	// misbehaviour.ValidateBasic which is called by the 02-client keeper.
	// verify first signature
	if err := verifySignatureAndData(cdc, cs, misbehaviour, misbehaviour.SignatureOne); err != nil {
		return sdkerrors.Wrap(err, "failed to verify signature one")
	}

	// verify second signature
	if err := verifySignatureAndData(cdc, cs, misbehaviour, misbehaviour.SignatureTwo); err != nil {
		return sdkerrors.Wrap(err, "failed to verify signature two")
	}

	return nil
}

// UpdateState updates the consensus state to the new public key and an incremented sequence.
func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) (exported.ClientState, exported.ConsensusState, error) {
	smHeader := clientMsg.(*Header)
	consensusState := &ConsensusState{
		PublicKey:   smHeader.NewPublicKey,
		Diversifier: smHeader.NewDiversifier,
		Timestamp:   smHeader.Timestamp,
	}

	cs.Sequence++
	cs.ConsensusState = consensusState
	return &cs, consensusState, nil
}
