package keeper

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// HandleUpgradeProposal sets the upgraded client state in the upgrade store. It clears
// an IBC client state and consensus state if a previous plan was set. Then  it
// will schedule an upgrade and finally set the upgraded client state in upgrade
// store.
func (k Keeper) HandleUpgradeProposal(ctx sdk.Context, p *types.UpgradeProposal) error {
	clientState, err := types.UnpackClientState(p.UpgradedClientState)
	if err != nil {
		return errorsmod.Wrap(err, "could not unpack UpgradedClientState")
	}

	// zero out any custom fields before setting
	cs := clientState.ZeroCustomFields()
	bz, err := types.MarshalClientState(k.cdc, cs)
	if err != nil {
		return errorsmod.Wrap(err, "could not marshal UpgradedClientState")
	}

	if err := k.upgradeKeeper.ScheduleUpgrade(ctx, p.Plan); err != nil {
		return err
	}

	// sets the new upgraded client in last height committed on this chain is at plan.Height,
	// since the chain will panic at plan.Height and new chain will resume at plan.Height
	if err = k.upgradeKeeper.SetUpgradedClient(ctx, p.Plan.Height, bz); err != nil {
		return err
	}

	// emitting an event for handling client upgrade proposal
	emitUpgradeClientProposalEvent(ctx, p.Title, p.Plan.Height)

	return nil
}
