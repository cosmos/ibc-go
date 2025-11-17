package cosmosevm

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	cdc           codec.BinaryCodec
	storeProvider clienttypes.StoreProvider
	clientKeeper ClientKeeper
}

// NewLightClientModule creates and returns a new 07-cosmosevm LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec, storeProvider clienttypes.StoreProvider, clientKeeper ClientKeeper) LightClientModule {
	return LightClientModule{
		cdc:           cdc,
		storeProvider: storeProvider,
		clientKeeper:  clientKeeper,
	}
}

// Initialize unmarshals the provided client and consensus states, performs basic validation, and sets them.
func (l LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	var clientState ClientState
	if err := l.cdc.Unmarshal(clientStateBz, &clientState); err != nil {
		return fmt.Errorf("failed to unmarshal client state bytes into client state: %w", err)
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	var consensusState ConsensusState
	if err := l.cdc.Unmarshal(consensusStateBz, &consensusState); err != nil {
		return fmt.Errorf("failed to unmarshal consensus state bytes into consensus state: %w", err)
	}

	if err := consensusState.ValidateBasic(); err != nil {
		return err
	}

	tmClientState, err := l.getTendermintClientState(ctx, clientState.TendermintClientId)
	if err != nil {
		return err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	setClientState(clientStore, l.cdc, &clientState)
	setConsensusState(clientStore, l.cdc, &consensusState, tmClientState.LatestHeight)

	return nil
}

// VerifyClientMessage is not supported for the CosmosEVM light client.
func (l LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	return ErrUpdatesNotAllowed
}

// CheckForMisbehaviour is not supported for the CosmosEVM light client.
func (l LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	panic(ErrUpdatesNotAllowed)
}

// UpdateStateOnMisbehaviour is not supported for the CosmosEVM light client.
func (l LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	panic(ErrUpdatesNotAllowed)
}

// UpdateState is not supported for the CosmosEVM light client.
func (l LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) []exported.Height {
	panic(ErrUpdatesNotAllowed)
}

// RecoverClient is not supported for the CosmosEVM light client.
func (l LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	return ErrUpdatesNotAllowed
}

// VerifyUpgradeAndUpdateState is not supported for the CosmosEVM light client.
func (l LightClientModule) VerifyUpgradeAndUpdateState(_ sdk.Context, _ string, _, _, _, _ []byte) error {
	return ErrUpdatesNotAllowed
}
