package keeper

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/light"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	localhost "github.com/cosmos/ibc-go/v8/modules/light-clients/09-localhost"
)

// Keeper represents a type that grants read and write permissions to any client
// state information
type Keeper struct {
	storeKey       storetypes.StoreKey
	cdc            codec.BinaryCodec
	router         *types.Router
	legacySubspace types.ParamSubspace
	stakingKeeper  types.StakingKeeper
	upgradeKeeper  types.UpgradeKeeper
}

// NewKeeper creates a new NewKeeper instance
func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, legacySubspace types.ParamSubspace, sk types.StakingKeeper, uk types.UpgradeKeeper) Keeper {
	router := types.NewRouter(key)
	localhostModule := localhost.NewLightClientModule(cdc, key)
	router.AddRoute(exported.Localhost, localhostModule)

	return Keeper{
		storeKey:       key,
		cdc:            cdc,
		router:         router,
		legacySubspace: legacySubspace,
		stakingKeeper:  sk,
		upgradeKeeper:  uk,
	}
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"/"+types.SubModuleName)
}

func (k Keeper) GetRouter() *types.Router {
	return k.router
}

// CreateLocalhostClient initialises the 09-localhost client state and sets it in state.
func (k Keeper) CreateLocalhostClient(ctx sdk.Context) error {
	clientModule, found := k.router.GetRoute(exported.LocalhostClientID)
	if !found {
		return errorsmod.Wrap(types.ErrRouteNotFound, exported.LocalhostClientID)
	}

	return clientModule.Initialize(ctx, exported.LocalhostClientID, nil, nil)
}

// UpdateLocalhostClient updates the 09-localhost client to the latest block height and chain ID.
func (k Keeper) UpdateLocalhostClient(ctx sdk.Context, clientState exported.ClientState) []exported.Height {
	clientModule, found := k.router.GetRoute(exported.LocalhostClientID)
	if !found {
		panic(errorsmod.Wrap(types.ErrRouteNotFound, exported.LocalhostClientID))
	}

	return clientModule.UpdateState(ctx, exported.LocalhostClientID, nil)
}

// GenerateClientIdentifier returns the next client identifier.
func (k Keeper) GenerateClientIdentifier(ctx sdk.Context, clientType string) string {
	nextClientSeq := k.GetNextClientSequence(ctx)
	clientID := types.FormatClientIdentifier(clientType, nextClientSeq)

	nextClientSeq++
	k.SetNextClientSequence(ctx, nextClientSeq)
	return clientID
}

// GetClientState gets a particular client from the store
func (k Keeper) GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool) {
	store := k.ClientStore(ctx, clientID)
	bz := store.Get(host.ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}

	clientState := k.MustUnmarshalClientState(bz)
	return clientState, true
}

// SetClientState sets a particular Client to the store
func (k Keeper) SetClientState(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	store := k.ClientStore(ctx, clientID)
	store.Set(host.ClientStateKey(), k.MustMarshalClientState(clientState))
}

// GetClientConsensusState gets the stored consensus state from a client at a given height.
func (k Keeper) GetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) (exported.ConsensusState, bool) {
	store := k.ClientStore(ctx, clientID)
	bz := store.Get(host.ConsensusStateKey(height))
	if len(bz) == 0 {
		return nil, false
	}

	consensusState := k.MustUnmarshalConsensusState(bz)
	return consensusState, true
}

// SetClientConsensusState sets a ConsensusState to a particular client at the given
// height
func (k Keeper) SetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height, consensusState exported.ConsensusState) {
	store := k.ClientStore(ctx, clientID)
	store.Set(host.ConsensusStateKey(height), k.MustMarshalConsensusState(consensusState))
}

// GetNextClientSequence gets the next client sequence from the store.
func (k Keeper) GetNextClientSequence(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.KeyNextClientSequence))
	if len(bz) == 0 {
		panic(errors.New("next client sequence is nil"))
	}

	return sdk.BigEndianToUint64(bz)
}

// SetNextClientSequence sets the next client sequence to the store.
func (k Keeper) SetNextClientSequence(ctx sdk.Context, sequence uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := sdk.Uint64ToBigEndian(sequence)
	store.Set([]byte(types.KeyNextClientSequence), bz)
}

// IterateConsensusStates provides an iterator over all stored consensus states.
// objects. For each State object, cb will be called. If the cb returns true,
// the iterator will close and stop.
func (k Keeper) IterateConsensusStates(ctx sdk.Context, cb func(clientID string, cs types.ConsensusStateWithHeight) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, host.KeyClientStorePrefix)

	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		// consensus key is in the format "clients/<clientID>/consensusStates/<height>"
		if len(keySplit) != 4 || keySplit[2] != string(host.KeyConsensusStatePrefix) {
			continue
		}
		clientID := keySplit[1]
		height := types.MustParseHeight(keySplit[3])
		consensusState := k.MustUnmarshalConsensusState(iterator.Value())

		consensusStateWithHeight := types.NewConsensusStateWithHeight(height, consensusState)

		if cb(clientID, consensusStateWithHeight) {
			break
		}
	}
}

// iterateMetadata provides an iterator over all stored metadata keys in the client store.
// For each metadata object, it will perform a callback.
func (k Keeper) iterateMetadata(ctx sdk.Context, cb func(clientID string, key, value []byte) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, host.KeyClientStorePrefix)

	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })
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
func (k Keeper) GetAllGenesisClients(ctx sdk.Context) types.IdentifiedClientStates {
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
func (k Keeper) GetAllClientMetadata(ctx sdk.Context, genClients []types.IdentifiedClientState) ([]types.IdentifiedGenesisMetadata, error) {
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
func (k Keeper) SetAllClientMetadata(ctx sdk.Context, genMetadata []types.IdentifiedGenesisMetadata) {
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
func (k Keeper) GetAllConsensusStates(ctx sdk.Context) types.ClientsConsensusStates {
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
func (k Keeper) HasClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) bool {
	store := k.ClientStore(ctx, clientID)
	return store.Has(host.ConsensusStateKey(height))
}

// GetLatestClientConsensusState gets the latest ConsensusState stored for a given client
func (k Keeper) GetLatestClientConsensusState(ctx sdk.Context, clientID string) (exported.ConsensusState, bool) {
	clientModule, found := k.router.GetRoute(clientID)
	if !found {
		return nil, false
	}

	return k.GetClientConsensusState(ctx, clientID, clientModule.LatestHeight(ctx, clientID))
}

// GetSelfConsensusState introspects the (self) past historical info at a given height
// and returns the expected consensus state at that height.
// For now, can only retrieve self consensus states for the current revision
func (k Keeper) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	selfHeight, ok := height.(types.Height)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", types.Height{}, height)
	}
	// check that height revision matches chainID revision
	revision := types.ParseChainID(ctx.ChainID())
	if revision != height.GetRevisionNumber() {
		return nil, errorsmod.Wrapf(types.ErrInvalidHeight, "chainID revision number does not match height revision number: expected %d, got %d", revision, height.GetRevisionNumber())
	}
	histInfo, err := k.stakingKeeper.GetHistoricalInfo(ctx, int64(selfHeight.RevisionHeight))
	if err != nil {
		return nil, errorsmod.Wrapf(err, "height %d", selfHeight.RevisionHeight)
	}

	consensusState := &ibctm.ConsensusState{
		Timestamp:          histInfo.Header.Time,
		Root:               commitmenttypes.NewMerkleRoot(histInfo.Header.GetAppHash()),
		NextValidatorsHash: histInfo.Header.NextValidatorsHash,
	}

	return consensusState, nil
}

// ValidateSelfClient validates the client parameters for a client of the running chain
// This function is only used to validate the client state the counterparty stores for this chain
// Client must be in same revision as the executing chain
func (k Keeper) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	tmClient, ok := clientState.(*ibctm.ClientState)
	if !ok {
		return errorsmod.Wrapf(types.ErrInvalidClient, "client must be a Tendermint client, expected: %T, got: %T",
			&ibctm.ClientState{}, tmClient)
	}

	if !tmClient.FrozenHeight.IsZero() {
		return types.ErrClientFrozen
	}

	if ctx.ChainID() != tmClient.ChainId {
		return errorsmod.Wrapf(types.ErrInvalidClient, "invalid chain-id. expected: %s, got: %s",
			ctx.ChainID(), tmClient.ChainId)
	}

	revision := types.ParseChainID(ctx.ChainID())

	// client must be in the same revision as executing chain
	if tmClient.LatestHeight.RevisionNumber != revision {
		return errorsmod.Wrapf(types.ErrInvalidClient, "client is not in the same revision as the chain. expected revision: %d, got: %d",
			tmClient.LatestHeight.RevisionNumber, revision)
	}

	selfHeight := types.NewHeight(revision, uint64(ctx.BlockHeight()))
	if tmClient.LatestHeight.GTE(selfHeight) {
		return errorsmod.Wrapf(types.ErrInvalidClient, "client has LatestHeight %d greater than or equal to chain height %d",
			tmClient.LatestHeight, selfHeight)
	}

	expectedProofSpecs := commitmenttypes.GetSDKSpecs()
	if !reflect.DeepEqual(expectedProofSpecs, tmClient.ProofSpecs) {
		return errorsmod.Wrapf(types.ErrInvalidClient, "client has invalid proof specs. expected: %v got: %v",
			expectedProofSpecs, tmClient.ProofSpecs)
	}

	if err := light.ValidateTrustLevel(tmClient.TrustLevel.ToTendermint()); err != nil {
		return errorsmod.Wrapf(types.ErrInvalidClient, "trust-level invalid: %v", err)
	}

	expectedUbdPeriod, err := k.stakingKeeper.UnbondingTime(ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to retrieve unbonding period")
	}

	if expectedUbdPeriod != tmClient.UnbondingPeriod {
		return errorsmod.Wrapf(types.ErrInvalidClient, "invalid unbonding period. expected: %s, got: %s",
			expectedUbdPeriod, tmClient.UnbondingPeriod)
	}

	if tmClient.UnbondingPeriod < tmClient.TrustingPeriod {
		return errorsmod.Wrapf(types.ErrInvalidClient, "unbonding period must be greater than trusting period. unbonding period (%d) < trusting period (%d)",
			tmClient.UnbondingPeriod, tmClient.TrustingPeriod)
	}

	if len(tmClient.UpgradePath) != 0 {
		// For now, SDK IBC implementation assumes that upgrade path (if defined) is defined by SDK upgrade module
		expectedUpgradePath := []string{upgradetypes.StoreKey, upgradetypes.KeyUpgradedIBCState}
		if !reflect.DeepEqual(expectedUpgradePath, tmClient.UpgradePath) {
			return errorsmod.Wrapf(types.ErrInvalidClient, "upgrade path must be the upgrade path defined by upgrade module. expected %v, got %v",
				expectedUpgradePath, tmClient.UpgradePath)
		}
	}
	return nil
}

// GetUpgradePlan executes the upgrade keeper GetUpgradePlan function.
func (k Keeper) GetUpgradePlan(ctx sdk.Context) (upgradetypes.Plan, error) {
	return k.upgradeKeeper.GetUpgradePlan(ctx)
}

// GetUpgradedClient executes the upgrade keeper GetUpgradeClient function.
func (k Keeper) GetUpgradedClient(ctx sdk.Context, planHeight int64) ([]byte, error) {
	return k.upgradeKeeper.GetUpgradedClient(ctx, planHeight)
}

// GetUpgradedConsensusState returns the upgraded consensus state
func (k Keeper) GetUpgradedConsensusState(ctx sdk.Context, planHeight int64) ([]byte, error) {
	return k.upgradeKeeper.GetUpgradedConsensusState(ctx, planHeight)
}

// SetUpgradedConsensusState executes the upgrade keeper SetUpgradedConsensusState function.
func (k Keeper) SetUpgradedConsensusState(ctx sdk.Context, planHeight int64, bz []byte) error {
	return k.upgradeKeeper.SetUpgradedConsensusState(ctx, planHeight, bz)
}

// IterateClientStates provides an iterator over all stored ibc ClientState
// objects using the provided store prefix. For each ClientState object, cb will be called. If the cb returns true,
// the iterator will close and stop.
func (k Keeper) IterateClientStates(ctx sdk.Context, storePrefix []byte, cb func(clientID string, cs exported.ClientState) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, host.PrefixedClientStoreKey(storePrefix))

	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		path := string(iterator.Key())
		if !strings.Contains(path, host.KeyClientState) {
			// skip non client state keys
			continue
		}

		clientID := host.MustParseClientStatePath(path)
		clientState := k.MustUnmarshalClientState(iterator.Value())

		if cb(clientID, clientState) {
			break
		}
	}
}

// GetAllClients returns all stored light client State objects.
func (k Keeper) GetAllClients(ctx sdk.Context) []exported.ClientState {
	var states []exported.ClientState
	k.IterateClientStates(ctx, nil, func(_ string, state exported.ClientState) bool {
		states = append(states, state)
		return false
	})

	return states
}

// ClientStore returns isolated prefix store for each client so they can read/write in separate
// namespace without being able to read/write other client's data
func (k Keeper) ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore {
	clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
	return prefix.NewStore(ctx.KVStore(k.storeKey), clientPrefix)
}

// GetClientStatus returns the status for a client state  given a client identifier. If the client type is not in the allowed
// clients param field, Unauthorized is returned, otherwise the client state status is returned.
func (k Keeper) GetClientStatus(ctx sdk.Context, clientID string) exported.Status {
	clientType, _, err := types.ParseClientIdentifier(clientID)
	if err != nil {
		return exported.Unauthorized
	}

	if !k.GetParams(ctx).IsAllowedClient(clientType) {
		return exported.Unauthorized
	}

	clientModule, found := k.router.GetRoute(clientID)
	if !found {
		return exported.Unauthorized
	}

	return clientModule.Status(ctx, clientID)
}

// GetLatestHeight returns the latest height of a client state for a given client identifier. If the client type is not in the allowed
// clients param field, a zero value height is returned, otherwise the client state latest height is returned.
func (k Keeper) GetLatestHeight(ctx sdk.Context, clientID string) types.Height {
	clientType, _, err := types.ParseClientIdentifier(clientID)
	if err != nil {
		return types.ZeroHeight()
	}

	if !k.GetParams(ctx).IsAllowedClient(clientType) {
		return types.ZeroHeight()
	}

	clientModule, found := k.router.GetRoute(clientID)
	if !found {
		return types.ZeroHeight()
	}

	return clientModule.LatestHeight(ctx, clientID).(types.Height)
}

// GetTimestampAtHeight returns the timestamp in nanoseconds of the consensus state at the given height.
func (k Keeper) GetTimestampAtHeight(ctx sdk.Context, clientID string, height exported.Height) (uint64, error) {
	clientType, _, err := types.ParseClientIdentifier(clientID)
	if err != nil {
		return 0, errorsmod.Wrapf(types.ErrClientNotFound, "clientID (%s)", clientID)
	}

	if !k.GetParams(ctx).IsAllowedClient(clientType) {
		return 0, errorsmod.Wrapf(types.ErrInvalidClientType, "client state type %s is not registered in the allowlist", clientType)
	}

	clientModule, found := k.router.GetRoute(clientID)
	if !found {
		return 0, errorsmod.Wrap(types.ErrRouteNotFound, clientType)
	}

	return clientModule.TimestampAtHeight(ctx, clientID, height)
}

// GetParams returns the total set of ibc-client parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.ParamsKey))
	if bz == nil { // only panic on unset params and not on empty params
		panic(errors.New("client params are not set in store"))
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the total set of ibc-client parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set([]byte(types.ParamsKey), bz)
}

// ScheduleIBCSoftwareUpgrade schedules an upgrade for the IBC client.
func (k Keeper) ScheduleIBCSoftwareUpgrade(ctx sdk.Context, plan upgradetypes.Plan, upgradedClientState exported.ClientState) error {
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
