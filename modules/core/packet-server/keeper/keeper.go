package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	channelKeeper types.ChannelKeeper
	clientKeeper  types.ClientKeeper
}

func NewKeeper(cdc codec.BinaryCodec, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper) *Keeper {
	return &Keeper{
		cdc:           cdc,
		channelKeeper: channelKeeper,
		clientKeeper:  clientKeeper,
	}
}
