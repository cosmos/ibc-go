package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

// sendPacket constructs a packet from the input arguments, writes a packet commitment to state
// in order for the packet to be sent to the counterparty.
func (k *Keeper) sendPacket(
	ctx context.Context,
	sourceID string,
	timeoutTimestamp uint64,
	data []channeltypesv2.PacketData,
) (uint64, error) {
	// TODO: add aliasing logic https://github.com/cosmos/ibc-go/issues/7387

	// Lookup counterparty associated with our source channel to retrieve the destination channel
	counterparty, ok := k.GetCounterparty(ctx, sourceID)
	if !ok {
		return 0, errorsmod.Wrap(types.ErrCounterpartyNotFound, sourceID)
	}

	// retrieve the sequence send for this channel
	// if no packets have been sent yet, initialize the sequence to 1.
	sequence, found := k.GetNextSequenceSend(ctx, counterparty.ClientId)
	if !found {
		sequence = 1
	}

	// construct packet from given fields and channel state
	packet := channeltypesv2.NewPacket(sequence, sourceID, counterparty.ClientId, timeoutTimestamp, data...)

	if err := packet.ValidateBasic(); err != nil {
		return 0, errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "constructed packet failed basic validation: %v", err)
	}

	// check that the client of counterparty chain is still active
	if status := k.ClientKeeper.GetClientStatus(ctx, counterparty.ClientId); status != exported.Active {
		return 0, errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", counterparty.ClientId, status)
	}

	// retrieve latest height and timestamp of the client of counterparty chain
	latestHeight := k.ClientKeeper.GetClientLatestHeight(ctx, counterparty.ClientId)
	if latestHeight.IsZero() {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "cannot send packet using client (%s) with zero height", counterparty.ClientId)
	}

	latestTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, counterparty.ClientId, latestHeight)
	if err != nil {
		return 0, err
	}
	// check if packet is timed out on the receiving chain
	timeout := channeltypes.NewTimeoutWithTimestamp(timeoutTimestamp)
	if timeout.TimestampElapsed(latestTimestamp) {
		return 0, errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}

	commitment := channeltypesv2.CommitPacket(packet)

	// bump the sequence and set the packet commitment so it is provable by the counterparty
	k.SetNextSequenceSend(ctx, counterparty.ClientId, sequence+1)
	k.SetPacketCommitment(ctx, counterparty.ClientId, packet.GetSequence(), commitment)

	k.Logger(ctx).Info("packet sent", "sequence", strconv.FormatUint(packet.Sequence, 10), "dest_id", packet.DestinationId, "src_id", packet.SourceId)

	EmitSendPacketEvents(ctx, packet)

	return sequence, nil
}
