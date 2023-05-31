package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// StartFlushUpgradeHandshake is an exported version of startFlushUpgradeHandshake to be used for unit testing
func (k Keeper) StartFlushUpgradeHandshake(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedUpgradeFields types.UpgradeFields,
	counterpartyChannel types.Channel,
	counterpartyUpgrade types.Upgrade,
	proofCounterpartyChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
) error {
	return k.startFlushUpgradeHandshake(ctx, portID, channelID, proposedUpgradeFields, counterpartyChannel, counterpartyUpgrade, proofCounterpartyChannel, proofUpgrade, proofHeight)
}
