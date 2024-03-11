package tendermint

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint/internal/keeper"
)

var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	keeper        keeper.Keeper
	storeProvider exported.ClientStoreProvider
}

// NewLightClientModule creates and returns a new 08-wasm LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec, authority string) LightClientModule {
	return LightClientModule{
		keeper: keeper.NewKeeper(cdc, authority),
	}
}

// RegisterStoreProvider is called by core IBC when a LightClientModule is added to the router.
// It allows the LightClientModule to set a ClientStoreProvider which supplies isolated prefix client stores
// to IBC light client instances.
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

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
//
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

// CheckForMisbehaviour checks for evidence of a misbehaviour in Header or Misbehaviour type. It assumes the ClientMessage
// has already been verified.
//
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

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified.
//
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

// UpdateState updates and stores as necessary any associated information for an IBC client, such as the ClientState and corresponding ConsensusState.
// Upon successful update, a list of consensus heights is returned. It assumes the ClientMessage has already been verified.
//
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

// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
//
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

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
//
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

// Status must return the status of the client. Only Active clients are allowed to process packets.
//
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

// LatestHeight returns the latest height for the client state for the given client identifier.
// If no client is present for the provided client identifier a zero value height is returned.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) LatestHeight(ctx sdk.Context, clientID string) exported.Height {
	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)

	clientState, found := getClientState(clientStore, lcm.keeper.Codec())
	if !found {
		return clienttypes.ZeroHeight()
	}

	return clientState.LatestHeight
}

// TimestampAtHeight must return the timestamp for the consensus state associated with the provided height.
//
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

// RecoverClient must verify that the provided substitute may be used to update the subject client.
// The light client must set the updated client and consensus states within the clientStore for the subject client.
//
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

	return clientState.CheckSubstituteAndUpdateState(ctx, cdc, clientStore, substituteClientStore, substituteClient)
}

// VerifyUpgradeAndUpdateState checks if the upgraded client has been committed by the current client
// It will zero out all client-specific fields (e.g. TrustingPeriod) and verify all data
// in client state that must be the same across all valid Tendermint clients for the new chain.
// VerifyUpgrade will return an error if:
// - the upgradedClient is not a Tendermint ClientState
// - the latest height of the client state does not have the same revision number or has a greater
// height than the committed client.
//   - the height of upgraded client is not greater than that of current client
//   - the latest height of the new client does not match or is greater than the height in committed client
//   - any Tendermint chain specified parameter in upgraded client such as ChainID, UnbondingPeriod,
//     and ProofSpecs do not match parameters set by committed client
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
func (lcm LightClientModule) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	clientID string,
	newClient []byte,
	newConsState []byte,
	upgradeClientProof,
	upgradeConsensusStateProof []byte,
) error {
	var (
		cdc               = lcm.keeper.Codec()
		newClientState    ClientState
		newConsensusState ConsensusState
	)

	if err := cdc.Unmarshal(newClient, &newClientState); err != nil {
		return err
	}

	// newTmClientState, ok := newClientState.(*ClientState)
	// if !ok {
	// 	return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "expected client state type %T, got %T", (*ClientState)(nil), newClientState)
	// }

	if err := cdc.Unmarshal(newConsState, &newConsensusState); err != nil {
		return err
	}

	// newTmConsensusState, ok := newConsensusState.(*ConsensusState)
	// if !ok {
	// 	return errorsmod.Wrapf(clienttypes.ErrInvalidConsensus, "expected consensus state type %T, got %T", (*ConsensusState)(nil), newConsensusState)
	// }

	clientStore := lcm.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	// last height of current counterparty chain must be client's latest height
	lastHeight := clientState.LatestHeight
	if !newClientState.LatestHeight.GT(lastHeight) {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidHeight, "upgraded client height %s must be at greater than current client height %s", newClientState.LatestHeight, lastHeight)
	}

	return clientState.VerifyUpgradeAndUpdateState(ctx, cdc, clientStore, &newClientState, &newConsensusState, upgradeClientProof, upgradeConsensusStateProof)
}
