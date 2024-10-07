package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
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
	// Lookup counterparty associated with our source channel to retrieve the destination channel
	counterparty, ok := k.GetCounterparty(ctx, sourceID)
	if !ok {
		// If the counterparty is not found, attempt to retrieve a v1 channel from the channel keeper
		// if it exists, then we will convert it to a v2 counterparty and store it in the packet server keeper
		// for future use.
		// TODO: figure out how aliasing will work when more than one packet data is sent.
		if counterparty, ok = k.AliasV1Channel(ctx, data[0].SourcePort, sourceID); ok {
			// we can key on just the source channel here since channel ids are globally unique
			k.SetCounterparty(ctx, sourceID, counterparty)
		} else {
			// if neither a counterparty nor channel is found then simply return an error
			return 0, errorsmod.Wrap(types.ErrCounterpartyNotFound, sourceID)
		}
	}

	destID := counterparty.CounterpartyChannelId
	clientID := counterparty.ClientId

	// retrieve the sequence send for this channel
	// if no packets have been sent yet, initialize the sequence to 1.
	sequence, found := k.GetNextSequenceSend(ctx, sourceID)
	if !found {
		sequence = 1
	}

	// construct packet from given fields and channel state
	packet := channeltypesv2.NewPacket(sequence, sourceID, destID, timeoutTimestamp, data...)

	if err := packet.ValidateBasic(); err != nil {
		return 0, errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "constructed packet failed basic validation: %v", err)
	}

	// check that the client of counterparty chain is still active
	if status := k.ClientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return 0, errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	// retrieve latest height and timestamp of the client of counterparty chain
	latestHeight := k.ClientKeeper.GetClientLatestHeight(ctx, clientID)
	if latestHeight.IsZero() {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "cannot send packet using client (%s) with zero height", clientID)
	}

	latestTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, clientID, latestHeight)
	if err != nil {
		return 0, err
	}
	// check if packet is timed out on the receiving chain
	timeout := channeltypes.NewTimeoutWithTimestamp(timeoutTimestamp)
	if timeout.TimestampElapsed(latestTimestamp) {
		return 0, errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}

	commitment := channeltypesv2.CommitPacket(packet)

	// bump the sequence and set the packet commitment, so it is provable by the counterparty
	k.SetNextSequenceSend(ctx, sourceID, sequence+1)
	k.SetPacketCommitment(ctx, sourceID, packet.GetSequence(), commitment)

	k.Logger(ctx).Info("packet sent", "sequence", strconv.FormatUint(packet.Sequence, 10), "dest_id", packet.DestinationId, "src_id", packet.SourceId)

	EmitSendPacketEvents(ctx, packet)

	return sequence, nil
}

// RecvPacket implements the packet receiving logic required by a packet handler.
// The packet is checked for correctness including asserting that the packet was
// sent and received on clients which are counterparties for one another.
// If the packet has already been received a no-op error is returned.
// The packet handler will verify that the packet has not timed out and that the
// counterparty stored a packet commitment. If successful, a packet receipt is stored
// to indicate to the counterparty successful delivery.
func (k Keeper) recvPacket(
	ctx context.Context,
	packet channeltypesv2.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packet.DestinationId)
	if !ok {
		// If the counterparty is not found, attempt to retrieve a v1 channel from the channel keeper
		// if it exists, then we will convert it to a v2 counterparty and store it in the packet server keeper
		// for future use.
		// TODO: figure out how aliasing will work when more than one packet data is sent.
		if counterparty, ok = k.AliasV1Channel(ctx, packet.Data[0].SourcePort, packet.SourceId); ok {
			// we can key on just the source channel here since channel ids are globally unique
			k.SetCounterparty(ctx, packet.SourceId, counterparty)
		} else {
			// if neither a counterparty nor channel is found then simply return an error
			return errorsmod.Wrap(types.ErrCounterpartyNotFound, packet.SourceId)
		}
	}
	if counterparty.ClientId != packet.SourceId {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// check if packet timed out by comparing it with the latest height of the chain
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	selfHeight, selfTimestamp := clienttypes.GetSelfHeight(ctx), uint64(sdkCtx.BlockTime().UnixNano())
	timeout := channeltypes.NewTimeoutWithTimestamp(packet.GetTimeoutTimestamp())
	if timeout.Elapsed(selfHeight, selfTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutElapsed(selfHeight, selfTimestamp), "packet timeout elapsed")
	}

	// REPLAY PROTECTION: Packet receipts will indicate that a packet has already been received
	// on unordered channels. Packet receipts must not be pruned, unless it has been marked stale
	// by the increase of the recvStartSequence.
	_, found := k.GetPacketReceipt(ctx, packet.DestinationId, packet.Sequence)
	if found {
		EmitRecvPacketEvents(ctx, packet)
		// This error indicates that the packet has already been relayed. Core IBC will
		// treat this error as a no-op in order to prevent an entire relay transaction
		// from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	path := hostv2.PacketCommitmentKey(packet.SourceId, sdk.Uint64ToBigEndian(packet.Sequence))
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	commitment := channeltypesv2.CommitPacket(packet)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		packet.DestinationId,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		commitment,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet commitment verification for client (%s)", packet.DestinationId)
	}

	// Set Packet Receipt to prevent timeout from occurring on counterparty
	k.SetPacketReceipt(ctx, packet.DestinationId, packet.Sequence)

	k.Logger(ctx).Info("packet received", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_id", packet.SourceId, "dst_id", packet.DestinationId)

	EmitRecvPacketEvents(ctx, packet)

	return nil
}
