package attestations

import (
	"fmt"

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

// UpdateState updates the consensus state to a new height and timestamp.
// A list containing the updated consensus height is returned.
// Since client message is validated in VerifyClientMessage, we don't validate much here, and panics on anything unexpected.
func (cs *ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	attestationProof, ok := clientMsg.(*AttestationProof)
	if !ok {
		panic(fmt.Sprintf("expected type %T, got type %T", (*AttestationProof)(nil), clientMsg))
	}

	var stateAttestation StateAttestation
	if err := cdc.Unmarshal(attestationProof.AttestationData, &stateAttestation); err != nil {
		panic(fmt.Sprintf("failed to unmarshal attestation data: %v", err))
	}

	height := clienttypes.NewHeight(0, stateAttestation.Height)
	consensusState := &ConsensusState{
		Timestamp: stateAttestation.Timestamp,
	}

	setConsensusState(clientStore, cdc, consensusState, height)

	if stateAttestation.Height > cs.LatestHeight {
		cs.LatestHeight = stateAttestation.Height
	}

	setClientState(clientStore, cdc, cs)

	return []exported.Height{height}
}
