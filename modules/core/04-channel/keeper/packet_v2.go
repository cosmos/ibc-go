package keeper

import (
	"crypto/sha256"
	"github.com/cosmos/cosmos-sdk/codec"
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

	commitment := CommitPacketV2(k.cdc, packet)

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

func CommitPacketV2(cdc codec.BinaryCodec, packet types.PacketV2) []byte {
	timeoutHeight := packet.GetTimeoutHeight()

	buf := sdk.Uint64ToBigEndian(packet.GetTimeoutTimestamp())

	revisionNumber := sdk.Uint64ToBigEndian(timeoutHeight.GetRevisionNumber())
	buf = append(buf, revisionNumber...)

	revisionHeight := sdk.Uint64ToBigEndian(timeoutHeight.GetRevisionHeight())
	buf = append(buf, revisionHeight...)

	// TODO
	//dataHash := sha256.Sum256(packet.GetData())
	//buf = append(buf, dataHash[:]...)

	hash := sha256.Sum256(buf)
	return hash[:]
}
