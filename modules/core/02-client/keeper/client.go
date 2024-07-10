package keeper

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/internal/telemetry"
)

// CreateClient generates a new client identifier and invokes the associated light client module in order to
// initialize a new client. An isolated prefixed store will be reserved for this client using the generated
// client identifier. The light client module is responsible for setting any client-specific data in the store
// via the Initialize method. This includes the client state, initial consensus state and any associated
// metadata. The generated client identifier will be returned if a client was successfully initialized.
func (k *Keeper) CreateClient(ctx sdk.Context, clientType string, clientState, consensusState []byte) (string, error) {
	if clientType == exported.Localhost {
		return "", errorsmod.Wrapf(types.ErrInvalidClientType, "cannot create client of type: %s", clientType)
	}

	clientID := k.GenerateClientIdentifier(ctx, clientType)

	clientModule, err := k.getLightClientModule(ctx, clientID)
	if err != nil {
		return "", err
	}

	if err := clientModule.Initialize(ctx, clientID, clientState, consensusState); err != nil {
		return "", err
	}

	if status := k.GetClientStatus(ctx, clientID); status != exported.Active {
		return "", errorsmod.Wrapf(types.ErrClientNotActive, "cannot create client (%s) with status %s", clientID, status)
	}

	initialHeight := clientModule.LatestHeight(ctx, clientID)
	k.Logger(ctx).Info("client created at height", "client-id", clientID, "height", initialHeight.String())

	defer telemetry.ReportCreateClient(clientType)

	emitCreateClientEvent(ctx, clientID, clientType, initialHeight)

	return clientID, nil
}

// UpdateClient updates the consensus state and the state root from a provided header.
func (k *Keeper) UpdateClient(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	if status := k.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "cannot update client (%s) with status %s", clientID, status)
	}

	clientType, _, err := types.ParseClientIdentifier(clientID)
	if err != nil {
		return errorsmod.Wrapf(err, "unable to parse client identifier %s", clientID)
	}

	clientModule, err := k.getLightClientModule(ctx, clientID)
	if err != nil {
		return err
	}

	if err := clientModule.VerifyClientMessage(ctx, clientID, clientMsg); err != nil {
		return err
	}

	foundMisbehaviour := clientModule.CheckForMisbehaviour(ctx, clientID, clientMsg)
	if foundMisbehaviour {
		clientModule.UpdateStateOnMisbehaviour(ctx, clientID, clientMsg)

		k.Logger(ctx).Info("client frozen due to misbehaviour", "client-id", clientID)

		defer telemetry.ReportUpdateClient(foundMisbehaviour, clientType, clientID)

		emitSubmitMisbehaviourEvent(ctx, clientID, clientType)

		return nil
	}

	consensusHeights := clientModule.UpdateState(ctx, clientID, clientMsg)

	k.Logger(ctx).Info("client state updated", "client-id", clientID, "heights", consensusHeights)

	defer telemetry.ReportUpdateClient(foundMisbehaviour, clientType, clientID)

	// emitting events in the keeper emits for both begin block and handler client updates
	emitUpdateClientEvent(ctx, clientID, clientType, consensusHeights, k.cdc, clientMsg)

	return nil
}

// UpgradeClient upgrades the client to a new client state if this new client was committed to
// by the old client at the specified upgrade height
func (k *Keeper) UpgradeClient(
	ctx sdk.Context,
	clientID string,
	upgradedClient, upgradedConsState, upgradeClientProof, upgradeConsensusStateProof []byte,
) error {
	if status := k.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "cannot upgrade client (%s) with status %s", clientID, status)
	}

	clientType, _, err := types.ParseClientIdentifier(clientID)
	if err != nil {
		return errorsmod.Wrapf(err, "unable to parse client identifier %s", clientID)
	}

	clientModule, err := k.getLightClientModule(ctx, clientID)
	if err != nil {
		return err
	}

	if err := clientModule.VerifyUpgradeAndUpdateState(ctx, clientID, upgradedClient, upgradedConsState, upgradeClientProof, upgradeConsensusStateProof); err != nil {
		return errorsmod.Wrapf(err, "cannot upgrade client with ID %s", clientID)
	}

	latestHeight := clientModule.LatestHeight(ctx, clientID)
	k.Logger(ctx).Info("client state upgraded", "client-id", clientID, "height", latestHeight.String())

	defer telemetry.ReportUpgradeClient(clientType, clientID)

	emitUpgradeClientEvent(ctx, clientID, clientType, latestHeight)

	return nil
}

// RecoverClient will invoke the light client module associated with the subject clientID requesting it to
// recover the subject client given a substitute client identifier. The light client implementation
// is responsible for validating the parameters of the substitute (ensuring they match the subject's parameters)
// as well as copying the necessary consensus states from the substitute to the subject client store.
// The substitute must be Active and the subject must not be Active.
func (k *Keeper) RecoverClient(ctx sdk.Context, subjectClientID, substituteClientID string) error {
	if status := k.GetClientStatus(ctx, subjectClientID); status == exported.Active {
		return errorsmod.Wrapf(types.ErrInvalidRecoveryClient, "cannot recover %s subject client", exported.Active)
	}

	if status := k.GetClientStatus(ctx, substituteClientID); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "substitute client is not %s, status is %s", exported.Active, status)
	}

	clientType, _, err := types.ParseClientIdentifier(subjectClientID)
	if err != nil {
		return errorsmod.Wrapf(types.ErrClientNotFound, "clientID (%s)", subjectClientID)
	}

	clientModule, found := k.router.GetRoute(subjectClientID)
	if !found {
		return errorsmod.Wrap(types.ErrRouteNotFound, subjectClientID)
	}

	subjectLatestHeight := clientModule.LatestHeight(ctx, subjectClientID)
	substituteLatestHeight := clientModule.LatestHeight(ctx, substituteClientID)
	if subjectLatestHeight.GTE(substituteLatestHeight) {
		return errorsmod.Wrapf(types.ErrInvalidHeight, "subject client state latest height is greater or equal to substitute client state latest height (%s >= %s)", subjectLatestHeight, substituteLatestHeight)
	}

	if err := clientModule.RecoverClient(ctx, subjectClientID, substituteClientID); err != nil {
		return err
	}

	k.Logger(ctx).Info("client recovered", "client-id", subjectClientID)

	defer telemetry.ReportRecoverClient(clientType, subjectClientID)

	// emitting events in the keeper for recovering clients
	emitRecoverClientEvent(ctx, subjectClientID, clientType)

	return nil
}
