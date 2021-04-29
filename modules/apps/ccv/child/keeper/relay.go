package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, data ccv.ValidatorSetChangePacketData) error {
	// TODO
	return nil
}
