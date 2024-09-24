package solomachine

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new ClientState instance.
func NewClientState(latestSequence uint64, consensusState *ConsensusState) *ClientState {
	return &ClientState{
		Sequence:       latestSequence,
		IsFrozen:       false,
		ConsensusState: consensusState,
	}
}

// ClientType is Solo Machine.
func (ClientState) ClientType() string {
	return exported.Solomachine
}

// Validate performs basic validation of the client state fields.
func (cs ClientState) Validate() error {
	if cs.Sequence == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "sequence cannot be 0")
	}
	if cs.ConsensusState == nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "consensus state cannot be nil")
	}
	return cs.ConsensusState.ValidateBasic()
}

// verifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the latest sequence.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
func (cs *ClientState) verifyMembership(
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	publicKey, sigData, timestamp, sequence, err := produceVerificationArgs(cdc, cs, proof)
	if err != nil {
		return err
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	if len(merklePath.GetKeyPath()) != 2 {
		return errorsmod.Wrapf(host.ErrInvalidPath, "path must be of length 2: %s", merklePath.GetKeyPath())
	}

	// in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
	key, err := merklePath.GetKey(1)
	if err != nil {
		return errorsmod.Wrapf(host.ErrInvalidPath, "key not found at index 1: %v", err)
	}

	signBytes := &SignBytes{
		Sequence:    sequence,
		Timestamp:   timestamp,
		Diversifier: cs.ConsensusState.Diversifier,
		Path:        key,
		Data:        value,
	}

	signBz, err := cdc.Marshal(signBytes)
	if err != nil {
		return err
	}

	if err := VerifySignature(publicKey, signBz, sigData); err != nil {
		return err
	}

	cs.Sequence++
	cs.ConsensusState.Timestamp = timestamp
	setClientState(clientStore, cdc, cs)

	return nil
}

// verifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at the latest sequence.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
func (cs *ClientState) verifyNonMembership(
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	proof []byte,
	path exported.Path,
) error {
	publicKey, sigData, timestamp, sequence, err := produceVerificationArgs(cdc, cs, proof)
	if err != nil {
		return err
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	if len(merklePath.GetKeyPath()) != 2 {
		return errorsmod.Wrapf(host.ErrInvalidPath, "path must be of length 2: %s", merklePath.GetKeyPath())
	}

	// in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
	key, err := merklePath.GetKey(1)
	if err != nil {
		return errorsmod.Wrapf(host.ErrInvalidPath, "key not found at index 1: %v", err)
	}

	signBytes := &SignBytes{
		Sequence:    sequence,
		Timestamp:   timestamp,
		Diversifier: cs.ConsensusState.Diversifier,
		Path:        key,
		Data:        nil,
	}

	signBz, err := cdc.Marshal(signBytes)
	if err != nil {
		return err
	}

	if err := VerifySignature(publicKey, signBz, sigData); err != nil {
		return err
	}

	cs.Sequence++
	cs.ConsensusState.Timestamp = timestamp
	setClientState(clientStore, cdc, cs)

	return nil
}

// produceVerificationArgs performs the basic checks on the arguments that are
// shared between the verification functions and returns the public key of the
// consensus state, the unmarshalled proof representing the signature and timestamp.
func produceVerificationArgs(
	cdc codec.BinaryCodec,
	cs *ClientState,
	proof []byte,
) (cryptotypes.PubKey, signing.SignatureData, uint64, uint64, error) {
	if proof == nil {
		return nil, nil, 0, 0, errorsmod.Wrap(ErrInvalidProof, "proof cannot be empty")
	}

	var timestampedSigData TimestampedSignatureData
	if err := cdc.Unmarshal(proof, &timestampedSigData); err != nil {
		return nil, nil, 0, 0, errorsmod.Wrapf(err, "failed to unmarshal proof into type %T", timestampedSigData)
	}

	timestamp := timestampedSigData.Timestamp
	if len(timestampedSigData.SignatureData) == 0 {
		return nil, nil, 0, 0, errorsmod.Wrap(ErrInvalidProof, "signature data cannot be empty")
	}

	sigData, err := UnmarshalSignatureData(cdc, timestampedSigData.SignatureData)
	if err != nil {
		return nil, nil, 0, 0, err
	}

	if cs.ConsensusState.GetTimestamp() > timestamp {
		return nil, nil, 0, 0, errorsmod.Wrapf(ErrInvalidProof, "the consensus state timestamp is greater than the signature timestamp (%d >= %d)", cs.ConsensusState.GetTimestamp(), timestamp)
	}

	sequence := cs.Sequence
	publicKey, err := cs.ConsensusState.GetPubKey()
	if err != nil {
		return nil, nil, 0, 0, err
	}

	return publicKey, sigData, timestamp, sequence, nil
}

// sets the client state to the store
func setClientState(store storetypes.KVStore, cdc codec.BinaryCodec, clientState exported.ClientState) {
	bz := clienttypes.MustMarshalClientState(cdc, clientState)
	store.Set(host.ClientStateKey(), bz)
}
