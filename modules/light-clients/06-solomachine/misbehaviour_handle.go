package solomachine

import (
	"github.com/cosmos/cosmos-sdk/codec"

	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
)

// verifySignatureAndData verifies that the currently registered public key has signed
// over the provided data and that the data is valid. The data is valid if it can be
// unmarshaled into the specified data type.
func (cs ClientState) verifySignatureAndData(cdc codec.BinaryCodec, misbehaviour *Misbehaviour, sigAndData *SignatureAndData) error {
	// do not check misbehaviour timestamp since we want to allow processing of past misbehaviour
	if err := cdc.Unmarshal(sigAndData.Path, new(commitmenttypes.MerklePath)); err != nil {
		return err
	}

	signBytes := SignBytes{
		Sequence:    misbehaviour.Sequence,
		Timestamp:   sigAndData.Timestamp,
		Diversifier: cs.ConsensusState.Diversifier,
		Path:        sigAndData.Path,
		Data:        sigAndData.Data,
	}

	data, err := cdc.Marshal(&signBytes)
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
