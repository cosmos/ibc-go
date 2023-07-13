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

// ValidateSelfUpgradeFields is a wrapper around validateSelfUpgradeFields to allow the function to be directly called in tests.
func (k Keeper) ValidateSelfUpgradeFields(ctx sdk.Context, proposedUpgrade types.UpgradeFields, currentChannel types.Channel) error {
	return k.validateSelfUpgradeFields(ctx, proposedUpgrade, currentChannel)
}

// CreateChannelUpgradeInitEvent is a wrapper around createChannelUpgradeInitEvent to allow the function to be directly called in tests.
func CreateChannelUpgradeInitEvent(portID string, channelID string, currentChannel types.Channel, upgrade types.Upgrade) sdk.Events {
	return createChannelUpgradeInitEvent(portID, channelID, currentChannel, upgrade)
}

// CreateChannelUpgradeTryEvent is a wrapper around createChannelUpgradeTryEvent to allow the function to be directly called in tests.
func CreateChannelUpgradeTryEvent(portID string, channelID string, currentChannel types.Channel, upgrade types.Upgrade) sdk.Events {
	return createChannelUpgradeTryEvent(portID, channelID, currentChannel, upgrade)
}

// CreateChannelUpgradeAckEvent is a wrapper around createChannelUpgradeAckEvent to allow the function to be directly called in tests.
func CreateChannelUpgradeAckEvent(portID string, channelID string, currentChannel types.Channel, upgrade types.Upgrade) sdk.Events {
	return createChannelUpgradeAckEvent(portID, channelID, currentChannel, upgrade)
}

// CreateChannelUpgradeOpenEvent is a wrapper around createChannelUpgradeOpenEvent to allow the function to be directly called in tests.
func CreateChannelUpgradeOpenEvent(portID string, channelID string, currentChannel types.Channel) sdk.Events {
	return createChannelUpgradeOpenEvent(portID, channelID, currentChannel)
}

// CreateChannelUpgradeTimeoutEvent is a wrapper around createChannelUpgradeTimeoutEvent to allow the function to be directly called in tests.
func CreateChannelUpgradeTimeoutEvent(portID string, channelID string, currentChannel types.Channel, upgrade types.Upgrade) sdk.Events {
	return createChannelUpgradeTimeoutEvent(portID, channelID, currentChannel, upgrade)
}

// CreateErrorReceiptEvent is a wrapper around createErrorReceiptEvent to allow the function to be directly called in tests.
func CreateErrorReceiptEvent(portID string, channelID string, currentChannel types.Channel, upgrade types.Upgrade, err error) sdk.Events {
	return createErrorReceiptEvent(portID, channelID, currentChannel, upgrade, err)
}

// CreateChannelUpgradeCancelEvent is a wrapper around createChannelUpgradeCancelEvent to allow the function to be directly called in tests.
func CreateChannelUpgradeCancelEvent(portID string, channelID string, currentChannel types.Channel, upgrade types.Upgrade) sdk.Events {
	return createChannelUpgradeCancelEvent(portID, channelID, currentChannel, upgrade)
}
