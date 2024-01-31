package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, portID, channelID string) (channeltypes.Channel, bool)
}
