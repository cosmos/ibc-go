package keeper

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// StartFlushing is a wrapper around startFlushing to allow the function to be directly called in tests.
func (k Keeper) StartFlushing(ctx sdk.Context, portID, channelID string, upgrade *types.Upgrade) error {
	return k.startFlushing(ctx, portID, channelID, upgrade)
}

// ValidateSelfUpgradeFields is a wrapper around validateSelfUpgradeFields to allow the function to be directly called in tests.
func (k Keeper) ValidateSelfUpgradeFields(ctx sdk.Context, proposedUpgrade types.UpgradeFields, currentChannel types.Channel) error {
	return k.validateSelfUpgradeFields(ctx, proposedUpgrade, currentChannel)
}

// CheckForUpgradeCompatibility is a wrapper around checkForUpgradeCompatibility to allow the function to be directly called in tests.
func (k Keeper) CheckForUpgradeCompatibility(ctx sdk.Context, upgradeFields, counterpartyUpgradeFields types.UpgradeFields) error {
	return k.checkForUpgradeCompatibility(ctx, upgradeFields, counterpartyUpgradeFields)
}

// SyncUpgradeSequence is a wrapper around syncUpgradeSequence to allow the function to be directly called in tests.
func (k Keeper) SyncUpgradeSequence(ctx sdk.Context, portID, channelID string, channel types.Channel, counterpartyUpgradeSequence uint64) error {
	return k.syncUpgradeSequence(ctx, portID, channelID, channel, counterpartyUpgradeSequence)
}

// WriteErrorReceipt is a wrapper around writeErrorReceipt to allow the function to be directly called in tests.
func (k Keeper) WriteErrorReceipt(ctx sdk.Context, portID, channelID string, upgradeError *types.UpgradeError) {
	k.writeErrorReceipt(ctx, portID, channelID, upgradeError)
}
