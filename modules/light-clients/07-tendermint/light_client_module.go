package tendermint

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint/internal/keeper"
)

var _ exported.LightClientModule = (*LightClientModule)(nil)

type LightClientModule struct {
	keeper        keeper.Keeper
	storeProvider exported.ClientStoreProvider
}

func NewLightClientModule(cdc codec.BinaryCodec, authority string) LightClientModule {
	return LightClientModule{
		keeper: keeper.NewKeeper(cdc, authority),
	}
}

func (lcm *LightClientModule) RegisterStoreProvider(storeProvider exported.ClientStoreProvider) {
	lcm.storeProvider = storeProvider
}

// Initialize checks that the initial consensus state is an 07-tendermint consensus state and
// sets the client state, consensus state and associated metadata in the provided client store.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	var clientState ClientState
	if err := lcm.keeper.Codec().Unmarshal(clientStateBz, &clientState); err != nil {
		return err
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	var consensusState ConsensusState
	if err := lcm.keeper.Codec().Unmarshal(consensusStateBz, &consensusState); err != nil {
		return err
	}

	if err := consensusState.ValidateBasic(); err != nil {
		return err
	}

	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)

	return clientState.Initialize(ctx, lcm.keeper.Codec(), clientStore, &consensusState)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyClientMessage(ctx, cdc, clientStore, clientMsg)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.CheckForMisbehaviour(ctx, cdc, clientStore, clientMsg)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	clientState.UpdateStateOnMisbehaviour(ctx, cdc, clientStore, clientMsg)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) []exported.Height {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.UpdateState(ctx, cdc, clientStore, clientMsg)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) VerifyMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyMembership(ctx, clientStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path, value)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) VerifyNonMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height, // TODO: change to concrete type
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path, // TODO: change to conrete type
) error {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyNonMembership(ctx, clientStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) Status(ctx sdk.Context, clientID string) exported.Status {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return exported.Unknown
	}

	return clientState.Status(ctx, clientStore, cdc)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) TimestampAtHeight(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
) (uint64, error) {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.GetTimestampAtHeight(ctx, clientStore, cdc, height)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	substituteClientType, _, err := clienttypes.ParseClientIdentifier(substituteClientID)
	if err != nil {
		return err
	}

	if substituteClientType != exported.Tendermint {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, substituteClientType)
	}

	cdc := lcm.keeper.Codec()

	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	substituteClientStore := lcm.storeProvider.ClientStore(ctx, substituteClientID)
	substituteClient, found := getClientState(substituteClientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, substituteClientID)
	}

	if clientState.GetLatestHeight().GTE(substituteClient.GetLatestHeight()) {
		return errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "subject client state latest height is greater or equal to substitute client state latest height (%s >= %s)", clientState.GetLatestHeight(), substituteClient.GetLatestHeight())
	}

	return clientState.CheckSubstituteAndUpdateState(ctx, cdc, clientStore, substituteClientStore, substituteClient)
}

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	clientID string,
	newClient []byte,
	newConsState []byte,
	upgradeClientProof,
	upgradeConsensusStateProof []byte,
) error {
	var newClientState ClientState
	if err := lcm.keeper.Codec().Unmarshal(newClient, &newClientState); err != nil {
		return err
	}

	if err := newClientState.Validate(); err != nil {
		return err
	}

	var newConsensusState ConsensusState
	if err := lcm.keeper.Codec().Unmarshal(newConsState, &newConsensusState); err != nil {
		return err
	}
	if err := newConsensusState.ValidateBasic(); err != nil {
		return err
	}

	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyUpgradeAndUpdateState(ctx, cdc, clientStore, &newClientState, &newConsensusState, upgradeClientProof, upgradeConsensusStateProof)
}
