package keeper

import (
	"golang.org/x/exp/slices"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// SendPacketV2 is called by a module in order to send an IBC packet on a channel.
// The packet sequence generated for the packet to be sent is returned. An error
// is returned if one occurs.
func (k *Keeper) SendPacketV2(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []types.PacketData,
) (uint64, error) {
	channel, found := k.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errorsmod.Wrap(types.ErrChannelNotFound, sourceChannel)
	}

	if channel.State != types.OPEN {
		return 0, errorsmod.Wrapf(types.ErrInvalidChannelState, "channel is not OPEN (got %s)", channel.State)
	}

	sequence, found := k.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errorsmod.Wrapf(
			types.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", sourcePort, sourceChannel,
		)
	}

	// construct packet from given fields and channel state
	packet := NewPacketV2(data, sequence, sourcePort, sourceChannel,
		channel.Counterparty.PortId, channel.Counterparty.ChannelId, timeoutHeight, timeoutTimestamp)

	//if err := packet.ValidateBasic(); err != nil {
	//	return 0, errorsmod.Wrap(err, "constructed packet failed basic validation")
	//}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return 0, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	// prevent accidental sends with clients that cannot be updated
	if status := k.clientKeeper.GetClientStatus(ctx, connectionEnd.ClientId); status != exported.Active {
		return 0, errorsmod.Wrapf(clienttypes.ErrClientNotActive, "cannot send packet using client (%s) with status %s", connectionEnd.ClientId, status)
	}

	latestHeight := k.clientKeeper.GetClientLatestHeight(ctx, connectionEnd.ClientId)
	if latestHeight.IsZero() {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "cannot send packet using client (%s) with zero height", connectionEnd.ClientId)
	}

	latestTimestamp, err := k.clientKeeper.GetClientTimestampAtHeight(ctx, connectionEnd.ClientId, latestHeight)
	if err != nil {
		return 0, err
	}

	// check if packet is timed out on the receiving chain
	timeout := types.NewTimeout(packet.TimeoutHeight, packet.TimeoutTimestamp)
	if timeout.Elapsed(latestHeight, latestTimestamp) {
		return 0, errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}

	commitment := types.CommitPacketV2(k.cdc, packet)

	k.SetNextSequenceSend(ctx, sourcePort, sourceChannel, sequence+1)
	k.SetPacketCommitment(ctx, sourcePort, sourceChannel, packet.GetSequence(), commitment)

	emitSendPacketEventV2(ctx, packet, channel, timeoutHeight)

	k.Logger(ctx).Info(
		"packet sent",
		"sequence", strconv.FormatUint(packet.GetSequence(), 10),
		"src_port", sourcePort,
		"src_channel", sourceChannel,
		"dst_port", packet.GetDestinationPort(),
		"dst_channel", packet.GetDestinationChannel(),
	)

	return packet.GetSequence(), nil
}

// RecvPacketV2 is called by a module in order to receive & process an IBC packet
// sent on the corresponding channel end on the counterparty chain.
func (k *Keeper) RecvPacketV2(
	ctx sdk.Context,
	packet types.PacketV2,
	proof []byte,
	proofHeight exported.Height,
) error {
	channel, found := k.GetChannel(ctx, packet.GetDestinationPort(), packet.GetDestinationChannel())
	if !found {
		return errorsmod.Wrap(types.ErrChannelNotFound, packet.GetDestinationChannel())
	}

	if !slices.Contains([]types.State{types.OPEN, types.FLUSHING, types.FLUSHCOMPLETE}, channel.State) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected channel state to be one of [%s, %s, %s], but got %s", types.OPEN, types.FLUSHING, types.FLUSHCOMPLETE, channel.State)
	}

	// If counterpartyUpgrade is stored we need to ensure that the
	// packet sequence is < counterparty next sequence send. If the
	// counterparty is implemented correctly, this may only occur
	// when we are in FLUSHCOMPLETE and the counterparty has already
	// completed the channel upgrade.
	counterpartyUpgrade, found := k.GetCounterpartyUpgrade(ctx, packet.GetDestinationPort(), packet.GetDestinationChannel())
	if found {
		counterpartyNextSequenceSend := counterpartyUpgrade.NextSequenceSend
		if packet.GetSequence() >= counterpartyNextSequenceSend {
			return errorsmod.Wrapf(types.ErrInvalidPacket, "cannot flush packet at sequence greater than or equal to counterparty next sequence send (%d) ≥ (%d).", packet.GetSequence(), counterpartyNextSequenceSend)
		}
	}

	// packet must come from the channel's counterparty
	if packet.GetSourcePort() != channel.Counterparty.PortId {
		return errorsmod.Wrapf(
			types.ErrInvalidPacket,
			"packet source port doesn't match the counterparty's port (%s ≠ %s)", packet.GetSourcePort(), channel.Counterparty.PortId,
		)
	}

	if packet.GetSourceChannel() != channel.Counterparty.ChannelId {
		return errorsmod.Wrapf(
			types.ErrInvalidPacket,
			"packet source channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetSourceChannel(), channel.Counterparty.ChannelId,
		)
	}

	// Connection must be OPEN to receive a packet. It is possible for connection to not yet be open if packet was
	// sent optimistically before connection and channel handshake completed. However, to receive a packet,
	// connection and channel must both be open
	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connectionEnd.State != connectiontypes.OPEN {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectionEnd.State)
	}

	// check if packet timed out by comparing it with the latest height of the chain
	selfHeight, selfTimestamp := clienttypes.GetSelfHeight(ctx), uint64(ctx.BlockTime().UnixNano())
	timeout := types.NewTimeout(packet.GetTimeoutHeight(), packet.GetTimeoutTimestamp())
	if timeout.Elapsed(selfHeight, selfTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutElapsed(selfHeight, selfTimestamp), "packet timeout elapsed")
	}

	commitment := types.CommitPacketV2(k.cdc, packet)

	// verify that the counterparty did commit to sending this packet
	if err := k.connectionKeeper.VerifyPacketCommitment(
		ctx, connectionEnd, proofHeight, proof,
		packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(),
		commitment,
	); err != nil {
		return errorsmod.Wrap(err, "couldn't verify counterparty packet commitment")
	}

	//if err := k.applyReplayProtection(ctx, packet, channel); err != nil {
	//	return err
	//}

	// log that a packet has been received & executed
	k.Logger(ctx).Info(
		"packet received",
		"sequence", strconv.FormatUint(packet.GetSequence(), 10),
		"src_port", packet.GetSourcePort(),
		"src_channel", packet.GetSourceChannel(),
		"dst_port", packet.GetDestinationPort(),
		"dst_channel", packet.GetDestinationChannel(),
	)

	// emit an event that the relayer can query for
	//emitRecvPacketEvent(ctx, packet, channel)

	return nil
}

// NewPacketV2 creates a new Packet instance. It panics if the provided
// packet data interface is not registered.
func NewPacketV2(
	data []types.PacketData,
	sequence uint64, sourcePort, sourceChannel,
	destinationPort, destinationChannel string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
) types.PacketV2 {
	return types.PacketV2{
		Data:               data,
		Sequence:           sequence,
		SourcePort:         sourcePort,
		SourceChannel:      sourceChannel,
		DestinationPort:    destinationPort,
		DestinationChannel: destinationChannel,
		TimeoutHeight:      timeoutHeight,
		TimeoutTimestamp:   timeoutTimestamp,
	}
}
