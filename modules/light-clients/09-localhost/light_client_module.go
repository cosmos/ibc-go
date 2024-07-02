package localhost

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	cdc           codec.BinaryCodec
	key           storetypes.StoreKey
	storeProvider exported.ClientStoreProvider
}

// NewLightClientModule creates and returns a new 09-localhost LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec, key storetypes.StoreKey) *LightClientModule {
	return &LightClientModule{
		cdc: cdc,
		key: key,
	}
}

// RegisterStoreProvider is called by core IBC when a LightClientModule is added to the router.
// It allows the LightClientModule to set a ClientStoreProvider which supplies isolated prefix client stores
// to IBC light client instances.
func (l *LightClientModule) RegisterStoreProvider(storeProvider exported.ClientStoreProvider) {
	l.storeProvider = storeProvider
}

// Initialize ensures that initial consensus state for localhost is nil.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (l LightClientModule) Initialize(ctx sdk.Context, clientID string, _, consensusStateBz []byte) error {
	if len(consensusStateBz) != 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "initial consensus state for localhost must be nil.")
	}

	clientState := ClientState{
		LatestHeight: clienttypes.GetSelfHeight(ctx),
	}

	clientStore := l.storeProvider.ClientStore(ctx, exported.LocalhostClientID)
	clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(l.cdc, &clientState))
	return nil
}

// VerifyClientMessage is unsupported by the 09-localhost client type and returns an error.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	return errorsmod.Wrap(clienttypes.ErrUpdateClientFailed, "client message verification is unsupported by the localhost client")
}

// CheckForMisbehaviour is unsupported by the 09-localhost client type and performs a no-op, returning false.
func (LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	return false
}

// UpdateStateOnMisbehaviour is unsupported by the 09-localhost client type and performs a no-op.
func (LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	// no-op
}

// UpdateState obtains the localhost client state and calls into the clientState.UpdateState method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (l LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) []exported.Height {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	cdc := l.cdc

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.UpdateState(ctx, cdc, clientStore, clientMsg)
}

// VerifyMembership obtains the localhost client state and calls into the clientState.VerifyMembership method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (l LightClientModule) VerifyMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	ibcStore := ctx.KVStore(l.key)
	cdc := l.cdc

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyMembership(ctx, ibcStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path, value)
}

// VerifyNonMembership obtains the localhost client state and calls into the clientState.VerifyNonMembership method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (l LightClientModule) VerifyNonMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	ibcStore := ctx.KVStore(l.key)
	cdc := l.cdc

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyNonMembership(ctx, ibcStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path)
}

// Status always returns Active. The 09-localhost status cannot be changed.
func (LightClientModule) Status(ctx sdk.Context, clientID string) exported.Status {
	return exported.Active
}

// LatestHeight returns the latest height for the client state for the given client identifier.
// If no client is present for the provided client identifier a zero value height is returned.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (l LightClientModule) LatestHeight(ctx sdk.Context, clientID string) exported.Height {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return clienttypes.ZeroHeight()
	}

	return clientState.LatestHeight
}

// TimestampAtHeight returns the current block time retrieved from the application context. The localhost client does not store consensus states and thus
// cannot provide a timestamp for the provided height.
func (LightClientModule) TimestampAtHeight(ctx sdk.Context, clientID string, height exported.Height) (uint64, error) {
	return uint64(ctx.BlockTime().UnixNano()), nil
}

// RecoverClient returns an error. The localhost cannot be modified by proposals.
func (LightClientModule) RecoverClient(_ sdk.Context, _, _ string) error {
	return errorsmod.Wrap(clienttypes.ErrUpdateClientFailed, "cannot update localhost client with a proposal")
}

// VerifyUpgradeAndUpdateState returns an error since localhost cannot be upgraded.
func (LightClientModule) VerifyUpgradeAndUpdateState(ctx sdk.Context, clientID string, newClient, newConsState, upgradeClientProof, upgradeConsensusStateProof []byte) error {
	return errorsmod.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade localhost client")
}
