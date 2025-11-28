package attestations

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// VerifyClientMessage introspects the provided ClientMessage and checks its validity.
// An AttestationProof is considered valid if it has valid signatures from unique attestors meeting quorum.
func (cs *ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) error {
	if cs.IsFrozen {
		return ErrClientFrozen
	}

	attestationProof, ok := clientMsg.(*AttestationProof)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected type %T, got type %T", (*AttestationProof)(nil), clientMsg)
	}

	return cs.verifySignatures(attestationProof)
}

// CheckForMisbehaviour checks for evidence of misbehaviour.
// For attestations client, misbehaviour is detected when a consensus state already exists
// for a height but with a different timestamp.
func (cs *ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) bool {
	if cs.IsFrozen {
		return false
	}

	attestationProof, ok := clientMsg.(*AttestationProof)
	if !ok {
		return false
	}

	var stateAttestation StateAttestation
	if err := cdc.Unmarshal(attestationProof.AttestationData, &stateAttestation); err != nil {
		return false
	}

	if stateAttestation.Height == 0 || stateAttestation.Timestamp == 0 {
		return false
	}

	height := clienttypes.NewHeight(0, stateAttestation.Height)
	existingConsensusState, found := getConsensusState(clientStore, cdc, height)
	if found && existingConsensusState.Timestamp != stateAttestation.Timestamp {
		return true
	}

	return false
}

// UpdateState updates the consensus state to a new height and timestamp.
// A list containing the updated consensus height is returned.
func (cs *ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	if cs.IsFrozen {
		return []exported.Height{}
	}

	attestationProof, ok := clientMsg.(*AttestationProof)
	if !ok {
		return []exported.Height{}
	}

	var stateAttestation StateAttestation
	if err := cdc.Unmarshal(attestationProof.AttestationData, &stateAttestation); err != nil {
		return []exported.Height{}
	}

	if stateAttestation.Height == 0 || stateAttestation.Timestamp == 0 {
		return []exported.Height{}
	}

	height := clienttypes.NewHeight(0, stateAttestation.Height)

	existingConsensusState, found := getConsensusState(clientStore, cdc, height)
	if found {
		if existingConsensusState.Timestamp != stateAttestation.Timestamp {
			cs.IsFrozen = true
			setClientState(clientStore, cdc, cs)
			return []exported.Height{}
		}
		return []exported.Height{height}
	}

	consensusState := &ConsensusState{
		Timestamp: stateAttestation.Timestamp,
	}

	setConsensusState(ctx, clientStore, cdc, consensusState, height)

	if stateAttestation.Height > cs.LatestHeight {
		cs.LatestHeight = stateAttestation.Height
	}

	setClientState(clientStore, cdc, cs)

	return []exported.Height{height}
}
