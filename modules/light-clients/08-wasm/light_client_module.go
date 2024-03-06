package wasm

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	keeper        wasmkeeper.Keeper
	storeProvider exported.ClientStoreProvider
}

// NewLightClientModule creates and returns a new 08-wasm LightClientModule.
func NewLightClientModule(keeper wasmkeeper.Keeper) LightClientModule {
	return LightClientModule{
		keeper: keeper,
	}
}

// RegisterStoreProvider is called by core IBC when a LightClientModule is added to the router.
// It allows the LightClientModule to set a ClientStoreProvider which supplies isolated prefix client stores
// to IBC light client instances.
func (lcm *LightClientModule) RegisterStoreProvider(storeProvider exported.ClientStoreProvider) {
	lcm.storeProvider = storeProvider
}

// Initialize is called upon client creation, it allows the client to perform validation on the initial consensus state and set the
// client state, consensus state and any client-specific metadata necessary for correct light client operation in the provided client store.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	var clientState types.ClientState
	if err := lcm.keeper.Codec().Unmarshal(clientStateBz, &clientState); err != nil {
		return err
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	var consensusState types.ConsensusState
	if err := lcm.keeper.Codec().Unmarshal(consensusStateBz, &consensusState); err != nil {
		return err
	}

	if err := consensusState.ValidateBasic(); err != nil {
		return err
	}

	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	return clientState.Initialize(ctx, cdc, clientStore, &consensusState)
}

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyClientMessage(ctx, lcm.keeper.Codec(), clientStore, clientMsg)
}

// CheckForMisbehaviour checks for evidence of a misbehaviour in Header or Misbehaviour type. It assumes the ClientMessage
// has already been verified.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) bool {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.CheckForMisbehaviour(ctx, cdc, clientStore, clientMsg)
}

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	clientState.UpdateStateOnMisbehaviour(ctx, cdc, clientStore, clientMsg)
}

// UpdateState updates and stores as necessary any associated information for an IBC client, such as the ClientState and corresponding ConsensusState.
// Upon successful update, a list of consensus heights is returned. It assumes the ClientMessage has already been verified.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) []exported.Height {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.UpdateState(ctx, cdc, clientStore, clientMsg)
}

// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) VerifyMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height, // TODO: change to concrete type
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path, // TODO: change to conrete type
	value []byte,
) error {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyMembership(ctx, clientStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path, value)
}

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
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

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyNonMembership(ctx, clientStore, cdc, height, delayTimePeriod, delayBlockPeriod, proof, path)
}

// Status must return the status of the client. Only Active clients are allowed to process packets.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) Status(ctx sdk.Context, clientID string) exported.Status {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		return exported.Unknown
	}

	return clientState.Status(ctx, clientStore, cdc)
}

// LatestHeight returns the latest height for the client state for the given client identifier.
// If no client is present for the provided client identifier a zero value height is returned.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) LatestHeight(ctx sdk.Context, clientID string) exported.Height {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		return clienttypes.ZeroHeight()
	}

	return clientState.LatestHeight
}

// TimestampAtHeight must return the timestamp for the consensus state associated with the provided height.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) TimestampAtHeight(ctx sdk.Context, clientID string, height exported.Height) (uint64, error) {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	cdc := lcm.keeper.Codec()

	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.GetTimestampAtHeight(ctx, clientStore, cdc, height)
}

// RecoverClient must verify that the provided substitute may be used to update the subject client.
// The light client must set the updated client and consensus states within the clientStore for the subject client.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (lcm LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	substituteClientType, _, err := clienttypes.ParseClientIdentifier(substituteClientID)
	if err != nil {
		return err
	}

	if substituteClientType != types.Wasm {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", types.Wasm, substituteClientType)
	}

	cdc := lcm.keeper.Codec()

	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	clientState, found := types.GetClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	substituteClientStore := lcm.storeProvider.ClientStore(ctx, substituteClientID)
	substituteClient, found := types.GetClientState(substituteClientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, substituteClientID)
	}

	return clientState.CheckSubstituteAndUpdateState(ctx, cdc, clientStore, substituteClientStore, substituteClient)
}

// VerifyUpgradeAndUpdateState, on a successful verification expects the contract to update
// the new client state, consensus state, and any other client metadata.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 08-wasm-{n}.
func (LightClientModule) VerifyUpgradeAndUpdateState(ctx sdk.Context, clientID string, newClient []byte, newConsState []byte, upgradeClientProof, upgradeConsensusStateProof []byte) error {
	return nil
}
