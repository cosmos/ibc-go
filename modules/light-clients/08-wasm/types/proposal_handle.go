package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type checkSubstituteAndUpdateStatePayload struct {
	CheckSubstituteAndUpdateState CheckSubstituteAndUpdateStatePayload `json:"check_substitute_and_update_state"`
}

type CheckSubstituteAndUpdateStatePayload struct{}

func (c ClientState) CheckSubstituteAndUpdateState(
	ctx sdk.Context, _ codec.BinaryCodec, subjectClientStore,
	substituteClientStore sdk.KVStore, substituteClient exported.ClientState,
) error {
	var (
		SubjectPrefix    = []byte("subject/")
		SubstitutePrefix = []byte("substitute/")
	)

	_, ok := substituteClient.(*ClientState)
	if !ok {
		return sdkerrors.Wrapf(
			ErrUnableToCall,
			fmt.Sprintf("substitute client state, expected type %T, got %T", &ClientState{}, substituteClient),
		)
	}

	store := NewWrappedStore(subjectClientStore, substituteClientStore, SubjectPrefix, SubstitutePrefix)

	payload := checkSubstituteAndUpdateStatePayload{
		CheckSubstituteAndUpdateState: CheckSubstituteAndUpdateStatePayload{},
	}

	_, err := call[contractResult](payload, &c, ctx, store)
	if err != nil {
		return err
	}

	return nil
}
