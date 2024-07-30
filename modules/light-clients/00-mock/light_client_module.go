package mock

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	cdc           codec.BinaryCodec
	storeProvider exported.ClientStoreProvider
}

// NewLightClientModule creates and returns a new 00-mock LightClientModule.
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

// Initialize sets clientState and consensusState in the clientStore for the mock client.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 00-mock.
func (l LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz_, consensusStateBz []byte) error {
	clientState := ClientState{}
	if err := l.cdc.Unmarshal(clientStateBz_, &clientState); err != nil {
		return err
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	consensusState := ConsensusState{}
	if err := l.cdc.Unmarshal(consensusStateBz, &consensusState); err != nil {
		return err
	}

	if err := consensusState.ValidateBasic(); err != nil {
		return err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	return clientState.Initialize(l.cdc, clientStore, consensusState)
}

// VerifyClientMessage always returns no error. The 00-mock client type does not support verification.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	return nil
}

// CheckForMisbehaviour is unsupported by the 00-mock client type and performs a no-op, returning false.
func (LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	return false
}

// UpdateStateOnMisbehaviour is unsupported by the 00-mock client type and performs a no-op.
func (LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	// no-op
}

// UpdateState just updates the state with no verification or validation.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (l LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) []exported.Height {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	cdc := l.cdc

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.UpdateState(cdc, clientStore, clientMsg)
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
	cdc := l.cdc

	_, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return nil
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
	cdc := l.cdc

	_, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return nil
}

// Status always returns Active. The 09-localhost status cannot be changed.
func (l LightClientModule) Status(ctx sdk.Context, clientID string) exported.Status {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	_, found := getClientState(clientStore, l.cdc)
	if !found {
		return exported.Unknown
	}

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
func (l LightClientModule) TimestampAtHeight(ctx sdk.Context, clientID string, height exported.Height) (uint64, error) {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.GetTimestampAtHeight(clientStore, l.cdc, height)
}

// RecoverClient returns an error. The localhost cannot be modified by proposals.
func (LightClientModule) RecoverClient(_ sdk.Context, _, _ string) error {
	return errorsmod.Wrap(clienttypes.ErrUpdateClientFailed, "cannot update localhost client with a proposal")
}

// VerifyUpgradeAndUpdateState returns an error since localhost cannot be upgraded.
func (LightClientModule) VerifyUpgradeAndUpdateState(ctx sdk.Context, clientID string, newClient, newConsState, upgradeClientProof, upgradeConsensusStateProof []byte) error {
	return errorsmod.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade localhost client")
}
