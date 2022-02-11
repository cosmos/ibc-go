package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// CheckMisbehaviourAndUpdateState determines whether or not the currently registered
// public key signed over two different messages with the same sequence. If this is true
// the client state is updated to a frozen status.
// NOTE: Misbehaviour is not tracked for previous public keys, a solo machine may update to
// a new public key before the misbehaviour is processed. Therefore, misbehaviour is data
// order processing dependent.
func (cs ClientState) CheckMisbehaviourAndUpdateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	clientStore sdk.KVStore,
	header exported.Header,
) (exported.ClientState, error) {

	misbehaviour, ok := header.(*ConflictingSignaturesHeader)
	if !ok {
		return nil, sdkerrors.Wrapf(
			clienttypes.ErrInvalidClientType,
			"misbehaviour type %T, expected %T", header, &ConflictingSignaturesHeader{},
		)
	}

	if err := cs.checkConflictingSignaturesHeader(ctx, cdc, clientStore, misbehaviour); err != nil {
		return nil, err // inner errors are wrapped
	}

	cs.IsFrozen = true

	return &cs, nil
}

func (cs ClientState) checkConflictingSignaturesHeader(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	clientStore sdk.KVStore,
	header *ConflictingSignaturesHeader,
) error {
	// NOTE: a check that the misbehaviour message data are not equal is done by
	// misbehaviour.ValidateBasic which is called by the 02-client keeper.

	// verify first signature
	if err := verifySignatureAndData(cdc, cs, header, header.SignatureOne); err != nil {
		return sdkerrors.Wrap(err, "failed to verify signature one")
	}

	// verify second signature
	if err := verifySignatureAndData(cdc, cs, header, header.SignatureTwo); err != nil {
		return sdkerrors.Wrap(err, "failed to verify signature two")
	}

	return nil
}

// verifySignatureAndData verifies that the currently registered public key has signed
// over the provided data and that the data is valid. The data is valid if it can be
// unmarshaled into the specified data type.
func verifySignatureAndData(cdc codec.BinaryCodec, clientState ClientState, misbehaviour *ConflictingSignaturesHeader, sigAndData *SignatureAndData) error {

	// do not check misbehaviour timestamp since we want to allow processing of past misbehaviour

	// ensure data can be unmarshaled to the specified data type
	if _, err := UnmarshalDataByType(cdc, sigAndData.DataType, sigAndData.Data); err != nil {
		return err
	}

	data, err := MisbehaviourSignBytes(
		cdc,
		misbehaviour.Sequence, sigAndData.Timestamp,
		clientState.ConsensusState.Diversifier,
		sigAndData.DataType,
		sigAndData.Data,
	)
	if err != nil {
		return err
	}

	return verifySignature(cdc, clientState, data, sigAndData.Signature)

}

func verifySignature(cdc codec.BinaryCodec, clientState ClientState, data []byte, sigDataBz []byte) error {
	sigData, err := UnmarshalSignatureData(cdc, sigDataBz)
	if err != nil {
		return err
	}

	publicKey, err := clientState.ConsensusState.GetPubKey()
	if err != nil {
		return err
	}

	return VerifySignature(publicKey, data, sigData)
}
