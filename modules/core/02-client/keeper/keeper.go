package keeper

import (
	"errors"
	"fmt"
	"strings"

	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	localhost "github.com/cosmos/ibc-go/v10/modules/light-clients/09-localhost"
)

// Keeper represents a type that grants read and write permissions to any client
// state information
type Keeper struct {
	storeService   corestore.KVStoreService
	cdc            codec.BinaryCodec
	router         *types.Router
	legacySubspace types.ParamSubspace
	upgradeKeeper  types.UpgradeKeeper
}

// NewKeeper creates a new NewKeeper instance
func NewKeeper(cdc codec.BinaryCodec, storeService corestore.KVStoreService, legacySubspace types.ParamSubspace, uk types.UpgradeKeeper) *Keeper {
	router := types.NewRouter()
	localhostModule := localhost.NewLightClientModule(cdc, storeService)
	router.AddRoute(exported.Localhost, localhostModule)

	return &Keeper{
		storeService:   storeService,
		cdc:            cdc,
		router:         router,
		legacySubspace: legacySubspace,
		upgradeKeeper:  uk,
	}
}

// Codec returns the IBC Client module codec.
func (k *Keeper) Codec() codec.BinaryCodec {
	return k.cdc
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"/"+types.SubModuleName)
}

// AddRoute adds a new route to the underlying router.
func (k *Keeper) AddRoute(clientType string, module exported.LightClientModule) {
	k.router.AddRoute(clientType, module)
}

// GetStoreProvider returns the light client store provider.
func (k *Keeper) GetStoreProvider() types.StoreProvider {
	return types.NewStoreProvider(k.storeService)
}

// Route returns the light client module for the given client identifier.
func (k *Keeper) Route(ctx sdk.Context, clientID string) (exported.LightClientModule, error) {
	clientType, _, err := types.ParseClientIdentifier(clientID)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "unable to parse client identifier %s", clientID)
	}

	if !k.GetParams(ctx).IsAllowedClient(clientType) {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidClientType,
			"client (%s) type %s is not in the allowed client list", clientID, clientType,
		)
	}

	clientModule, found := k.router.GetRoute(clientType)
	if !found {
		return nil, errorsmod.Wrap(types.ErrRouteNotFound, clientID)
	}

	return clientModule, nil
}

// GenerateClientIdentifier returns the next client identifier.
func (k *Keeper) GenerateClientIdentifier(ctx sdk.Context, clientType string) string {
	nextClientSeq := k.GetNextClientSequence(ctx)
	clientID := types.FormatClientIdentifier(clientType, nextClientSeq)

	nextClientSeq++
	k.SetNextClientSequence(ctx, nextClientSeq)
	return clientID
}

// GetClientState gets a particular client from the store
func (k *Keeper) GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool) {
	store := k.ClientStore(ctx, clientID)
	bz := store.Get(host.ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}

	clientState := types.MustUnmarshalClientState(k.cdc, bz)
	return clientState, true
}

// SetClientState sets a particular Client to the store
func (k *Keeper) SetClientState(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	store := k.ClientStore(ctx, clientID)
	store.Set(host.ClientStateKey(), types.MustMarshalClientState(k.cdc, clientState))
}

// GetClientCreator returns the creator of a client
func (k *Keeper) GetClientCreator(ctx sdk.Context, clientID string) sdk.AccAddress {
	store := k.ClientStore(ctx, clientID)
	bz := store.Get(types.CreatorKey())
	if len(bz) == 0 {
		return nil
	}
	return sdk.AccAddress(bz)
}

// SetClientCreator sets the creator of a client
func (k *Keeper) SetClientCreator(ctx sdk.Context, clientID string, creator sdk.AccAddress) {
	store := k.ClientStore(ctx, clientID)
	store.Set(types.CreatorKey(), creator.Bytes())
}

// DeleteClientCreator deletes the creator of a client
func (k *Keeper) DeleteClientCreator(ctx sdk.Context, clientID string) {
	store := k.ClientStore(ctx, clientID)
	store.Delete(types.CreatorKey())
}

// GetClientConsensusState gets the stored consensus state from a client at a given height.
func (k *Keeper) GetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) (exported.ConsensusState, bool) {
	store := k.ClientStore(ctx, clientID)
	bz := store.Get(host.ConsensusStateKey(height))
	if len(bz) == 0 {
		return nil, false
	}

	consensusState := types.MustUnmarshalConsensusState(k.cdc, bz)
	return consensusState, true
}

// SetClientConsensusState sets a ConsensusState to a particular client at the given
// height
func (k *Keeper) SetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height, consensusState exported.ConsensusState) {
	store := k.ClientStore(ctx, clientID)
	store.Set(host.ConsensusStateKey(height), types.MustMarshalConsensusState(k.cdc, consensusState))
}

// GetNextClientSequence gets the next client sequence from the store.
func (k *Keeper) GetNextClientSequence(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get([]byte(types.KeyNextClientSequence))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		panic(errors.New("next client sequence is nil"))
	}

	return sdk.BigEndianToUint64(bz)
}

// SetNextClientSequence sets the next client sequence to the store.
func (k *Keeper) SetNextClientSequence(ctx sdk.Context, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set([]byte(types.KeyNextClientSequence), bz); err != nil {
		panic(err)
	}
}

// IterateConsensusStates provides an iterator over all stored consensus states.
// objects. For each State object, cb will be called. If the cb returns true,
// the iterator will close and stop.
func (k *Keeper) IterateConsensusStates(ctx sdk.Context, cb func(clientID string, cs types.ConsensusStateWithHeight) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, host.KeyClientStorePrefix)

	defer sdk.LogDeferred(k.Logger(ctx), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		// consensus key is in the format "clients/<clientID>/consensusStates/<height>"
		if len(keySplit) != 4 || keySplit[2] != string(host.KeyConsensusStatePrefix) {
			continue
		}
		clientID := keySplit[1]
		height := types.MustParseHeight(keySplit[3])
		consensusState := types.MustUnmarshalConsensusState(k.cdc, iterator.Value())

		consensusStateWithHeight := types.NewConsensusStateWithHeight(height, consensusState)

		if cb(clientID, consensusStateWithHeight) {
			break
		}
	}
}

// iterateMetadata provides an iterator over all stored metadata keys in the client store.
// For each metadata object, it will perform a callback.
func (k *Keeper) iterateMetadata(ctx sdk.Context, cb func(clientID string, key, value []byte) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, host.KeyClientStorePrefix)

	defer sdk.LogDeferred(k.Logger(ctx), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		split := strings.Split(string(iterator.Key()), "/")
		if len(split) == 3 && split[2] == string(host.KeyClientState) {
			// skip client state keys
			continue
		}

		if len(split) == 4 && split[2] == string(host.KeyConsensusStatePrefix) {
			// skip consensus state keys
			continue
		}

		if split[0] != string(host.KeyClientStorePrefix) {
			panic(errorsmod.Wrapf(host.ErrInvalidPath, "path does not begin with client store prefix: expected %s, got %s", host.KeyClientStorePrefix, split[0]))
		}
		if strings.TrimSpace(split[1]) == "" {
			panic(errorsmod.Wrap(host.ErrInvalidPath, "clientID is empty"))
		}

		clientID := split[1]

		key := []byte(strings.Join(split[2:], "/"))

		if cb(clientID, key, iterator.Value()) {
			break
		}
	}
}

// GetAllGenesisClients returns all the clients in state with their client ids returned as IdentifiedClientState
func (k *Keeper) GetAllGenesisClients(ctx sdk.Context) types.IdentifiedClientStates {
	var genClients types.IdentifiedClientStates
	k.IterateClientStates(ctx, nil, func(clientID string, cs exported.ClientState) bool {
		genClients = append(genClients, types.NewIdentifiedClientState(clientID, cs))
		return false
	})

	return genClients.Sort()
}

// GetAllClientMetadata will take a list of IdentifiedClientState and return a list
// of IdentifiedGenesisMetadata necessary for exporting and importing client metadata
// into the client store.
func (k *Keeper) GetAllClientMetadata(ctx sdk.Context, genClients []types.IdentifiedClientState) ([]types.IdentifiedGenesisMetadata, error) {
	metadataMap := make(map[string][]types.GenesisMetadata)
	k.iterateMetadata(ctx, func(clientID string, key, value []byte) bool {
		metadataMap[clientID] = append(metadataMap[clientID], types.NewGenesisMetadata(key, value))
		return false
	})

	genMetadata := make([]types.IdentifiedGenesisMetadata, 0)
	for _, ic := range genClients {
		metadata := metadataMap[ic.ClientId]
		if len(metadata) != 0 {
			genMetadata = append(genMetadata, types.NewIdentifiedGenesisMetadata(
				ic.ClientId,
				metadata,
			))
		}
	}

	return genMetadata, nil
}

// SetAllClientMetadata takes a list of IdentifiedGenesisMetadata and stores all of the metadata in the client store at the appropriate paths.
func (k *Keeper) SetAllClientMetadata(ctx sdk.Context, genMetadata []types.IdentifiedGenesisMetadata) {
	for _, igm := range genMetadata {
		// create client store
		store := k.ClientStore(ctx, igm.ClientId)
		// set all metadata kv pairs in client store
		for _, md := range igm.ClientMetadata {
			store.Set(md.GetKey(), md.GetValue())
		}
	}
}

// GetAllConsensusStates returns all stored client consensus states.
func (k *Keeper) GetAllConsensusStates(ctx sdk.Context) types.ClientsConsensusStates {
	clientConsStates := make(types.ClientsConsensusStates, 0)
	mapClientIDToConsStateIdx := make(map[string]int)

	k.IterateConsensusStates(ctx, func(clientID string, cs types.ConsensusStateWithHeight) bool {
		idx, ok := mapClientIDToConsStateIdx[clientID]
		if ok {
			clientConsStates[idx].ConsensusStates = append(clientConsStates[idx].ConsensusStates, cs)
			return false
		}

		clientConsState := types.ClientConsensusStates{
			ClientId:        clientID,
			ConsensusStates: []types.ConsensusStateWithHeight{cs},
		}

		clientConsStates = append(clientConsStates, clientConsState)
		mapClientIDToConsStateIdx[clientID] = len(clientConsStates) - 1
		return false
	})

	return clientConsStates.Sort()
}

// HasClientConsensusState returns if keeper has a ConsensusState for a particular
// client at the given height
func (k *Keeper) HasClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) bool {
	store := k.ClientStore(ctx, clientID)
	return store.Has(host.ConsensusStateKey(height))
}

// GetLatestClientConsensusState gets the latest ConsensusState stored for a given client
func (k *Keeper) GetLatestClientConsensusState(ctx sdk.Context, clientID string) (exported.ConsensusState, bool) {
	clientModule, err := k.Route(ctx, clientID)
	if err != nil {
		return nil, false
	}

	return k.GetClientConsensusState(ctx, clientID, clientModule.LatestHeight(ctx, clientID))
}

// VerifyMembership retrieves the light client module for the clientID and verifies the proof of the existence of a key-value pair at a specified height.
func (k *Keeper) VerifyMembership(ctx sdk.Context, clientID string, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path, value []byte) error {
	clientModule, err := k.Route(ctx, clientID)
	if err != nil {
		return err
	}

	if status := clientModule.Status(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "cannot call verify membership on client (%s) with status %s", clientID, status)
	}

	return clientModule.VerifyMembership(ctx, clientID, height, delayTimePeriod, delayBlockPeriod, proof, path, value)
}

// VerifyNonMembership retrieves the light client module for the clientID and verifies the absence of a given key at a specified height.
func (k *Keeper) VerifyNonMembership(ctx sdk.Context, clientID string, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path) error {
	clientModule, err := k.Route(ctx, clientID)
	if err != nil {
		return err
	}

	if status := clientModule.Status(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "cannot call verify non membership on client (%s) with status %s", clientID, status)
	}

	return clientModule.VerifyNonMembership(ctx, clientID, height, delayTimePeriod, delayBlockPeriod, proof, path)
}

// GetUpgradePlan executes the upgrade keeper GetUpgradePlan function.
func (k *Keeper) GetUpgradePlan(ctx sdk.Context) (upgradetypes.Plan, error) {
	return k.upgradeKeeper.GetUpgradePlan(ctx)
}

// GetUpgradedClient executes the upgrade keeper GetUpgradeClient function.
func (k *Keeper) GetUpgradedClient(ctx sdk.Context, planHeight int64) ([]byte, error) {
	return k.upgradeKeeper.GetUpgradedClient(ctx, planHeight)
}

// GetUpgradedConsensusState returns the upgraded consensus state
func (k *Keeper) GetUpgradedConsensusState(ctx sdk.Context, planHeight int64) ([]byte, error) {
	return k.upgradeKeeper.GetUpgradedConsensusState(ctx, planHeight)
}

// SetUpgradedConsensusState executes the upgrade keeper SetUpgradedConsensusState function.
func (k *Keeper) SetUpgradedConsensusState(ctx sdk.Context, planHeight int64, bz []byte) error {
	return k.upgradeKeeper.SetUpgradedConsensusState(ctx, planHeight, bz)
}

// IterateClientStates provides an iterator over all stored ibc ClientState
// objects using the provided store prefix. For each ClientState object, cb will be called. If the cb returns true,
// the iterator will close and stop.
func (k *Keeper) IterateClientStates(ctx sdk.Context, storePrefix []byte, cb func(clientID string, cs exported.ClientState) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, host.PrefixedClientStoreKey(storePrefix))

	defer sdk.LogDeferred(k.Logger(ctx), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		path := string(iterator.Key())
		if !strings.Contains(path, host.KeyClientState) {
			// skip non client state keys
			continue
		}

		clientID := host.MustParseClientStatePath(path)
		clientState := types.MustUnmarshalClientState(k.cdc, iterator.Value())

		if cb(clientID, clientState) {
			break
		}
	}
}

// GetAllClients returns all stored light client State objects.
func (k *Keeper) GetAllClients(ctx sdk.Context) []exported.ClientState {
	var states []exported.ClientState
	k.IterateClientStates(ctx, nil, func(_ string, state exported.ClientState) bool {
		states = append(states, state)
		return false
	})

	return states
}

// ClientStore returns isolated prefix store for each client so they can read/write in separate
// namespace without being able to read/write other client's data
func (k *Keeper) ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore {
	clientPrefix := fmt.Appendf(nil, "%s/%s/", host.KeyClientStorePrefix, clientID)
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(store, clientPrefix)
}

// GetClientStatus returns the status for a client state  given a client identifier. If the client type is not in the allowed
// clients param field, Unauthorized is returned, otherwise the client state status is returned.
func (k *Keeper) GetClientStatus(ctx sdk.Context, clientID string) exported.Status {
	clientModule, err := k.Route(ctx, clientID)
	if err != nil {
		return exported.Unauthorized
	}

	return clientModule.Status(ctx, clientID)
}

// GetClientLatestHeight returns the latest height of a client state for a given client identifier. If the client type is not in the allowed
// clients param field, a zero value height is returned, otherwise the client state latest height is returned.
func (k *Keeper) GetClientLatestHeight(ctx sdk.Context, clientID string) types.Height {
	clientModule, err := k.Route(ctx, clientID)
	if err != nil {
		return types.ZeroHeight()
	}

	var latestHeight types.Height
	latestHeight, ok := clientModule.LatestHeight(ctx, clientID).(types.Height)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to %T", clientModule.LatestHeight, latestHeight))
	}
	return latestHeight
}

// GetClientTimestampAtHeight returns the timestamp in nanoseconds of the consensus state at the given height.
func (k *Keeper) GetClientTimestampAtHeight(ctx sdk.Context, clientID string, height exported.Height) (uint64, error) {
	clientModule, err := k.Route(ctx, clientID)
	if err != nil {
		return 0, err
	}

	return clientModule.TimestampAtHeight(ctx, clientID, height)
}

// GetParams returns the total set of ibc-client parameters.
func (k *Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get([]byte(types.ParamsKey))
	if err != nil {
		panic(err)
	}
	if bz == nil { // only panic on unset params and not on empty params
		panic(errors.New("client params are not set in store"))
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the total set of ibc-client parameters.
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&params)
	if err := store.Set([]byte(types.ParamsKey), bz); err != nil {
		panic(err)
	}
}

// ScheduleIBCSoftwareUpgrade schedules an upgrade for the IBC client.
func (k *Keeper) ScheduleIBCSoftwareUpgrade(ctx sdk.Context, plan upgradetypes.Plan, upgradedClientState exported.ClientState) error {
	// zero out any custom fields before setting
	cs, ok := upgradedClientState.(*ibctm.ClientState)
	if !ok {
		return errorsmod.Wrapf(types.ErrInvalidClientType, "expected: %T, got: %T", &ibctm.ClientState{}, upgradedClientState)
	}

	cs = cs.ZeroCustomFields()
	bz, err := types.MarshalClientState(k.cdc, cs)
	if err != nil {
		return errorsmod.Wrap(err, "could not marshal UpgradedClientState")
	}

	if err := k.upgradeKeeper.ScheduleUpgrade(ctx, plan); err != nil {
		return err
	}

	// sets the new upgraded client last height committed on this chain at plan.Height,
	// since the chain will panic at plan.Height and new chain will resume at plan.Height
	if err = k.upgradeKeeper.SetUpgradedClient(ctx, plan.Height, bz); err != nil {
		return err
	}

	// emitting an event for scheduling an upgrade plan
	emitScheduleIBCSoftwareUpgradeEvent(ctx, plan.Name, plan.Height)

	return nil
}
