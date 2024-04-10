package celestia

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	cdc           codec.BinaryCodec
	storeProvider exported.ClientStoreProvider
}

// NewLightClientModule creates and returns a new 07-celestia LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec) LightClientModule {
	return LightClientModule{
		cdc: cdc,
	}
}

// RegisterStoreProvider is called by core IBC when a LightClientModule is added to the router.
// It allows the LightClientModule to set a ClientStoreProvider which supplies isolated prefix client stores
// to IBC light client instances.
func (l *LightClientModule) RegisterStoreProvider(storeProvider exported.ClientStoreProvider) {
	l.storeProvider = storeProvider
}

// Initialize unmarshals the provided client and consensus states and performs basic validation. It calls into the
// clientState.Initialize method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	var clientState ClientState
	if err := l.cdc.Unmarshal(clientStateBz, &clientState); err != nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, err.Error())
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	var consensusState ibctm.ConsensusState
	if err := l.cdc.Unmarshal(consensusStateBz, &consensusState); err != nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, err.Error())
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	return clientState.BaseClient.Initialize(ctx, l.cdc, clientStore, &consensusState)
}

// VerifyClientMessage obtains the client state associated with the client identifier and calls into the clientState.VerifyClientMessage method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.BaseClient.VerifyClientMessage(ctx, l.cdc, clientStore, clientMsg)
}

// CheckForMisbehaviour obtains the client state associated with the client identifier and calls into the clientState.CheckForMisbehaviour method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.BaseClient.CheckForMisbehaviour(ctx, l.cdc, clientStore, clientMsg)
}

// UpdateStateOnMisbehaviour obtains the client state associated with the client identifier and calls into the clientState.UpdateStateOnMisbehaviour method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	clientState.BaseClient.UpdateStateOnMisbehaviour(ctx, l.cdc, clientStore, clientMsg)
}

// UpdateState obtains the client state associated with the client identifier and calls into the clientState.UpdateState method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) []exported.Height {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	// execute custom 07-celestia update state logic
	return clientState.UpdateState(ctx, l.cdc, clientStore, clientMsg)
}

// VerifyMembership obtains the client state associated with the client identifier and calls into the clientState.VerifyMembership method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
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
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	// execute custom 07-celestia verify membership logic
	return clientState.VerifyMembership(ctx, clientStore, l.cdc, height, delayTimePeriod, delayBlockPeriod, proof, path, value)
}

// VerifyNonMembership obtains the client state associated with the client identifier and calls into the clientState.VerifyNonMembership method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (LightClientModule) VerifyNonMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	panic("07-celestia light clients do not verify non-membership proofs")
}

// Status obtains the client state associated with the client identifier and calls into the clientState.Status method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) Status(ctx sdk.Context, clientID string) exported.Status {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return exported.Unknown
	}

	return clientState.BaseClient.Status(ctx, clientStore, l.cdc)
}

// LatestHeight returns the latest height for the client state for the given client identifier.
// If no client is present for the provided client identifier a zero value height is returned.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) LatestHeight(ctx sdk.Context, clientID string) exported.Height {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return clienttypes.ZeroHeight()
	}

	return clientState.BaseClient.LatestHeight
}

// TimestampAtHeight obtains the client state associated with the client identifier and calls into the clientState.GetTimestampAtHeight method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) TimestampAtHeight(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
) (uint64, error) {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.BaseClient.GetTimestampAtHeight(ctx, clientStore, l.cdc, height)
}

// RecoverClient asserts that the substitute client is a celestia client. It obtains the client state associated with the
// subject client and calls into the subjectClientState.CheckSubstituteAndUpdateState method.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (l LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	substituteClientType, _, err := clienttypes.ParseClientIdentifier(substituteClientID)
	if err != nil {
		return err
	}

	if substituteClientType != ModuleName {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", ModuleName, substituteClientType)
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	substituteClientStore := l.storeProvider.ClientStore(ctx, substituteClientID)
	substituteClient, found := getClientState(substituteClientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, substituteClientID)
	}

	return clientState.BaseClient.CheckSubstituteAndUpdateState(ctx, l.cdc, clientStore, substituteClientStore, substituteClient.BaseClient)
}

// VerifyUpgradeAndUpdateState obtains the client state associated with the client identifier and calls into the clientState.VerifyUpgradeAndUpdateState method.
// The new client and consensus states will be unmarshaled and an error is returned if the new client state is not at a height greater
// than the existing client.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-celestia-{n}.
func (LightClientModule) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	clientID string,
	newClient []byte,
	newConsState []byte,
	upgradeClientProof,
	upgradeConsensusStateProof []byte,
) error {
	// TODO: do we need to implement this?
	panic("unimplemented")
}
