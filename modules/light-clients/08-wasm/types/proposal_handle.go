package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// CheckSubstituteAndUpdateState will try to update the client with the state of the
// substitute.
func (cs ClientState) CheckSubstituteAndUpdateState(
	ctx sdk.Context,
	_ codec.BinaryCodec,
	subjectClientStore, substituteClientStore storetypes.KVStore,
	substituteClient exported.ClientState,
) error {
	var (
		subjectPrefix    = []byte("subject/")
		substitutePrefix = []byte("substitute/")
	)

	_, ok := substituteClient.(*ClientState)
	if !ok {
		return errorsmod.Wrap(
			clienttypes.ErrInvalidClient,
			fmt.Sprintf("invalid substitute client state. expected type %T, got %T", &ClientState{}, substituteClient),
		)
	}

	store := newUpdateProposalWrappedStore(subjectClientStore, substituteClientStore, subjectPrefix, substitutePrefix)

	payload := SudoMsg{
		CheckSubstituteAndUpdateState: &CheckSubstituteAndUpdateStateMsg{},
	}

	_, err := wasmCall[EmptyResult](ctx, store, &cs, payload)
	return err
}
