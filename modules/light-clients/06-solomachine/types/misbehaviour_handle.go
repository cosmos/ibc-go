package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// verifySignatureAndData verifies that the currently registered public key has signed
// over the provided data and that the data is valid. The data is valid if it can be
// unmarshaled into the specified data type.
func (cs ClientState) verifySignatureAndData(cdc codec.BinaryCodec, misbehaviour *Misbehaviour, sigAndData *SignatureAndData) error {

	// do not check misbehaviour timestamp since we want to allow processing of past misbehaviour

	// ensure data can be unmarshaled to the specified data type
	if _, err := UnmarshalDataByType(cdc, sigAndData.DataType, sigAndData.Data); err != nil {
		return err
	}

	data, err := MisbehaviourSignBytes(
		cdc,
		misbehaviour.Sequence, sigAndData.Timestamp,
		cs.ConsensusState.Diversifier,
		sigAndData.DataType,
		sigAndData.Data,
	)
	if err != nil {
		return err
	}

	sigData, err := UnmarshalSignatureData(cdc, sigAndData.Signature)
	if err != nil {
		return err
	}

	publicKey, err := cs.ConsensusState.GetPubKey()
	if err != nil {
		return err
	}

	if err := VerifySignature(publicKey, data, sigData); err != nil {
		return err
	}

	return nil

}
