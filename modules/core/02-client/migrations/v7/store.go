package v7

import (
	"context"
	"strings"

	"cosmossdk.io/core/log"
	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	coretypes "github.com/cosmos/ibc-go/v9/modules/core/types"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
)

// Localhost is the client type for a localhost client. It is also used as the clientID
// for the localhost client.
const Localhost string = "09-localhost"

// MigrateStore performs in-place store migrations from ibc-go v6 to ibc-go v7.
// The migration includes:
//
// - Migrating solo machine client states from v2 to v3 protobuf definition
// - Pruning all solo machine consensus states
// - Removing the localhost client
// - Asserting existing tendermint clients are properly registered on the chain codec
func MigrateStore(ctx context.Context, logger log.Logger, storeService corestore.KVStoreService, cdc codec.BinaryCodec, clientKeeper ClientKeeper) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

	if err := handleSolomachineMigration(ctx, logger, store, cdc, clientKeeper); err != nil {
		return err
	}

	if err := handleTendermintMigration(ctx, logger, store, clientKeeper); err != nil {
		return err
	}

	return handleLocalhostMigration(ctx, logger, store, clientKeeper)
}

// handleSolomachineMigration iterates over the solo machine clients and migrates client state from
// protobuf definition v2 to v3. All consensus states stored outside of the client state are pruned.
func handleSolomachineMigration(ctx context.Context, logger log.Logger, store storetypes.KVStore, cdc codec.BinaryCodec, clientKeeper ClientKeeper) error {
	clients, err := collectClients(logger, store, exported.Solomachine)
	if err != nil {
		return err
	}

	for _, clientID := range clients {
		clientStore := clientKeeper.ClientStore(ctx, clientID)

		bz := clientStore.Get(host.ClientStateKey())
		if len(bz) == 0 {
			return errorsmod.Wrapf(clienttypes.ErrClientNotFound, "clientID %s", clientID)
		}

		var protoAny codectypes.Any
		if err := cdc.Unmarshal(bz, &protoAny); err != nil {
			return errorsmod.Wrap(err, "failed to unmarshal client state bytes into solo machine client state")
		}

		var clientState ClientState
		if err := cdc.Unmarshal(protoAny.Value, &clientState); err != nil {
			return errorsmod.Wrap(err, "failed to unmarshal client state bytes into solo machine client state")
		}

		updatedClientState := migrateSolomachine(clientState)

		// update solomachine in store
		clientKeeper.SetClientState(ctx, clientID, &updatedClientState)

		removeAllClientConsensusStates(clientStore)
	}

	return nil
}

// handleTendermintMigration asserts that the tendermint client in state can be decoded properly.
// This ensures the upgrading chain properly registered the tendermint client types on the chain codec.
func handleTendermintMigration(ctx context.Context, logger log.Logger, store storetypes.KVStore, clientKeeper ClientKeeper) error {
	clients, err := collectClients(logger, store, exported.Tendermint)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		return nil // no-op if no tm clients exist
	}

	if len(clients) > 1 {
		return errorsmod.Wrap(ibcerrors.ErrLogic, "more than one Tendermint client collected")
	}

	clientID := clients[0]

	// unregistered tendermint client types will panic when unmarshaling the client state
	// in GetClientState
	clientState, ok := clientKeeper.GetClientState(ctx, clientID)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrClientNotFound, "clientID %s", clientID)
	}

	_, ok = clientState.(*ibctm.ClientState)
	if !ok {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "client state is not tendermint even though client id contains 07-tendermint")
	}

	return nil
}

// handleLocalhostMigration removes all client and consensus states associated with the localhost client type.
func handleLocalhostMigration(ctx context.Context, logger log.Logger, store storetypes.KVStore, clientKeeper ClientKeeper) error {
	clients, err := collectClients(logger, store, Localhost)
	if err != nil {
		return err
	}

	for _, clientID := range clients {
		clientStore := clientKeeper.ClientStore(ctx, clientID)

		// delete the client state
		clientStore.Delete(host.ClientStateKey())

		removeAllClientConsensusStates(clientStore)
	}

	return nil
}

// collectClients will iterate over the provided client type prefix in the client store
// and return a list of clientIDs associated with the client type. This is necessary to
// avoid state corruption as modifying state during iteration is unsafe. A special case
// for tendermint clients is included as only one tendermint clientID is required for
// v7 migrations.
func collectClients(logger log.Logger, store storetypes.KVStore, clientType string) (clients []string, err error) {
	clientPrefix := host.PrefixedClientStoreKey([]byte(clientType))
	iterator := storetypes.KVStorePrefixIterator(store, clientPrefix)

	defer coretypes.LogDeferred(logger, func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		path := string(iterator.Key())
		if !strings.Contains(path, host.KeyClientState) {
			// skip non client state keys
			continue
		}

		clientID := host.MustParseClientStatePath(path)
		clients = append(clients, clientID)

		// optimization: exit after a single tendermint client iteration
		if strings.Contains(clientID, exported.Tendermint) {
			return clients, nil
		}
	}

	return clients, nil
}

// removeAllClientConsensusStates removes all client consensus states from the associated
// client store.
func removeAllClientConsensusStates(clientStore storetypes.KVStore) {
	iterator := storetypes.KVStorePrefixIterator(clientStore, []byte(host.KeyConsensusStatePrefix))
	var heights []exported.Height

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		// key is in the format "consensusStates/<height>"
		if len(keySplit) != 2 || keySplit[0] != string(host.KeyConsensusStatePrefix) {
			continue
		}

		// collect consensus states to be pruned
		heights = append(heights, clienttypes.MustParseHeight(keySplit[1]))
	}

	// delete all consensus states
	for _, height := range heights {
		clientStore.Delete(host.ConsensusStateKey(height))
	}
}

// migrateSolomachine migrates the solomachine from v2 to v3 solo machine protobuf definition.
// Notably it drops the AllowUpdateAfterProposal field.
func migrateSolomachine(clientState ClientState) solomachine.ClientState {
	consensusState := &solomachine.ConsensusState{
		PublicKey:   clientState.ConsensusState.PublicKey,
		Diversifier: clientState.ConsensusState.Diversifier,
		Timestamp:   clientState.ConsensusState.Timestamp,
	}

	return solomachine.ClientState{
		Sequence:       clientState.Sequence,
		IsFrozen:       clientState.IsFrozen,
		ConsensusState: consensusState,
	}
}
