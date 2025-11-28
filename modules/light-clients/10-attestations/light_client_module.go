package attestations

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	cdc           codec.BinaryCodec
	storeProvider clienttypes.StoreProvider
}

// NewLightClientModule creates and returns a new 10-attestations LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec, storeProvider clienttypes.StoreProvider) LightClientModule {
	return LightClientModule{
		cdc:           cdc,
		storeProvider: storeProvider,
	}
}

// Initialize unmarshals the provided client and consensus states and performs basic validation.
func (l LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	var clientState ClientState
	if err := l.cdc.Unmarshal(clientStateBz, &clientState); err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal client state bytes into client state")
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	var consensusState ConsensusState
	if err := l.cdc.Unmarshal(consensusStateBz, &consensusState); err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal consensus state bytes into consensus state")
	}

	if err := consensusState.ValidateBasic(); err != nil {
		return err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	if clientState.LatestHeight == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "initial height must be non-zero")
	}

	initialHeight := clienttypes.NewHeight(0, clientState.LatestHeight)
	setConsensusState(clientStore, l.cdc, &consensusState, initialHeight)
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

// CheckForMisbehaviour is not supported in this version.
func (LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	panic(errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "checkForMisbehaviour is not supported"))
}

// UpdateStateOnMisbehaviour is not supported in this version.
func (LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	panic(errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "updateStateOnMisbehaviour is not supported"))
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

	return clientState.verifyMembership(clientStore, l.cdc, height, proof, path, value)
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

	return clientState.verifyNonMembership(clientStore, l.cdc, height, proof, path)
}

// Status returns the status of the attestations client.
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
// NOTE: RevisionNumber is always 0 for attestations client heights.
func (l LightClientModule) LatestHeight(ctx sdk.Context, clientID string) exported.Height {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)

	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return clienttypes.ZeroHeight()
	}

	return clienttypes.NewHeight(0, clientState.LatestHeight)
}

// TimestampAtHeight obtains the client state associated with the client identifier and returns the timestamp in nanoseconds of the consensus state at the given height.
func (l LightClientModule) TimestampAtHeight(ctx sdk.Context, clientID string, height exported.Height) (uint64, error) {
	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	consensusState, found := getConsensusState(clientStore, l.cdc, height)
	if !found {
		return 0, errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "consensus state not found for height %s", height)
	}

	return consensusState.Timestamp, nil
}

// RecoverClient is not supported in this version.
func (LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	return errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "recoverClient is not supported")
}

// VerifyUpgradeAndUpdateState returns an error since attestations client does not support upgrades.
func (LightClientModule) VerifyUpgradeAndUpdateState(ctx sdk.Context, clientID string, newClient, newConsState, upgradeClientProof, upgradeConsensusStateProof []byte) error {
	return errorsmod.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade attestations client")
}
