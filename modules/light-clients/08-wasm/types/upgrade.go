package types

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type verifyUpgradeAndUpdateStatePayload struct {
	VerifyUpgradeAndUpdateStateMsg verifyUpgradeAndUpdateStateMsgPayload `json:"verify_upgrade_and_update_state_msg"`
}

type verifyUpgradeAndUpdateStateMsgPayload struct {
	ClientState                ClientState             `json:"old_client_state"`
	SubjectConsensusState      exported.ClientState    `json:"upgrade_client_state"`
	UpgradeConsensusState      exported.ConsensusState `json:"upgrade_consensus_state"`
	ProofUpgradeClient         []byte                  `json:"proof_upgrade_client"`
	ProofUpgradeConsensusState []byte                  `json:"proof_upgrade_consensus_state"`
}

func (c ClientState) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	store sdk.KVStore,
	newClient exported.ClientState,
	newConsState exported.ConsensusState,
	proofUpgradeClient,
	proofUpgradeConsState []byte,
) error {
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
			ClientState:                c,
			SubjectConsensusState:      newClient,
			UpgradeConsensusState:      newConsState,
			ProofUpgradeClient:         proofUpgradeClient,
			ProofUpgradeConsensusState: proofUpgradeConsState,
		},
	}

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToMarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	out, err := callContract(c.CodeId, ctx, store, encodedData)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToCall, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	output := contractResult{}
	if err := json.Unmarshal(out.Data, &output); err != nil {
		return sdkerrors.Wrapf(ErrUnableToUnmarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	if !output.IsValid {
		return fmt.Errorf("%s error occurred while verifyig upgrade and updating client state", output.ErrorMsg)
	}

	return nil
}