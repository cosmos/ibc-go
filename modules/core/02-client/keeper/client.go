package keeper

import (
	"encoding/hex"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// CreateClient creates a new client state and populates it with a given consensus
// state as defined in https://github.com/cosmos/ibc/tree/master/spec/core/ics-002-client-semantics#create
func (k Keeper) CreateClient(
	ctx sdk.Context, clientState exported.ClientState, consensusState exported.ConsensusState,
) (string, error) {
	params := k.GetParams(ctx)
	if !params.IsAllowedClient(clientState.ClientType()) {
		return "", sdkerrors.Wrapf(
			types.ErrInvalidClientType,
			"client state type %s is not registered in the allowlist", clientState.ClientType(),
		)
	}

	clientID := k.GenerateClientIdentifier(ctx, clientState.ClientType())

	k.SetClientState(ctx, clientID, clientState)
	k.Logger(ctx).Info("client created at height", "client-id", clientID, "height", clientState.GetLatestHeight().String())

	// verifies initial consensus state against client state and initializes client store with any client-specific metadata
	// e.g. set ProcessedTime in Tendermint clients
	if err := clientState.Initialize(ctx, k.cdc, k.ClientStore(ctx, clientID), consensusState); err != nil {
		return "", err
	}

	k.SetClientConsensusState(ctx, clientID, clientState.GetLatestHeight(), consensusState)

	k.Logger(ctx).Info("client created at height", "client-id", clientID, "height", clientState.GetLatestHeight().String())

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"ibc", "client", "create"},
			1,
			[]metrics.Label{telemetry.NewLabel(types.LabelClientType, clientState.ClientType())},
		)
	}()

	EmitCreateClientEvent(ctx, clientID, clientState)

	return clientID, nil
}

// UpdateClient updates the consensus state and the state root from a provided header.
func (k Keeper) UpdateClient(ctx sdk.Context, clientID string, clientMsg exported.ClientMessage) error {
	clientState, found := k.GetClientState(ctx, clientID)
	if !found {
		return sdkerrors.Wrapf(types.ErrClientNotFound, "cannot update client with ID %s", clientID)
	}

	clientStore := k.ClientStore(ctx, clientID)

	if status := clientState.Status(ctx, clientStore, k.cdc); status != exported.Active {
		return sdkerrors.Wrapf(types.ErrClientNotActive, "cannot update client (%s) with status %s", clientID, status)
	}

	if err := clientState.VerifyClientMessage(ctx, k.cdc, clientStore, clientMsg); err != nil {
		return err
	}

	foundMisbehaviour := clientState.CheckForMisbehaviour(ctx, k.cdc, clientStore, clientMsg)
	if foundMisbehaviour {
		clientState.UpdateStateOnMisbehaviour(ctx, k.cdc, clientStore, clientMsg)

		k.Logger(ctx).Info("client frozen due to misbehaviour", "client-id", clientID)

		defer func() {
			telemetry.IncrCounterWithLabels(
				[]string{"ibc", "client", "misbehaviour"},
				1,
				[]metrics.Label{
					telemetry.NewLabel(types.LabelClientType, clientState.ClientType()),
					telemetry.NewLabel(types.LabelClientID, clientID),
					telemetry.NewLabel(types.LabelMsgType, "update"),
				},
			)
		}()

		EmitSubmitMisbehaviourEvent(ctx, clientID, clientState)

		return nil
	}

	consensusHeights := clientState.UpdateState(ctx, k.cdc, clientStore, clientMsg)

	k.Logger(ctx).Info("client state updated", "client-id", clientID, "heights", consensusHeights)

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"ibc", "client", "update"},
			1,
			[]metrics.Label{
				telemetry.NewLabel(types.LabelClientType, clientState.ClientType()),
				telemetry.NewLabel(types.LabelClientID, clientID),
				telemetry.NewLabel(types.LabelUpdateType, "msg"),
			},
		)
	}()

	// Marshal the ClientMessage as an Any and encode the resulting bytes to hex.
	// This prevents the event value from containing invalid UTF-8 characters
	// which may cause data to be lost when JSON encoding/decoding.
	clientMsgStr := hex.EncodeToString(types.MustMarshalClientMessage(k.cdc, clientMsg))

	// emitting events in the keeper emits for both begin block and handler client updates
	EmitUpdateClientEvent(ctx, clientID, clientState.ClientType(), consensusHeights, clientMsgStr)

	return nil
}

// UpgradeClient upgrades the client to a new client state if this new client was committed to
// by the old client at the specified upgrade height
func (k Keeper) UpgradeClient(ctx sdk.Context, clientID string, upgradedClient exported.ClientState, upgradedConsState exported.ConsensusState,
	proofUpgradeClient, proofUpgradeConsState []byte) error {
	clientState, found := k.GetClientState(ctx, clientID)
	if !found {
		return sdkerrors.Wrapf(types.ErrClientNotFound, "cannot update client with ID %s", clientID)
	}

	clientStore := k.ClientStore(ctx, clientID)

	if status := clientState.Status(ctx, clientStore, k.cdc); status != exported.Active {
		return sdkerrors.Wrapf(types.ErrClientNotActive, "cannot upgrade client (%s) with status %s", clientID, status)
	}

	if err := clientState.VerifyUpgradeAndUpdateState(ctx, k.cdc, clientStore,
		upgradedClient, upgradedConsState, proofUpgradeClient, proofUpgradeConsState,
	); err != nil {
		return sdkerrors.Wrapf(err, "cannot upgrade client with ID %s", clientID)
	}

	k.Logger(ctx).Info("client state upgraded", "client-id", clientID, "height", upgradedClient.GetLatestHeight().String())

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"ibc", "client", "upgrade"},
			1,
			[]metrics.Label{
				telemetry.NewLabel(types.LabelClientType, upgradedClient.ClientType()),
				telemetry.NewLabel(types.LabelClientID, clientID),
			},
		)
	}()

	EmitUpgradeClientEvent(ctx, clientID, upgradedClient)

	return nil
}
