package keeper

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

// RecvPacketReCheckTx applies replay protection ensuring that when relay messages are
// re-executed in ReCheckTx, we can appropriately filter out redundant relay transactions.
func (k *Keeper) RecvPacketReCheckTx(ctx sdk.Context, packet types.Packet) error {
	channel, found := k.GetChannel(ctx, packet.GetDestPort(), packet.GetDestChannel())
	if !found {
		return errorsmod.Wrap(types.ErrChannelNotFound, packet.GetDestChannel())
	}

	if err := k.applyReplayProtection(ctx, packet, channel); err != nil {
		return err
	}

	return nil
}
