package attestations

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new ClientState instance.
func NewClientState(attestorAddresses []string, minRequiredSigs uint32, latestHeight uint64) *ClientState {
	return &ClientState{
		AttestorAddresses: attestorAddresses,
		MinRequiredSigs:   minRequiredSigs,
		LatestHeight:      latestHeight,
		IsFrozen:          false,
	}
}

// ClientType is Attestations.
func (ClientState) ClientType() string {
	return exported.Attestations
}

// Validate performs basic validation of the client state fields.
func (cs ClientState) Validate() error {
	if len(cs.AttestorAddresses) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "attestor addresses cannot be empty")
	}
	if cs.MinRequiredSigs == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "min required sigs cannot be 0")
	}
	if cs.MinRequiredSigs > uint32(len(cs.AttestorAddresses)) {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "min required sigs cannot exceed number of attestors")
	}

	seen := make(map[string]bool)
	for _, addr := range cs.AttestorAddresses {
		if addr == "" {
			return errorsmod.Wrap(clienttypes.ErrInvalidClient, "attestor address cannot be empty")
		}
		if seen[addr] {
			return errorsmod.Wrap(clienttypes.ErrInvalidClient, "duplicate attestor address")
		}
		seen[addr] = true
	}

	if cs.LatestHeight == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "latest height must be greater than 0")
	}

	return nil
}

// verifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
func (cs *ClientState) verifyMembership(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	if cs.IsFrozen {
		return ErrClientFrozen
	}

	if path == nil || path.Empty() {
		return errorsmod.Wrap(ErrInvalidPath, "path cannot be empty")
	}

	if len(value) == 0 {
		return errorsmod.Wrap(ErrInvalidAttestationData, "value cannot be empty")
	}

	if _, found := getConsensusState(clientStore, cdc, height); !found {
		return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "consensus state not found for height %s", height)
	}

	var attestationProof AttestationProof
	if err := cdc.Unmarshal(proof, &attestationProof); err != nil {
		return errorsmod.Wrapf(ErrInvalidAttestationProof, "failed to unmarshal proof: %v", err)
	}

	if err := cs.verifySignatures(cdc, &attestationProof); err != nil {
		return err
	}

	var packetAttestation PacketAttestation
	if err := cdc.Unmarshal(attestationProof.AttestationData, &packetAttestation); err != nil {
		return errorsmod.Wrapf(ErrInvalidAttestationData, "failed to unmarshal attestation data: %v", err)
	}

	if packetAttestation.Height != height.GetRevisionHeight() {
		return errorsmod.Wrapf(ErrInvalidHeight, "height mismatch: expected %d, got %d", height.GetRevisionHeight(), packetAttestation.Height)
	}

	if len(packetAttestation.Packets) == 0 {
		return errorsmod.Wrap(ErrInvalidAttestationData, "packets cannot be empty")
	}

	commitmentPath, err := canonicalizePath(path)
	if err != nil {
		return err
	}

	for _, packet := range packetAttestation.Packets {
		if len(packet.Commitment) == 32 && len(value) == 32 && bytes.Equal(packet.Commitment, value) && bytes.Equal(packet.Path, commitmentPath) {
			return nil
		}
	}

	return ErrNotMember
}

// verifyNonMembership is not supported in this version.
func (cs *ClientState) verifyNonMembership(
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	proof []byte,
	path exported.Path,
) error {
	return errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "verifyNonMembership is not supported")
}

func canonicalizePath(path exported.Path) ([]byte, error) {
	switch p := path.(type) {
	case interface{ GetKeyPath() [][]byte }:
		flattened := bytes.Join(p.GetKeyPath(), []byte("/"))
		return normalizePathBytes(flattened), nil
	case interface{ Bytes() []byte }:
		return normalizePathBytes(p.Bytes()), nil
	default:
		if stringer, ok := path.(fmt.Stringer); ok {
			return normalizePathBytes([]byte(stringer.String())), nil
		}
	}

	return nil, errorsmod.Wrapf(ErrInvalidPath, "unsupported path type %T", path)
}

func normalizePathBytes(raw []byte) []byte {
	if len(raw) == 32 {
		return raw
	}

	sum := sha256.Sum256(raw)
	return sum[:]
}

// sets the client state to the store
func setClientState(store storetypes.KVStore, cdc codec.BinaryCodec, clientState exported.ClientState) {
	bz := clienttypes.MustMarshalClientState(cdc, clientState)
	store.Set(host.ClientStateKey(), bz)
}
