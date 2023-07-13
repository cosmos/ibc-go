package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type verifyUpgradeAndUpdateStatePayload struct {
	VerifyUpgradeAndUpdateStateMsg verifyUpgradeAndUpdateStateMsgPayload `json:"verify_upgrade_and_update_state"`
}

type verifyUpgradeAndUpdateStateMsgPayload struct {
	UpgradeClientState         exported.ClientState    `json:"upgrade_client_state"`
	UpgradeConsensusState      exported.ConsensusState `json:"upgrade_consensus_state"`
	ProofUpgradeClient         []byte                  `json:"proof_upgrade_client"`
	ProofUpgradeConsensusState []byte                  `json:"proof_upgrade_consensus_state"`
}

// VerifyUpgradeAndUpdateState, on a successful verification expects the contract to update
// the new client state, consensus state, and any other client metadata
func (c ClientState) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	store sdk.KVStore,
	newClient exported.ClientState,
	newConsState exported.ConsensusState,
	proofUpgradeClient,
	proofUpgradeConsState []byte,
) error {
	wasmUpgradeClientState, ok := newClient.(*ClientState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "upgraded client state must be wasm light client state. expected %T, got: %T",
			&ClientState{}, wasmUpgradeClientState)
	}

	wasmUpgradeConsState, ok := newConsState.(*ConsensusState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "upgraded consensus state must be wasm light consensus state. expected %T, got: %T",
			&ConsensusState{}, wasmUpgradeConsState)
	}

	// last height of current counterparty chain must be client's latest height
	lastHeight := c.LatestHeight
	_, err := GetConsensusState(store, cdc, lastHeight)
	if err != nil {
		return sdkerrors.Wrap(err, "could not retrieve consensus state for lastHeight")
	}

	payload := verifyUpgradeAndUpdateStatePayload{
		VerifyUpgradeAndUpdateStateMsg: verifyUpgradeAndUpdateStateMsgPayload{
			UpgradeClientState:         newClient,
			UpgradeConsensusState:      newConsState,
			ProofUpgradeClient:         proofUpgradeClient,
			ProofUpgradeConsensusState: proofUpgradeConsState,
		},
	}

	_, err = call[contractResult](payload, &c, ctx, store)

	return err
}
