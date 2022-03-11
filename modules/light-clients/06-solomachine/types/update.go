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
	msg exported.Header, // TODO: Update to exported.ClientMessage
) (exported.ClientState, exported.ConsensusState, error) {
	if err := cs.VerifyClientMessage(cdc, msg); err != nil {
		return nil, nil, err
	}

	return cs.UpdateState(ctx, cdc, clientStore, msg)
}

// VerifyClientMessage checks if the Solo Machine update signature(s) is valid.
func (cs ClientState) VerifyClientMessage(cdc codec.BinaryCodec, clientMsg exported.Header) error {
	switch msg := clientMsg.(type) {
	case *Header:
		// assert update sequence is current sequence
		if msg.Sequence != cs.Sequence {
			return sdkerrors.Wrapf(
				clienttypes.ErrInvalidHeader,
				"header sequence does not match the client state sequence (%d != %d)", msg.Sequence, cs.Sequence,
			)
		}

		// assert update timestamp is not less than current consensus state timestamp
		if msg.Timestamp < cs.ConsensusState.Timestamp {
			return sdkerrors.Wrapf(
				clienttypes.ErrInvalidHeader,
				"header timestamp is less than to the consensus state timestamp (%d < %d)", msg.Timestamp, cs.ConsensusState.Timestamp,
			)
		}

		// assert currently registered public key signed over the new public key with correct sequence
		data, err := HeaderSignBytes(cdc, msg)
		if err != nil {
			return err
		}

		sigData, err := UnmarshalSignatureData(cdc, msg.Signature)
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
	case *Misbehaviour:
		// NOTE: a check that the misbehaviour message data are not equal is done by
		// misbehaviour.ValidateBasic which is called by the 02-client keeper.
		// verify first signature
		if err := verifySignatureAndData(cdc, cs, msg, msg.SignatureOne); err != nil {
			return sdkerrors.Wrap(err, "failed to verify signature one")
		}

		// verify second signature
		if err := verifySignatureAndData(cdc, cs, msg, msg.SignatureTwo); err != nil {
			return sdkerrors.Wrap(err, "failed to verify signature two")
		}

	default:
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "expected type %T, got type %T", Header{}, msg)
	}
	return nil
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

	cs.Sequence++
	cs.ConsensusState = consensusState
	return &cs, consensusState, nil
}
