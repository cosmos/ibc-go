package keeper

import (
	"context"
	"fmt"
	"strconv"

	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	connectionkeeper "github.com/cosmos/ibc-go/v9/modules/core/03-connection/keeper"
	channelkeeperv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmentv2types "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// Keeper defines the channel keeper v2.
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService corestore.KVStoreService
	ClientKeeper types.ClientKeeper
	// channelKeeperV1 is used for channel aliasing only.
	channelKeeperV1  *channelkeeperv1.Keeper
	connectionKeeper *connectionkeeper.Keeper

	// Router is used to route messages to the appropriate module callbacks
	// NOTE: it must be explicitly set before usage.
	Router *api.Router
}

// NewKeeper creates a new channel v2 keeper
func NewKeeper(cdc codec.BinaryCodec, storeService corestore.KVStoreService, clientKeeper types.ClientKeeper, channelKeeperV1 *channelkeeperv1.Keeper, connectionKeeper *connectionkeeper.Keeper) *Keeper {
	return &Keeper{
		cdc:              cdc,
		storeService:     storeService,
		channelKeeperV1:  channelKeeperV1,
		connectionKeeper: connectionKeeper,
		ClientKeeper:     clientKeeper,
	}
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: https://github.com/cosmos/ibc-go/issues/5917
	return sdkCtx.Logger().With("module", "x/"+exported.ModuleName+"/"+types.SubModuleName)
}

func (k Keeper) ChannelStore(ctx context.Context, channelID string) storetypes.KVStore {
	channelPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyChannelStorePrefix, channelID))
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), channelPrefix)
}

// SetCounterparty sets the Counterparty for a given client identifier.
func (k *Keeper) SetCounterparty(ctx context.Context, clientID string, counterparty types.Counterparty) {
	bz := k.cdc.MustMarshal(&counterparty)
	k.ChannelStore(ctx, clientID).Set([]byte(types.CounterpartyKey), bz)
}

// GetCounterparty gets the Counterparty for a given client identifier.
func (k *Keeper) GetCounterparty(ctx context.Context, clientID string) (types.Counterparty, bool) {
	store := k.ChannelStore(ctx, clientID)
	bz := store.Get([]byte(types.CounterpartyKey))
	if len(bz) == 0 {
		return types.Counterparty{}, false
	}

	var counterparty types.Counterparty
	k.cdc.MustUnmarshal(bz, &counterparty)
	return counterparty, true
}

// GetPacketReceipt returns the packet receipt from the packet receipt path based on the sourceID and sequence.
func (k *Keeper) GetPacketReceipt(ctx context.Context, sourceID string, sequence uint64) (string, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	bz, err := store.Get(hostv2.PacketReceiptKey(sourceID, bigEndianBz))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return "", false
	}
	return string(bz), true
}

// SetPacketReceipt writes the packet receipt under the receipt path
// This is a public path that is standardized by the IBC V2 specification.
func (k *Keeper) SetPacketReceipt(ctx context.Context, sourceID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set(hostv2.PacketReceiptKey(sourceID, bigEndianBz), []byte{byte(1)}); err != nil {
		panic(err)
	}
}

// SetPacketAcknowledgement writes the acknowledgement hash under the acknowledgement path
// This is a public path that is standardized by the IBC V2 specification.
func (k *Keeper) SetPacketAcknowledgement(ctx context.Context, sourceID string, sequence uint64, ackHash []byte) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set(hostv2.PacketAcknowledgementKey(sourceID, bigEndianBz), ackHash); err != nil {
		panic(err)
	}
}

// HasPacketAcknowledgement check if the packet ack hash is already on the store.
func (k *Keeper) HasPacketAcknowledgement(ctx context.Context, sourceID string, sequence uint64) bool {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	found, err := store.Has(hostv2.PacketAcknowledgementKey(sourceID, bigEndianBz))
	if err != nil {
		panic(err)
	}

	return found
}

// SetInFlightAcknowledgement sets the ack which has not be fully processed.
func (k *Keeper) SetInFlightAcknowledgement(ctx context.Context, destID string, sequence uint64, ack types.Acknowledgement) {
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&ack)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set(hostv2.InFlightAckKey(destID, bigEndianBz), bz); err != nil {
		panic(err)
	}
}

// GetInFlightAcknowledgement gets the ack result which has not be fully processed.
func (k *Keeper) GetInFlightAcknowledgement(ctx context.Context, destID string, sequence uint64) (types.Acknowledgement, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	bz, err := store.Get(hostv2.InFlightAckKey(destID, bigEndianBz))
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return types.Acknowledgement{}, false
	}

	var res types.Acknowledgement
	k.cdc.MustUnmarshal(bz, &res)
	return res, true
}

// GetPacketCommitment returns the packet commitment hash under the commitment path.
func (k *Keeper) GetPacketCommitment(ctx context.Context, sourceID string, sequence uint64) (string, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	bz, err := store.Get(hostv2.PacketCommitmentKey(sourceID, bigEndianBz))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return "", false
	}
	return string(bz), true
}

// SetPacketCommitment writes the commitment hash under the commitment path.
func (k *Keeper) SetPacketCommitment(ctx context.Context, sourceID string, sequence uint64, commitment []byte) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set(hostv2.PacketCommitmentKey(sourceID, bigEndianBz), commitment); err != nil {
		panic(err)
	}
}

// DeletePacketCommitment deletes the packet commitment hash under the commitment path.
func (k *Keeper) DeletePacketCommitment(ctx context.Context, sourceID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Delete(hostv2.PacketCommitmentKey(sourceID, bigEndianBz)); err != nil {
		panic(err)
	}
}

// GetNextSequenceSend returns the next send sequence from the sequence path
func (k *Keeper) GetNextSequenceSend(ctx context.Context, sourceID string) (uint64, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.NextSequenceSendKey(sourceID))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return 0, false
	}
	return sdk.BigEndianToUint64(bz), true
}

// SetNextSequenceSend writes the next send sequence under the sequence path
func (k *Keeper) SetNextSequenceSend(ctx context.Context, sourceID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set(hostv2.NextSequenceSendKey(sourceID), bigEndianBz); err != nil {
		panic(err)
	}
}

// AliasV1Channel returns a version 2 channel for the given port and channel ID
// by converting the channel into a version 2 channel.
func (k *Keeper) AliasV1Channel(ctx context.Context, portID, channelID string) (types.Counterparty, bool) {
	channel, ok := k.channelKeeperV1.GetChannel(ctx, portID, channelID)
	if !ok {
		return types.Counterparty{}, false
	}
	// Do not allow channel to be converted into a version 2 counterparty
	// if the channel is not OPEN or if it is ORDERED
	if channel.State != channeltypesv1.OPEN || channel.Ordering == channeltypesv1.ORDERED {
		return types.Counterparty{}, false
	}
	connection, ok := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !ok {
		return types.Counterparty{}, false
	}
	merklePathPrefix := commitmentv2types.NewMerklePath(connection.Counterparty.Prefix.KeyPrefix, []byte(""))

	counterparty := types.Counterparty{
		CounterpartyChannelId: channel.Counterparty.ChannelId,
		ClientId:              connection.ClientId,
		MerklePathPrefix:      merklePathPrefix,
	}
	return counterparty, true
}

// WriteAcknowledgement writes the acknowledgement to the store. In the synchronous case, this is done
// in the core IBC handler. Async applications should call WriteAcknowledgementAsync to update
// the RecvPacketResult of the relevant application's recvResult.
func (k Keeper) WriteAcknowledgement(
	ctx context.Context,
	packet types.Packet,
	ack types.Acknowledgement,
) error {
	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packet.DestinationId)
	if !ok {
		// TODO: figure out how aliasing will work when more than one packet data is sent.
		counterparty, ok = k.getV1Counterparty(ctx, packet.Data[0].DestinationPort, packet.DestinationId)
		if !ok {
			return errorsmod.Wrap(types.ErrCounterpartyNotFound, packet.DestinationId)
		}
	}

	if counterparty.ClientId != packet.SourceId {
		return channeltypesv1.ErrInvalidChannelIdentifier
	}

	// NOTE: IBC app modules might have written the acknowledgement synchronously on
	// the OnRecvPacket callback so we need to check if the acknowledgement is already
	// set on the store and return an error if so.
	if k.HasPacketAcknowledgement(ctx, packet.DestinationId, packet.Sequence) {
		return channeltypesv1.ErrAcknowledgementExists
	}

	if _, found := k.GetPacketReceipt(ctx, packet.DestinationId, packet.Sequence); !found {
		return errorsmod.Wrap(channeltypesv1.ErrInvalidPacket, "receipt not found for packet")
	}

	multiAckBz := k.cdc.MustMarshal(&ack)
	// set the acknowledgement so that it can be verified on the other side
	k.SetPacketAcknowledgement(
		ctx, packet.DestinationId, packet.GetSequence(),
		channeltypesv1.CommitAcknowledgement(multiAckBz),
	)

	k.Logger(ctx).Info("acknowledgement written", "sequence", strconv.FormatUint(packet.Sequence, 10), "dst_id", packet.DestinationId)

	// TODO: figure out events, we MUST emit the MultiAck structure here
	// channelkeeper.EmitWriteAcknowledgementEventV2(ctx, packet, sentinelChannel(packet.DestinationChannel), multiAck)

	return nil
}
