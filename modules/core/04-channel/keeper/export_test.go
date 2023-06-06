package keeper

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// ValidateUpgradeFields is a wrapper around validateUpgradeFields to allow the function to be directly called in tests.
func (k Keeper) ValidateUpgradeFields(ctx sdk.Context, proposedUpgrade types.UpgradeFields, currentChannel types.Channel) error {
	return k.validateUpgradeFields(ctx, proposedUpgrade, currentChannel)
}

// ValidateUpgradeFields is a wrapper around validateUpgradeFields to allow the function to be directly called in tests.
func (k Keeper) AbortHandshake(ctx sdk.Context, portID, channelID string, upgradeError *types.UpgradeError) error {
	return k.abortHandshake(ctx, portID, channelID, upgradeError)
}
