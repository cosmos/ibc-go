package keeper

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
)

// EmitUpgradeClientProposalEvent emits an upgrade client proposal event
func EmitUpgradeClientProposalEvent(ctx sdk.Context, title string, height int64) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpgradeClientProposal,
			sdk.NewAttribute(types.AttributeKeyUpgradePlanTitle, title),
			sdk.NewAttribute(types.AttributeKeyUpgradePlanHeight, fmt.Sprintf("%d", height)),
		),
	)
}

// EmitUpgradeChainEvent emits an upgrade chain event.
func EmitUpgradeChainEvent(ctx sdk.Context, height int64) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpgradeChain,
			sdk.NewAttribute(types.AttributeKeyUpgradePlanHeight, strconv.FormatInt(height, 10)),
			sdk.NewAttribute(types.AttributeKeyUpgradeStore, upgradetypes.StoreKey), // which store to query proof of consensus state from
		),
	})
}
