package keeper

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// StartFlushUpgradeHandshake is a wrapper around startFlushUpgradeHandshake to allow the function to be directly called in tests.
func (k Keeper) StartFlushUpgradeHandshake(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedUpgradeFields types.UpgradeFields,
	counterpartyChannel types.Channel,
	counterpartyUpgrade types.Upgrade,
	proofCounterpartyChannel,
	proofCounterpartyUpgrade []byte,
	proofHeight clienttypes.Height,
) error {
	return k.startFlushUpgradeHandshake(ctx, portID, channelID, proposedUpgradeFields, counterpartyChannel, counterpartyUpgrade, proofCounterpartyChannel, proofCounterpartyUpgrade, proofHeight)
}

// ValidateUpgradeFields is a wrapper around validateUpgradeFields to allow the function to be directly called in tests.
func (k Keeper) ValidateUpgradeFields(ctx sdk.Context, proposedUpgrade types.UpgradeFields, currentChannel types.Channel) error {
	return k.validateUpgradeFields(ctx, proposedUpgrade, currentChannel)
}

<<<<<<< HEAD
// AbortUpgrade is a wrapper around abortUpgrade to allow the function to be directly called in tests.
func (k Keeper) AbortUpgrade(ctx sdk.Context, portID, channelID string, err error) error {
	return k.abortUpgrade(ctx, portID, channelID, err)
=======
// WriteUpgradeOpenChannel is a wrapper around writeUpgradeOpenChannel to allow the function to be directly called in tests.
func (k Keeper) WriteUpgradeOpenChannel(ctx sdk.Context, portID, channelID string) {
	k.writeUpgradeOpenChannel(ctx, portID, channelID)
>>>>>>> 04-channel-upgrades
}
