package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/modules/apps/ccv/child/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// OnRecvPacket sets the pending validator set changes that will be flushed to ABCI on Endblock
// and set the unbonding time for the packet so that we can WriteAcknowledgement after unbonding time is over.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, data ccv.ValidatorSetChangePacketData) error {
	// Set PendingChanges to be flushed and the unbonding time and unbonding packet.
	// TODO: Get PendingChanges and update the pending changes if they already exist
	k.SetPendingChanges(ctx, data)
	k.SetUnbondingTime(ctx, packet.Sequence)
	k.SetUnbondingPacket(ctx, packet.Sequence, packet)
	return nil
}

// UnbondMaturePackets will iterate over the unbonding packets in order and write acknowledgements for all
// packets that have finished unbonding.
func (k Keeper) UnbondMaturePackets(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)
	unbondingIterator := sdk.KVStorePrefixIterator(store, []byte(types.UnbondingTimePrefix))
	defer unbondingIterator.Close()

	currentTime := uint64(ctx.BlockTime().UnixNano())

	channelID, ok := k.GetParentChannel(ctx)
	if !ok {
		return sdkerrors.Wrap(channeltypes.ErrChannelNotFound, "parent channel not set")
	}
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(types.PortID, channelID))
	if !ok {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	for unbondingIterator.Valid() {
		sequence := binary.BigEndian.Uint64(unbondingIterator.Key())
		if currentTime > binary.BigEndian.Uint64(unbondingIterator.Value()) {
			// write successful ack and delete unbonding information
			packet, err := k.GetUnbondingPacket(ctx, sequence)
			if err != nil {
				return err
			}
			ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
			k.channelKeeper.WriteAcknowledgement(ctx, channelCap, packet, ack.Acknowledgement())
			k.DeleteUnbondingTime(ctx, sequence)
			k.DeleteUnbondingPacket(ctx, sequence)
		} else {
			break
		}
		unbondingIterator.Next()
	}
	return nil
}
