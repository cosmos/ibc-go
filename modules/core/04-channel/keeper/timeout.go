package keeper

import (
	"bytes"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// TimeoutPacket is called by a module which originally attempted to send a
// packet to a counterparty module, where the timeout height has passed on the
// counterparty chain without the packet being committed, to prove that the
// packet can no longer be executed and to allow the calling module to safely
// perform appropriate state transitions. Its intended usage is within the
// ante handler.
func (k *Keeper) TimeoutPacket(
	ctx sdk.Context,
	packet types.Packet,
	proof []byte,
	proofHeight exported.Height,
	nextSequenceRecv uint64,
) (string, error) {
	channel, found := k.GetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return "", errorsmod.Wrapf(
			types.ErrChannelNotFound,
			"port ID (%s) channel ID (%s)", packet.GetSourcePort(), packet.GetSourceChannel(),
		)
	}

	if packet.GetDestPort() != channel.Counterparty.PortId {
		return "", errorsmod.Wrapf(
			types.ErrInvalidPacket,
			"packet destination port doesn't match the counterparty's port (%s ≠ %s)", packet.GetDestPort(), channel.Counterparty.PortId,
		)
	}

	if packet.GetDestChannel() != channel.Counterparty.ChannelId {
		return "", errorsmod.Wrapf(
			types.ErrInvalidPacket,
			"packet destination channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetDestChannel(), channel.Counterparty.ChannelId,
		)
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return "", errorsmod.Wrap(
			connectiontypes.ErrConnectionNotFound,
			channel.ConnectionHops[0],
		)
	}

	// check that timeout height or timeout timestamp has passed on the other end
	proofTimestamp, err := k.clientKeeper.GetClientTimestampAtHeight(ctx, connectionEnd.ClientId, proofHeight)
	if err != nil {
		return "", err
	}

	timeout := types.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if !timeout.Elapsed(proofHeight.(clienttypes.Height), proofTimestamp) {
		return "", errorsmod.Wrap(timeout.ErrTimeoutNotReached(proofHeight.(clienttypes.Height), proofTimestamp), "packet timeout not reached")
	}

	commitment := k.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if len(commitment) == 0 {
		emitTimeoutPacketEvent(ctx, packet, channel)
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return "", types.ErrNoOpMsg
	}

	packetCommitment := types.CommitPacket(packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return "", errorsmod.Wrapf(types.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	switch channel.Ordering {
	case types.ORDERED:
		// check that packet has not been received
		if nextSequenceRecv > packet.GetSequence() {
			return "", errorsmod.Wrapf(
				types.ErrPacketReceived,
				"packet already received, next sequence receive > packet sequence (%d > %d)", nextSequenceRecv, packet.GetSequence(),
			)
		}

		// check that the recv sequence is as claimed
		err = k.connectionKeeper.VerifyNextSequenceRecv(
			ctx, connectionEnd, proofHeight, proof,
			packet.GetDestPort(), packet.GetDestChannel(), nextSequenceRecv,
		)
	case types.UNORDERED:
		err = k.connectionKeeper.VerifyPacketReceiptAbsence(
			ctx, connectionEnd, proofHeight, proof,
			packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		)
	default:
		panic(errorsmod.Wrap(types.ErrInvalidChannelOrdering, channel.Ordering.String()))
	}

	if err != nil {
		return "", err
	}

	if err = k.timeoutExecuted(ctx, channel, packet); err != nil {
		return "", err
	}

	return channel.Version, nil
}

// timeoutExecuted deletes the commitment send from this chain after it verifies timeout.
// If the timed-out packet came from an ORDERED channel then this channel will be closed.
//
// CONTRACT: this function must be called in the IBC handler
func (k *Keeper) timeoutExecuted(
	ctx sdk.Context,
	channel types.Channel,
	packet types.Packet,
) error {
	k.deletePacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if channel.Ordering == types.ORDERED {
		// Close the channel since the packet timed-out and the channel is ORDERED
		channel.State = types.CLOSED
		k.SetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), channel)
		emitChannelClosedEvent(ctx, packet, channel)
	}

	k.Logger(ctx).Info(
		"packet timed-out",
		"sequence", strconv.FormatUint(packet.GetSequence(), 10),
		"src_port", packet.GetSourcePort(),
		"src_channel", packet.GetSourceChannel(),
		"dst_port", packet.GetDestPort(),
		"dst_channel", packet.GetDestChannel(),
	)

	// emit an event marking that we have processed the timeout
	emitTimeoutPacketEvent(ctx, packet, channel)

	return nil
}

// TimeoutOnClose is called by a module in order to prove that the channel to
// which an unreceived packet was addressed has been closed, so the packet will
// never be received (even if the timeoutHeight has not yet been reached).
func (k *Keeper) TimeoutOnClose(
	ctx sdk.Context,
	packet types.Packet,
	proof,
	closedProof []byte,
	proofHeight exported.Height,
	nextSequenceRecv uint64,
) (string, error) {
	channel, found := k.GetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return "", errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", packet.GetSourcePort(), packet.GetSourceChannel())
	}

	if packet.GetDestPort() != channel.Counterparty.PortId {
		return "", errorsmod.Wrapf(
			types.ErrInvalidPacket,
			"packet destination port doesn't match the counterparty's port (%s ≠ %s)", packet.GetDestPort(), channel.Counterparty.PortId,
		)
	}

	if packet.GetDestChannel() != channel.Counterparty.ChannelId {
		return "", errorsmod.Wrapf(
			types.ErrInvalidPacket,
			"packet destination channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetDestChannel(), channel.Counterparty.ChannelId,
		)
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return "", errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	commitment := k.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if len(commitment) == 0 {
		emitTimeoutPacketEvent(ctx, packet, channel)
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return "", types.ErrNoOpMsg
	}

	packetCommitment := types.CommitPacket(packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return "", errorsmod.Wrapf(types.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	counterpartyHops := []string{connectionEnd.Counterparty.ConnectionId}

	counterparty := types.NewCounterparty(packet.GetSourcePort(), packet.GetSourceChannel())
	expectedChannel := types.Channel{
		State:          types.CLOSED,
		Ordering:       channel.Ordering,
		Counterparty:   counterparty,
		ConnectionHops: counterpartyHops,
		Version:        channel.Version,
	}

	// check that the opposing channel end has closed
	if err := k.connectionKeeper.VerifyChannelState(
		ctx, connectionEnd, proofHeight, closedProof,
		channel.Counterparty.PortId, channel.Counterparty.ChannelId,
		expectedChannel,
	); err != nil {
		return "", err
	}

	var err error
	switch channel.Ordering {
	case types.ORDERED:
		// check that packet has not been received
		if nextSequenceRecv > packet.GetSequence() {
			return "", errorsmod.Wrapf(types.ErrInvalidPacket, "packet already received, next sequence receive > packet sequence (%d > %d", nextSequenceRecv, packet.GetSequence())
		}

		// check that the recv sequence is as claimed
		err = k.connectionKeeper.VerifyNextSequenceRecv(
			ctx, connectionEnd, proofHeight, proof,
			packet.GetDestPort(), packet.GetDestChannel(), nextSequenceRecv,
		)
	case types.UNORDERED:
		err = k.connectionKeeper.VerifyPacketReceiptAbsence(
			ctx, connectionEnd, proofHeight, proof,
			packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		)
	default:
		panic(errorsmod.Wrap(types.ErrInvalidChannelOrdering, channel.Ordering.String()))
	}

	if err != nil {
		return "", err
	}

	if err = k.timeoutExecuted(ctx, channel, packet); err != nil {
		return "", err
	}

	return channel.Version, nil
}
