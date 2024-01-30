package tendermint

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint/internal/keeper"
)

type LightClientModule struct {
	keeper keeper.Keeper
}

func NewLightClientModule(cdc codec.BinaryCodec, key storetypes.StoreKey, authority string) LightClientModule {
	return LightClientModule{
		keeper: keeper.NewKeeper(cdc, key, authority),
	}
}

// Initialize checks that the initial consensus state is an 07-tendermint consensus state and
// sets the client state, consensus state and associated metadata in the provided client store.
func (lcm LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Tendermint {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType)
	}

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

	clientStore := lcm.keeper.ClientStore(ctx, clientID)

	return clientState.Initialize(ctx, lcm.keeper.Codec(), clientStore, &consensusState)
}

func (lcm LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Tendermint {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType)
	}

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyClientMessage(ctx, cdc, clientStore, clientMsg)
}

func (lcm LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		panic(err)
	}

	if clientType != exported.Tendermint {
		panic(errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType))
	}

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.CheckForMisbehaviour(ctx, cdc, clientStore, clientMsg)
}

func (lcm LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		panic(err)
	}

	if clientType != exported.Tendermint {
		panic(errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType))
	}

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	clientState.UpdateStateOnMisbehaviour(ctx, cdc, clientStore, clientMsg)
}

func (lcm LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) []exported.Height {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		panic(err)
	}

	if clientType != exported.Tendermint {
		panic(errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType))
	}
	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.UpdateState(ctx, cdc, clientStore, clientMsg)
}

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
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Tendermint {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType)
	}

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyMembership(ctx, clientStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path, value)
}

func (lcm LightClientModule) VerifyNonMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height, // TODO: change to concrete type
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path, // TODO: change to conrete type
) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Tendermint {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType)
	}

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyNonMembership(ctx, clientStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path)
}

func (lcm LightClientModule) Status(ctx sdk.Context, clientID string) exported.Status {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return exported.Unknown
	}

	if clientType != exported.Tendermint {
		return exported.Unknown
	}

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return exported.Unknown
	}

	return clientState.Status(ctx, clientStore, cdc)
}

func (lcm LightClientModule) TimestampAtHeight(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
) (uint64, error) {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return 0, err
	}

	if clientType != exported.Tendermint {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType)
	}

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.GetTimestampAtHeight(ctx, clientStore, cdc, height)
}

func (lcm LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Tendermint {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType)
	}

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	substituteClientStore := lcm.keeper.ClientStore(ctx, substituteClientID)
	substituteClient, found := getClientState(substituteClientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, substituteClientID)
	}

	return clientState.CheckSubstituteAndUpdateState(ctx, cdc, clientStore, substituteClientStore, substituteClient)
}

func (lcm LightClientModule) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	clientID string,
	newClient []byte,
	newConsState []byte,
	upgradeClientProof,
	upgradeConsensusStateProof []byte,
) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Tendermint {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Tendermint, clientType)
	}

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

	clientStore := lcm.keeper.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyUpgradeAndUpdateState(ctx, cdc, clientStore, &newClientState, &newConsensusState, upgradeClientProof, upgradeConsensusStateProof)
}
