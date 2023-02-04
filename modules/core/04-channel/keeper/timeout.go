package keeper

import (
	"bytes"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	mh "github.com/cosmos/ibc-go/v7/modules/core/multihop"
)

// TimeoutPacket is called by a module which originally attempted to send a
// packet to a counterparty module, where the timeout height has passed on the
// counterparty chain without the packet being committed, to prove that the
// packet can no longer be executed and to allow the calling module to safely
// perform appropriate state transitions. Its intended usage is within the
// ante handler.
func (k Keeper) TimeoutPacket(
	ctx sdk.Context,
	packet exported.PacketI,
	proof []byte,
	proofHeight exported.Height,
	nextSequenceRecv uint64,
) error {
	channel, found := k.GetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return sdkerrors.Wrapf(
			types.ErrChannelNotFound,
			"port ID (%s) channel ID (%s)", packet.GetSourcePort(), packet.GetSourceChannel(),
		)
	}

	// NOTE: TimeoutPacket is called by the AnteHandler which acts upon the packet.Route(),
	// so the capability authentication can be omitted here

	if packet.GetDestPort() != channel.Counterparty.PortId {
		return sdkerrors.Wrapf(
			types.ErrInvalidPacket,
			"packet destination port doesn't match the counterparty's port (%s ≠ %s)", packet.GetDestPort(), channel.Counterparty.PortId,
		)
	}

	if packet.GetDestChannel() != channel.Counterparty.ChannelId {
		return sdkerrors.Wrapf(
			types.ErrInvalidPacket,
			"packet destination channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetDestChannel(), channel.Counterparty.ChannelId,
		)
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return sdkerrors.Wrap(
			connectiontypes.ErrConnectionNotFound,
			channel.ConnectionHops[0],
		)
	}

	var mProof types.MsgMultihopProofs
	var proofTimestamp uint64
	var err error
	if len(channel.ConnectionHops) > 1 {
		err := k.cdc.Unmarshal(proof, &mProof)
		if err != nil {
			return err
		}

		consensusState, err := mProof.GetMultihopCounterpartyConsensus(k.cdc)
		if err != nil {
			return err
		}
		proofTimestamp = consensusState.GetTimestamp()
	} else {
		// check that timeout height or timeout timestamp has passed on the other end
		var err error
		proofTimestamp, err = k.connectionKeeper.GetTimestampAtHeight(ctx, connectionEnd, proofHeight)
		if err != nil {
			return err
		}
	}

	timeoutHeight := packet.GetTimeoutHeight()
	if (timeoutHeight.IsZero() || proofHeight.LT(timeoutHeight)) &&
		(packet.GetTimeoutTimestamp() == 0 || proofTimestamp < packet.GetTimeoutTimestamp()) {
		return sdkerrors.Wrap(types.ErrPacketTimeout, "packet timeout has not been reached for height or timestamp")
	}

	commitment := k.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if len(commitment) == 0 {
		EmitTimeoutPacketEvent(ctx, packet, channel)
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return types.ErrNoOpMsg
	}

	if channel.State != types.OPEN {
		return sdkerrors.Wrapf(
			types.ErrInvalidChannelState,
			"channel state is not OPEN (got %s)", channel.State.String(),
		)
	}

	packetCommitment := types.CommitPacket(k.cdc, packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return sdkerrors.Wrapf(types.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	switch channel.Ordering {
	case types.ORDERED:
		// check that packet has not been received
		if nextSequenceRecv > packet.GetSequence() {
			return sdkerrors.Wrapf(
				types.ErrPacketReceived,
				"packet already received, next sequence receive > packet sequence (%d > %d)", nextSequenceRecv, packet.GetSequence(),
			)
		}

		// check that the recv sequence is as claimed
		if len(channel.ConnectionHops) > 1 {
			// verify multihop proof
			consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, connectionEnd.ClientId, proofHeight)
			if !found {
				err = sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound,
					"consensus state not found for client id: %s", connectionEnd.ClientId)
			}
			key := host.NextSequenceRecvPath(packet.GetSourcePort(), packet.GetSourceChannel())
			prefix := connectionEnd.GetCounterparty().GetPrefix()
			val := sdk.Uint64ToBigEndian(nextSequenceRecv)
			err = mh.VerifyMultihopProof(k.cdc, consensusState, channel.ConnectionHops, proof, prefix, key, val)
		} else {
			err = k.connectionKeeper.VerifyNextSequenceRecv(
				ctx, connectionEnd, proofHeight, proof,
				packet.GetDestPort(), packet.GetDestChannel(), nextSequenceRecv,
			)
		}
	case types.UNORDERED:
		if len(channel.ConnectionHops) > 1 {
			// verify multihop proof
			consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, connectionEnd.ClientId, proofHeight)
			if !found {
				err = sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound,
					"consensus state not found for client id: %s", connectionEnd.ClientId)
			}
			key := host.PacketReceiptPath(
				packet.GetSourcePort(),
				packet.GetSourceChannel(),
				packet.GetSequence(),
			)
			prefix := connectionEnd.GetCounterparty().GetPrefix()
			var value []byte = nil
			clientState, found := k.clientKeeper.GetClientState(ctx, connectionEnd.Counterparty.ClientId)

			////////////////////////////////////////////////////////////////////////////////////////////////
			// NOTE: If the clientState is found and type is virtual the do a timeout inclusion check. This
			// is a hack to work around the fact that virtual chains may not support non-inclusion proofs.
			////////////////////////////////////////////////////////////////////////////////////////////////
			if found && clientState.ClientType() == "virtual" {
				value = commitment
			}
			err = mh.VerifyMultihopProof(k.cdc, consensusState, channel.ConnectionHops, proof, prefix, key, value)
		} else {
			err = k.connectionKeeper.VerifyPacketReceiptAbsence(
				ctx, connectionEnd, proofHeight, proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
			)
		}
	default:
		panic(sdkerrors.Wrapf(types.ErrInvalidChannelOrdering, channel.Ordering.String()))
	}

	if err != nil {
		return err
	}

	// NOTE: the remaining code is located in the TimeoutExecuted function
	return nil
}

// TimeoutExecuted deletes the commitment send from this chain after it verifies timeout.
// If the timed-out packet came from an ORDERED channel then this channel will be closed.
//
// CONTRACT: this function must be called in the IBC handler
func (k Keeper) TimeoutExecuted(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
) error {
	channel, found := k.GetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return sdkerrors.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", packet.GetSourcePort(), packet.GetSourceChannel())
	}

	capName := host.ChannelCapabilityPath(packet.GetSourcePort(), packet.GetSourceChannel())
	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, capName) {
		return sdkerrors.Wrapf(
			types.ErrChannelCapabilityNotFound,
			"caller does not own capability for channel with capability name %s", capName,
		)
	}

	k.deletePacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if channel.Ordering == types.ORDERED {
		channel.State = types.CLOSED
		k.SetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), channel)
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
	EmitTimeoutPacketEvent(ctx, packet, channel)

	if channel.Ordering == types.ORDERED && channel.State == types.CLOSED {
		EmitChannelClosedEvent(ctx, packet, channel)
	}

	return nil
}

// TimeoutOnClose is called by a module in order to prove that the channel to
// which an unreceived packet was addressed has been closed, so the packet will
// never be received (even if the timeoutHeight has not yet been reached).
func (k Keeper) TimeoutOnClose(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
	proof,
	proofClosed []byte,
	proofHeight exported.Height,
	nextSequenceRecv uint64,
) error {
	channel, found := k.GetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return sdkerrors.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", packet.GetSourcePort(), packet.GetSourceChannel())
	}

	capName := host.ChannelCapabilityPath(packet.GetSourcePort(), packet.GetSourceChannel())
	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, capName) {
		return sdkerrors.Wrapf(
			types.ErrInvalidChannelCapability,
			"channel capability failed authentication with capability name %s", capName,
		)
	}

	if packet.GetDestPort() != channel.Counterparty.PortId {
		return sdkerrors.Wrapf(
			types.ErrInvalidPacket,
			"packet destination port doesn't match the counterparty's port (%s ≠ %s)", packet.GetDestPort(), channel.Counterparty.PortId,
		)
	}

	if packet.GetDestChannel() != channel.Counterparty.ChannelId {
		return sdkerrors.Wrapf(
			types.ErrInvalidPacket,
			"packet destination channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetDestChannel(), channel.Counterparty.ChannelId,
		)
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return sdkerrors.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	commitment := k.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if len(commitment) == 0 {
		EmitTimeoutPacketEvent(ctx, packet, channel)
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return types.ErrNoOpMsg
	}

	packetCommitment := types.CommitPacket(k.cdc, packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return sdkerrors.Wrapf(types.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	var mProof types.MsgMultihopProofs
	var counterpartyHops []string
	if len(channel.ConnectionHops) > 1 {
		var err error
		if err = k.cdc.Unmarshal(proofClosed, &mProof); err != nil {
			return err
		}
		counterpartyHops, err = mProof.GetCounterpartyHops(k.cdc, &connectionEnd)
		if err != nil {
			return err
		}
	} else {
		counterpartyHops = []string{connectionEnd.GetCounterparty().GetConnectionID()}
	}

	counterparty := types.NewCounterparty(packet.GetSourcePort(), packet.GetSourceChannel())
	expectedChannel := types.NewChannel(
		types.CLOSED, channel.Ordering, counterparty, counterpartyHops, channel.Version,
	)

	if len(channel.ConnectionHops) > 1 {

		// expected value bytes
		value, err := expectedChannel.Marshal()
		if err != nil {
			return err
		}

		// verify multihop proof
		consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, connectionEnd.ClientId, proofHeight)
		if !found {
			return sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound,
				"consensus state not found for client id: %s", connectionEnd.ClientId)
		}

		multihopConnectionEnd, err := mProof.GetMultihopConnectionEnd(k.cdc)
		if err != nil {
			return err
		}

		key := host.ChannelPath(counterparty.PortId, counterparty.ChannelId)
		prefix := multihopConnectionEnd.GetCounterparty().GetPrefix()

		if err := mh.VerifyMultihopProof(k.cdc, consensusState, channel.ConnectionHops, proofClosed, prefix, key, value); err != nil {
			return err
		}
	} else {
		// check that the opposing channel end has closed
		if err := k.connectionKeeper.VerifyChannelState(
			ctx, connectionEnd, proofHeight, proofClosed,
			channel.Counterparty.PortId, channel.Counterparty.ChannelId,
			expectedChannel,
		); err != nil {
			return err
		}
	}

	var err error
	switch channel.Ordering {
	case types.ORDERED:
		// check that packet has not been received
		if nextSequenceRecv > packet.GetSequence() {
			return sdkerrors.Wrapf(types.ErrInvalidPacket, "packet already received, next sequence receive > packet sequence (%d > %d", nextSequenceRecv, packet.GetSequence())
		}

		// check that the recv sequence is as claimed
		if len(channel.ConnectionHops) > 1 {
			// verify multihop proof
			consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, connectionEnd.ClientId, proofHeight)
			if !found {
				err = sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound,
					"consensus state not found for client id: %s", connectionEnd.ClientId)
			}
			key := host.NextSequenceRecvPath(packet.GetSourcePort(), packet.GetSourceChannel())
			prefix := connectionEnd.GetCounterparty().GetPrefix()
			val := sdk.Uint64ToBigEndian(nextSequenceRecv)
			err = mh.VerifyMultihopProof(k.cdc, consensusState, channel.ConnectionHops, proof, prefix, key, val)
		} else {
			err = k.connectionKeeper.VerifyNextSequenceRecv(
				ctx, connectionEnd, proofHeight, proof,
				packet.GetDestPort(), packet.GetDestChannel(), nextSequenceRecv,
			)
		}
	case types.UNORDERED:
		if len(channel.ConnectionHops) > 1 {
			// verify multihop proof
			consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, connectionEnd.ClientId, proofHeight)
			if !found {
				err = sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound,
					"consensus state not found for client id: %s", connectionEnd.ClientId)
			}
			key := host.PacketReceiptPath(
				packet.GetSourcePort(),
				packet.GetSourceChannel(),
				packet.GetSequence(),
			)
			prefix := connectionEnd.GetCounterparty().GetPrefix()
			var value []byte = nil
			clientState, found := k.clientKeeper.GetClientState(ctx, connectionEnd.Counterparty.ClientId)

			////////////////////////////////////////////////////////////////////////////////////////////////
			// NOTE: If the clientState is found and type is virtual the do a timeout inclusion check. This
			// is a hack to work around the fact that virtual chains may not support non-inclusion proofs.
			////////////////////////////////////////////////////////////////////////////////////////////////
			if found && clientState.ClientType() == "virtual" {
				value = commitment
			}

			err = mh.VerifyMultihopProof(k.cdc, consensusState, channel.ConnectionHops, proof, prefix, key, value)
		} else {
			err = k.connectionKeeper.VerifyPacketReceiptAbsence(
				ctx, connectionEnd, proofHeight, proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
			)
		}
	default:
		panic(sdkerrors.Wrapf(types.ErrInvalidChannelOrdering, channel.Ordering.String()))
	}

	if err != nil {
		return err
	}

	// NOTE: the remaining code is located in the TimeoutExecuted function
	return nil
}
