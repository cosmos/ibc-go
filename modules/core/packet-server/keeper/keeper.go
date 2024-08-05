package keeper

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	ChannelKeeper types.ChannelKeeper
	ClientKeeper  types.ClientKeeper
}

func NewKeeper(cdc codec.BinaryCodec, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper) *Keeper {
	return &Keeper{
		cdc:           cdc,
		ChannelKeeper: channelKeeper,
		ClientKeeper:  clientKeeper,
	}
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	// TODO: prefix some submodule identifier?
	return ctx.Logger().With("module", "x/"+exported.ModuleName)
}

func (k Keeper) RecvPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current keeper
	counterparty, ok := k.ClientKeeper.GetCounterparty(ctx, packet.DestinationChannel)
	if !ok {
		return channeltypes.ErrChannelNotFound
	}
	if counterparty.ClientId != packet.SourceChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// check if packet timed out by comparing it with the latest height of the chain
	selfHeight, selfTimestamp := clienttypes.GetSelfHeight(ctx), uint64(ctx.BlockTime().UnixNano())
	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if timeout.Elapsed(selfHeight, selfTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutElapsed(selfHeight, selfTimestamp), "packet timeout elapsed")
	}

	// create key/value pair for proof verification by appending the ICS24 path to the last element of the counterparty merklepath

	// TODO: allow for custom prefix
	path := host.PacketCommitmentKey(packet.SourcePort, packet.SourceChannel, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	commitment := channeltypes.CommitPacket(packet)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		packet.DestinationChannel,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		commitment,
	); err != nil {
		return err
	}

	// REPLAY PROTECTION: Packet receipts will indicate that a packet has already been received
	// on unordered channels. Packet receipts must not be pruned, unless it has been marked stale
	// by the increase of the recvStartSequence.
	_, found := k.ChannelKeeper.GetPacketReceipt(ctx, packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	if found {
		channelkeeper.EmitRecvPacketEvent(ctx, packet, nil)
		// This error indicates that the packet has already been relayed. Core IBC will
		// treat this error as a no-op in order to prevent an entire relay transaction
		// from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	// Set Packet Receipt to prevent timeout from occurring on counterparty
	k.ChannelKeeper.SetPacketReceipt(ctx, packet.DestinationPort, packet.DestinationChannel, packet.Sequence)

	// log that a packet has been received & executed
	k.Logger(ctx).Info("packet received", "sequence", strconv.FormatUint(packet.GetSequence(), 10), "src_port", packet.GetSourcePort(), "src_channel", packet.GetSourceChannel(), "dst_port", packet.GetDestPort(), "dst_channel", packet.GetDestChannel())

	// emit the same events as receive packet without channel fields
	channelkeeper.EmitRecvPacketEvent(ctx, packet, nil)

	return nil
}
