package migrations

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
)

// PruneExpiredConsensusStates prunes all expired tendermint consensus states. This function
// may optionally be called during in-place store migrations. The ibc store key must be provided.
func PruneExpiredConsensusStates(ctx context.Context, cdc codec.BinaryCodec, clientKeeper ClientKeeper) (int, error) {
	var clientIDs []string
	clientKeeper.IterateClientStates(ctx, []byte(exported.Tendermint), func(clientID string, _ exported.ClientState) bool {
		clientIDs = append(clientIDs, clientID)
		return false
	})

	// keep track of the total consensus states pruned so chains can
	// understand how much space is saved when the migration is run
	var totalPruned int

	for _, clientID := range clientIDs {
		clientStore := clientKeeper.ClientStore(ctx, clientID)

		clientState, ok := clientKeeper.GetClientState(ctx, clientID)
		if !ok {
			return 0, errorsmod.Wrapf(clienttypes.ErrClientNotFound, "clientID %s", clientID)
		}

		tmClientState, ok := clientState.(*ibctm.ClientState)
		if !ok {
			return 0, errorsmod.Wrap(clienttypes.ErrInvalidClient, "client state is not tendermint even though client id contains 07-tendermint")
		}

		totalPruned += ibctm.PruneAllExpiredConsensusStates(ctx, clientStore, cdc, tmClientState)
	}

	clientKeeper.Logger(ctx).Info("pruned expired tendermint consensus states", "total", totalPruned)

	return totalPruned, nil
}
