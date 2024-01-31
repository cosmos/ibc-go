package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, portID, channelID string) (channeltypes.Channel, bool)
	SendPacket(ctx sdk.Context, portID, channelID string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error)
	WriteAcknowledgement(
		ctx sdk.Context,
		packet exported.PacketI,
		acknowledgement exported.Acknowledgement,
	) error
}
