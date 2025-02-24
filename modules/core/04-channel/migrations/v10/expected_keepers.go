package v10

import (
	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

type ChannelKeeper interface {
	Logger(ctx sdk.Context) log.Logger
	SetChannel(ctx sdk.Context, portID, channelID string, channel types.Channel)
}
