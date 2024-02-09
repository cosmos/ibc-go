package keeper

import (
	metrics "github.com/hashicorp/go-metrics"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// CreateClient generates a new client identifier and isolated prefix store for the provided client state.
// The client state is responsible for setting any client-specific data in the store via the Initialize method.
// This includes the client state, initial consensus state and any associated metadata.
func (k Keeper) CreateClient(
	ctx sdk.Context, clientType string, clientState []byte, consensusState []byte,
) (string, error) {
	if clientType == exported.Localhost {
		return "", errorsmod.Wrapf(types.ErrInvalidClientType, "cannot create client of type: %s", clientType)
	}

	params := k.GetParams(ctx)
	if !params.IsAllowedClient(clientType) {
		return "", errorsmod.Wrapf(
			types.ErrInvalidClientType,
			"client state type %s is not registered in the allowlist", clientType,
		)
	}

	clientID := k.GenerateClientIdentifier(ctx, clientType)

	lightClientModule, found := k.router.GetRoute(clientID)
	if !found {
		return "", errorsmod.Wrap(types.ErrRouteNotFound, clientID)
	}

	if err := lightClientModule.Initialize(ctx, clientID, clientState, consensusState); err != nil {
		return "", err
	}

	if status := k.GetClientStatus(ctx, clientID); status != exported.Active {
		return "", errorsmod.Wrapf(types.ErrClientNotActive, "cannot create client (%s) with status %s", clientID, status)
	}

	// 	k.Logger(ctx).Info("client created at height", "client-id", clientID, "height", clientState.GetLatestHeight().String())

	defer telemetry.IncrCounterWithLabels(
		[]string{"ibc", "client", "create"},
		1,
		[]metrics.Label{telemetry.NewLabel(types.LabelClientType, clientType)},
	)

	emitCreateClientEvent(ctx, clientID, clientType)

	return clientID, nil
}

// UpdateClient updates the consensus state and the state root from a provided header.
func (k Keeper) UpdateClient(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	if status := k.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "cannot update client (%s) with status %s", clientID, status)
	}

	clientType, _, err := types.ParseClientIdentifier(clientID)
	if err != nil {
		return errorsmod.Wrapf(types.ErrClientNotFound, "clientID (%s)", clientID)
	}

	lightClientModule, found := k.router.GetRoute(clientID)
	if !found {
		return errorsmod.Wrap(types.ErrRouteNotFound, clientID)
	}

	if err := lightClientModule.VerifyClientMessage(ctx, clientID, clientMsg); err != nil {
		return err
	}

	foundMisbehaviour := lightClientModule.CheckForMisbehaviour(ctx, clientID, clientMsg)
	if foundMisbehaviour {
		lightClientModule.UpdateStateOnMisbehaviour(ctx, clientID, clientMsg)

		k.Logger(ctx).Info("client frozen due to misbehaviour", "client-id", clientID)

		defer telemetry.IncrCounterWithLabels(
			[]string{"ibc", "client", "misbehaviour"},
			1,
			[]metrics.Label{
				telemetry.NewLabel(types.LabelClientType, clientType),
				telemetry.NewLabel(types.LabelClientID, clientID),
				telemetry.NewLabel(types.LabelMsgType, "update"),
			},
		)

		emitSubmitMisbehaviourEvent(ctx, clientID, clientType)

		return nil
	}

	consensusHeights := lightClientModule.UpdateState(ctx, clientID, clientMsg)

	k.Logger(ctx).Info("client state updated", "client-id", clientID, "heights", consensusHeights)

	defer telemetry.IncrCounterWithLabels(
		[]string{"ibc", "client", "update"},
		1,
		[]metrics.Label{
			telemetry.NewLabel(types.LabelClientType, clientType),
			telemetry.NewLabel(types.LabelClientID, clientID),
			telemetry.NewLabel(types.LabelUpdateType, "msg"),
		},
	)

	// emitting events in the keeper emits for both begin block and handler client updates
	emitUpdateClientEvent(ctx, clientID, clientType, consensusHeights, k.cdc, clientMsg)

	return nil
}

// UpgradeClient upgrades the client to a new client state if this new client was committed to
// by the old client at the specified upgrade height
func (k Keeper) UpgradeClient(
	ctx sdk.Context,
	clientID string,
	upgradedClient, upgradedConsState, upgradeClientProof, upgradeConsensusStateProof []byte,
) error {
	if status := k.GetClientStatus(ctx, clientID); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "cannot upgrade client (%s) with status %s", clientID, status)
	}

	clientType, _, err := types.ParseClientIdentifier(clientID)
	if err != nil {
		return errorsmod.Wrapf(types.ErrClientNotFound, "clientID (%s)", clientID)
	}

	lightClientModule, found := k.router.GetRoute(clientID)
	if !found {
		return errorsmod.Wrap(types.ErrRouteNotFound, clientID)
	}

	if err := lightClientModule.VerifyUpgradeAndUpdateState(ctx, clientID, upgradedClient, upgradedConsState, upgradeClientProof, upgradeConsensusStateProof); err != nil {
		return errorsmod.Wrapf(err, "cannot upgrade client with ID %s", clientID)
	}

	// k.Logger(ctx).Info("client state upgraded", "client-id", clientID, "height", upgradedClient.GetLatestHeight().String())

	defer telemetry.IncrCounterWithLabels(
		[]string{"ibc", "client", "upgrade"},
		1,
		[]metrics.Label{
			telemetry.NewLabel(types.LabelClientType, clientType),
			telemetry.NewLabel(types.LabelClientID, clientID),
		},
	)

	emitUpgradeClientEvent(ctx, clientID, clientType)

	return nil
}

// RecoverClient will retrieve the subject and substitute client.
// A callback will occur to the subject client state with the client
// prefixed store being provided for both the subject and the substitute client.
// The IBC client implementations are responsible for validating the parameters of the
// substitute (ensuring they match the subject's parameters) as well as copying
// the necessary consensus states from the substitute to the subject client
// store. The substitute must be Active and the subject must not be Active.
func (k Keeper) RecoverClient(ctx sdk.Context, subjectClientID, substituteClientID string) error {
	subjectClientState, found := k.GetClientState(ctx, subjectClientID)
	if !found {
		return errorsmod.Wrapf(types.ErrClientNotFound, "subject client with ID %s", subjectClientID)
	}

	subjectClientStore := k.ClientStore(ctx, subjectClientID)

	if status := k.GetClientStatus(ctx, subjectClientID); status == exported.Active {
		return errorsmod.Wrapf(types.ErrInvalidRecoveryClient, "cannot recover %s subject client", exported.Active)
	}

	substituteClientState, found := k.GetClientState(ctx, substituteClientID)
	if !found {
		return errorsmod.Wrapf(types.ErrClientNotFound, "substitute client with ID %s", substituteClientID)
	}

	if subjectClientState.GetLatestHeight().GTE(substituteClientState.GetLatestHeight()) {
		return errorsmod.Wrapf(types.ErrInvalidHeight, "subject client state latest height is greater or equal to substitute client state latest height (%s >= %s)", subjectClientState.GetLatestHeight(), substituteClientState.GetLatestHeight())
	}

	substituteClientStore := k.ClientStore(ctx, substituteClientID)

	if status := k.GetClientStatus(ctx, substituteClientID); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "substitute client is not %s, status is %s", exported.Active, status)
	}

	if err := subjectClientState.CheckSubstituteAndUpdateState(ctx, k.cdc, subjectClientStore, substituteClientStore, substituteClientState); err != nil {
		return errorsmod.Wrap(err, "failed to validate substitute client")
	}

	k.Logger(ctx).Info("client recovered", "client-id", subjectClientID)

	defer telemetry.IncrCounterWithLabels(
		[]string{"ibc", "client", "update"},
		1,
		[]metrics.Label{
			telemetry.NewLabel(types.LabelClientType, substituteClientState.ClientType()),
			telemetry.NewLabel(types.LabelClientID, subjectClientID),
			telemetry.NewLabel(types.LabelUpdateType, "recovery"),
		},
	)

	// emitting events in the keeper for recovering clients
	emitRecoverClientEvent(ctx, subjectClientID, substituteClientState.ClientType())

	return nil
}
