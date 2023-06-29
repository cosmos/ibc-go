package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

type (
	checkSubstituteAndUpdateStateInnerPayload struct{}
	checkSubstituteAndUpdateStatePayload      struct {
		CheckSubstituteAndUpdateState checkSubstituteAndUpdateStateInnerPayload `json:"check_substitute_and_update_state"`
	}
)

// CheckSubstituteAndUpdateState will try to update the client with the state of the
// substitute.
func (cs ClientState) CheckSubstituteAndUpdateState(
	ctx sdk.Context,
	_ codec.BinaryCodec,
	subjectClientStore, substituteClientStore sdk.KVStore,
	substituteClient exported.ClientState,
) error {
	var (
		SubjectPrefix    = []byte("subject/")
		SubstitutePrefix = []byte("substitute/")
	)

	_, ok := substituteClient.(*ClientState)
	if !ok {
		return sdkerrors.Wrapf(
			clienttypes.ErrInvalidClient,
			fmt.Sprintf("invalid substitute client state. expected type %T, got %T", &ClientState{}, substituteClient),
		)
	}

	store := newWrappedStore(subjectClientStore, substituteClientStore, SubjectPrefix, SubstitutePrefix)

	payload := checkSubstituteAndUpdateStatePayload{
		CheckSubstituteAndUpdateState: checkSubstituteAndUpdateStateInnerPayload{},
	}

	_, err := call[contractResult](ctx, store, &cs, payload)
	return err
}
