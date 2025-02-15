package solomachine

import (
	"reflect"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface
type LightClientModule struct {
	cdc           codec.BinaryCodec
	storeProvider clienttypes.StoreProvider
}

// NewLightClientModule creates and returns a new 06-solomachine LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec, storeProvider clienttypes.StoreProvider) LightClientModule {
	return LightClientModule{
		cdc:           cdc,
		storeProvider: storeProvider,
	}
}

// Initialize unmarshals the provided client and consensus states and performs basic validation. It calls into the
// clientState.Initialize method.
func (l LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	var clientState ClientState
	if err := l.cdc.Unmarshal(clientStateBz, &clientState); err != nil {
		return err
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	var consensusState ConsensusState
	if err := l.cdc.Unmarshal(consensusStateBz, &consensusState); err != nil {
		return err
	}

	if err := consensusState.ValidateBasic(); err != nil {
		return err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	if !reflect.DeepEqual(clientState.ConsensusState, &consensusState) {
		return errorsmod.Wrapf(clienttypes.ErrInvalidConsensus, "consensus state in initial client does not equal initial consensus state. expected: %s, got: %s",
			clientState.ConsensusState, &consensusState)
	}

	setClientState(clientStore, l.cdc, &clientState)

	return nil
}

// VerifyClientMessage obtains the client state associated with the client identifier and calls into the clientState.VerifyClientMessage method.
func (l LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyClientMessage(ctx, l.cdc, clientStore, clientMsg)
}

// CheckForMisbehaviour obtains the client state associated with the client identifier and calls into the clientState.CheckForMisbehaviour method.
func (l LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.CheckForMisbehaviour(ctx, l.cdc, clientStore, clientMsg)
}

// UpdateStateOnMisbehaviour updates state upon misbehaviour, freezing the ClientState.
// This method should only be called when misbehaviour is detected as it does not perform
// any misbehaviour checks.
func (l LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	clientState.IsFrozen = true
	setClientState(clientStore, l.cdc, clientState)
}

// UpdateState obtains the client state associated with the client identifier and calls into the clientState.UpdateState method.
func (l LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) []exported.Height {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.UpdateState(ctx, l.cdc, clientStore, clientMsg)
}

// VerifyMembership obtains the client state associated with the client identifier and calls into the clientState.verifyMembership method.
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

	return clientState.verifyMembership(clientStore, l.cdc, proof, path, value)
}

// VerifyNonMembership obtains the client state associated with the client identifier and calls into the clientState.verifyNonMembership method.
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
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.verifyNonMembership(clientStore, l.cdc, proof, path)
}

// Status returns the status of the solo machine client.
// The client may be:
// - Active: if `IsFrozen` is false.
// - Frozen: if `IsFrozen` is true.
// - Unknown: if the client state associated with the provided client identifier is not found.
func (l LightClientModule) Status(ctx sdk.Context, clientID string) exported.Status {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return exported.Unknown
	}

	if clientState.IsFrozen {
		return exported.Frozen
	}

	return exported.Active
}

// LatestHeight returns the latest height for the client state for the given client identifier.
// If no client is present for the provided client identifier a zero value height is returned.
// NOTE: RevisionNumber is always 0 for solomachine client heights.
func (l LightClientModule) LatestHeight(ctx sdk.Context, clientID string) exported.Height {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return clienttypes.ZeroHeight()
	}

	return clienttypes.NewHeight(0, clientState.Sequence)
}

// TimestampAtHeight obtains the client state associated with the client identifier and returns the timestamp in nanoseconds of the consensus state at the given height.
func (l LightClientModule) TimestampAtHeight(ctx sdk.Context, clientID string, height exported.Height) (uint64, error) {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.ConsensusState.Timestamp, nil
}

// RecoverClient asserts that the substitute client is a solo machine client. It obtains the client state associated with the
// subject client and calls into the subjectClientState.CheckSubstituteAndUpdateState method.
func (l LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	substituteClientType, _, err := clienttypes.ParseClientIdentifier(substituteClientID)
	if err != nil {
		return err
	}

	if substituteClientType != exported.Solomachine {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Solomachine, substituteClientType)
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

	return clientState.CheckSubstituteAndUpdateState(ctx, l.cdc, clientStore, substituteClientStore, substituteClient)
}

// VerifyUpgradeAndUpdateState returns an error since solomachine client does not support upgrades
func (LightClientModule) VerifyUpgradeAndUpdateState(ctx sdk.Context, clientID string, newClient, newConsState, upgradeClientProof, upgradeConsensusStateProof []byte) error {
	return errorsmod.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade solomachine client")
}
