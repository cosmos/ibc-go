package keeper

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	metrics "github.com/hashicorp/go-metrics"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// ClientUpdateProposal will retrieve the subject and substitute client.
// A callback will occur to the subject client state with the client
// prefixed store being provided for both the subject and the substitute client.
// The IBC client implementations are responsible for validating the parameters of the
// subtitute (enusring they match the subject's parameters) as well as copying
// the necessary consensus states from the subtitute to the subject client
// store. The substitute must be Active and the subject must not be Active.
//
// Deprecated: This method is deprecated in favour of RecoverClient and will be removed in a future release.
func (k Keeper) ClientUpdateProposal(ctx sdk.Context, p *types.ClientUpdateProposal) error {
	subjectClientState, found := k.GetClientState(ctx, p.SubjectClientId)
	if !found {
		return errorsmod.Wrapf(types.ErrClientNotFound, "subject client with ID %s", p.SubjectClientId)
	}

	subjectClientStore := k.ClientStore(ctx, p.SubjectClientId)

	if status := k.GetClientStatus(ctx, subjectClientState, p.SubjectClientId); status == exported.Active {
		return errorsmod.Wrap(types.ErrInvalidUpdateClientProposal, "cannot update Active subject client")
	}

	substituteClientState, found := k.GetClientState(ctx, p.SubstituteClientId)
	if !found {
		return errorsmod.Wrapf(types.ErrClientNotFound, "substitute client with ID %s", p.SubstituteClientId)
	}

	if subjectClientState.GetLatestHeight().GTE(substituteClientState.GetLatestHeight()) {
		return errorsmod.Wrapf(types.ErrInvalidHeight, "subject client state latest height is greater or equal to substitute client state latest height (%s >= %s)", subjectClientState.GetLatestHeight(), substituteClientState.GetLatestHeight())
	}

	substituteClientStore := k.ClientStore(ctx, p.SubstituteClientId)

	if status := k.GetClientStatus(ctx, substituteClientState, p.SubstituteClientId); status != exported.Active {
		return errorsmod.Wrapf(types.ErrClientNotActive, "substitute client is not Active, status is %s", status)
	}

	if err := subjectClientState.CheckSubstituteAndUpdateState(ctx, k.cdc, subjectClientStore, substituteClientStore, substituteClientState); err != nil {
		return err
	}

	k.Logger(ctx).Info("client updated after governance proposal passed", "client-id", p.SubjectClientId)

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"ibc", "client", "update"},
			1,
			[]metrics.Label{
				telemetry.NewLabel(types.LabelClientType, substituteClientState.ClientType()),
				telemetry.NewLabel(types.LabelClientID, p.SubjectClientId),
				telemetry.NewLabel(types.LabelUpdateType, "proposal"),
			},
		)
	}()

	// emitting events in the keeper for proposal updates to clients
	EmitUpdateClientProposalEvent(ctx, p.SubjectClientId, substituteClientState.ClientType())

	return nil
}
