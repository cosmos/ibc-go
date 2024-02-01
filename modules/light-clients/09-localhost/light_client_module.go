package localhost

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/api"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ api.LightClientModule = (*LightClientModule)(nil)

type LightClientModule struct {
	cdc           codec.BinaryCodec
	key           storetypes.StoreKey
	storeProvider api.ClientStoreProvider
}

func NewLightClientModule(cdc codec.BinaryCodec, key storetypes.StoreKey) *LightClientModule {
	return &LightClientModule{
		cdc: cdc,
		key: key,
	}
}

func (l *LightClientModule) RegisterStoreProvider(storeProvider api.ClientStoreProvider) {
	l.storeProvider = storeProvider
}

func (l LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Localhost {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Localhost, clientType)
	}

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

func (l LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg api.ClientMessage) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Localhost {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Localhost, clientType)
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	cdc := l.cdc

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyClientMessage(ctx, cdc, clientStore, clientMsg)
}

func (l LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg api.ClientMessage) bool {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		panic(err)
	}

	if clientType != exported.Localhost {
		panic(errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Localhost, clientType))
	}

	return false
}

func (l LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg api.ClientMessage) {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		panic(err)
	}

	if clientType != exported.Localhost {
		panic(errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Localhost, clientType))
	}
}

func (l LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg api.ClientMessage) []api.Height {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		panic(err)
	}

	if clientType != exported.Localhost {
		panic(errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Localhost, clientType))
	}
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	cdc := l.cdc

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.UpdateState(ctx, cdc, clientStore, clientMsg)
}

func (l LightClientModule) VerifyMembership(
	ctx sdk.Context,
	clientID string,
	height api.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path api.Path,
	value []byte,
) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Localhost {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Localhost, clientType)
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	ibcStore := ctx.KVStore(l.key)
	cdc := l.cdc

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyMembership(ctx, ibcStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path, value)
}

func (l LightClientModule) VerifyNonMembership(
	ctx sdk.Context,
	clientID string,
	height api.Height, // TODO: change to concrete type
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path api.Path, // TODO: change to conrete type
) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Localhost {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Localhost, clientType)
	}

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
func (l LightClientModule) Status(ctx sdk.Context, clientID string) api.Status {
	return api.Active
}

func (l LightClientModule) TimestampAtHeight(
	ctx sdk.Context,
	clientID string,
	height api.Height,
) (uint64, error) {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return 0, err
	}

	if clientType != exported.Localhost {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Localhost, clientType)
	}

	return uint64(ctx.BlockTime().UnixNano()), nil
}

func (l LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	return errorsmod.Wrap(clienttypes.ErrUpdateClientFailed, "cannot update localhost client with a proposal")
}

func (l LightClientModule) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	clientID string,
	newClient []byte,
	newConsState []byte,
	upgradeClientProof,
	upgradeConsensusStateProof []byte,
) error {
	return errorsmod.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade localhost client")
}
