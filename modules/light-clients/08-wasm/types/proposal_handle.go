package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type checkSubstituteAndUpdateStatePayload struct {
	CheckSubstituteAndUpdateState CheckSubstituteAndUpdateStatePayload `json:"check_substitute_and_update_state"`
}

type CheckSubstituteAndUpdateStatePayload struct {
	ClientState              ClientState             `json:"client_state"`
	SubjectConsensusState    exported.ConsensusState `json:"subject_consensus_state"`
	SubstituteConsensusState exported.ClientState    `json:"substitute_client_state"`
}

func (c ClientState) CheckSubstituteAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore,
	substituteClientStore sdk.KVStore, substituteClient exported.ClientState,
) error {
	var (
		SubjectPrefix    = []byte("subject/")
		SubstitutePrefix = []byte("substitute/")
	)

	consensusState, err := GetConsensusState(subjectClientStore, cdc, c.LatestHeight)
	if err != nil {
		return sdkerrors.Wrapf(
			err, "unexpected error: could not get consensus state from clientstore at height: %d", c.GetLatestHeight(),
		)
	}

	store := NewWrappedStore(subjectClientStore, substituteClientStore, SubjectPrefix, SubstitutePrefix)

	payload := checkSubstituteAndUpdateStatePayload{
		CheckSubstituteAndUpdateState: CheckSubstituteAndUpdateStatePayload{
			ClientState:              c,
			SubjectConsensusState:    consensusState,
			SubstituteConsensusState: substituteClient,
		},
	}

	output, err := call[clientStateCallResponse](payload, &c, ctx, store)
	if err != nil {
		return err
	}

	output.resetImmutables(&c)
	return nil
}
