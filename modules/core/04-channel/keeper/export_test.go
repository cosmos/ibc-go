package keeper

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// StartFlushing is a wrapper around startFlushing to allow the function to be directly called in tests.
func (k *Keeper) StartFlushing(ctx sdk.Context, portID, channelID string, upgrade *types.Upgrade) error {
	return k.startFlushing(ctx, portID, channelID, upgrade)
}

// ValidateSelfUpgradeFields is a wrapper around validateSelfUpgradeFields to allow the function to be directly called in tests.
func (k *Keeper) ValidateSelfUpgradeFields(ctx sdk.Context, proposedUpgrade types.UpgradeFields, channel types.Channel) error {
	return k.validateSelfUpgradeFields(ctx, proposedUpgrade, channel)
}

// CheckForUpgradeCompatibility is a wrapper around checkForUpgradeCompatibility to allow the function to be directly called in tests.
func (k *Keeper) CheckForUpgradeCompatibility(ctx sdk.Context, upgradeFields, counterpartyUpgradeFields types.UpgradeFields) error {
	return k.checkForUpgradeCompatibility(ctx, upgradeFields, counterpartyUpgradeFields)
}

// SetUpgradeErrorReceipt is a wrapper around setUpgradeErrorReceipt to allow the function to be directly called in tests.
func (k *Keeper) SetUpgradeErrorReceipt(ctx sdk.Context, portID, channelID string, errorReceipt types.ErrorReceipt) {
	k.setUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
}

// SetRecvStartSequence is a wrapper around setRecvStartSequence to allow the function to be directly called in tests.
func (k *Keeper) SetRecvStartSequence(ctx sdk.Context, portID, channelID string, sequence uint64) {
	k.setRecvStartSequence(ctx, portID, channelID, sequence)
}
