package keeper

import (
	"bytes"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clientv2keeper "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/keeper"
	connectionkeeper "github.com/cosmos/ibc-go/v10/modules/core/03-connection/keeper"
	channelkeeperv1 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// Keeper defines the channel keeper v2.
type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.BinaryCodec
	ClientKeeper types.ClientKeeper
	// clientV2Keeper is used for counterparty access.
	clientV2Keeper *clientv2keeper.Keeper

	// channelKeeperV1 is used for channel aliasing only.
	channelKeeperV1  *channelkeeperv1.Keeper
	connectionKeeper *connectionkeeper.Keeper

	// Router is used to route messages to the appropriate module callbacks
	// NOTE: it must be explicitly set before usage.
	Router *api.Router
}

// NewKeeper creates a new channel v2 keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService corestore.KVStoreService,
	clientKeeper types.ClientKeeper,
	clientV2Keeper *clientv2keeper.Keeper,
	channelKeeperV1 *channelkeeperv1.Keeper,
	connectionKeeper *connectionkeeper.Keeper,
) *Keeper {
	return &Keeper{
		storeService:     storeService,
		cdc:              cdc,
		channelKeeperV1:  channelKeeperV1,
		clientV2Keeper:   clientV2Keeper,
		connectionKeeper: connectionKeeper,
		ClientKeeper:     clientKeeper,
	}
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"/"+types.SubModuleName)
}

// GetPacketReceipt returns the packet receipt from the packet receipt path based on the clientID and sequence.
func (k *Keeper) GetPacketReceipt(ctx sdk.Context, clientID string, sequence uint64) ([]byte, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.PacketReceiptKey(clientID, sequence))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return nil, false
	}
	return bz, true
}

// HasPacketReceipt returns true if the packet receipt exists, otherwise false.
func (k *Keeper) HasPacketReceipt(ctx sdk.Context, clientID string, sequence uint64) bool {
	store := k.storeService.OpenKVStore(ctx)
	has, err := store.Has(hostv2.PacketReceiptKey(clientID, sequence))
	if err != nil {
		panic(err)
	}

	return has
}

// SetPacketReceipt writes the packet receipt under the receipt path
// This is a public path that is standardized by the IBC V2 specification.
func (k *Keeper) SetPacketReceipt(ctx sdk.Context, clientID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketReceiptKey(clientID, sequence), []byte{byte(2)}); err != nil {
		panic(err)
	}
}

// GetPacketAcknowledgement fetches the packet acknowledgement from the store.
func (k *Keeper) GetPacketAcknowledgement(ctx sdk.Context, clientID string, sequence uint64) []byte {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.PacketAcknowledgementKey(clientID, sequence))
	if err != nil {
		panic(err)
	}
	return bz
}

// SetPacketAcknowledgement writes the acknowledgement hash under the acknowledgement path
// This is a public path that is standardized by the IBC V2 specification.
func (k *Keeper) SetPacketAcknowledgement(ctx sdk.Context, clientID string, sequence uint64, ackHash []byte) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketAcknowledgementKey(clientID, sequence), ackHash); err != nil {
		panic(err)
	}
}

// HasPacketAcknowledgement checks if the packet ack hash is already on the store.
func (k *Keeper) HasPacketAcknowledgement(ctx sdk.Context, clientID string, sequence uint64) bool {
	return len(k.GetPacketAcknowledgement(ctx, clientID, sequence)) > 0
}

// GetPacketCommitment returns the packet commitment hash under the commitment path.
func (k *Keeper) GetPacketCommitment(ctx sdk.Context, clientID string, sequence uint64) []byte {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.PacketCommitmentKey(clientID, sequence))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return nil
	}
	return bz
}

// SetPacketCommitment writes the commitment hash under the commitment path.
func (k *Keeper) SetPacketCommitment(ctx sdk.Context, clientID string, sequence uint64, commitment []byte) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketCommitmentKey(clientID, sequence), commitment); err != nil {
		panic(err)
	}
}

// DeletePacketCommitment deletes the packet commitment hash under the commitment path.
func (k *Keeper) DeletePacketCommitment(ctx sdk.Context, clientID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(hostv2.PacketCommitmentKey(clientID, sequence)); err != nil {
		panic(err)
	}
}

// GetNextSequenceSend returns the next send sequence from the sequence path
func (k *Keeper) GetNextSequenceSend(ctx sdk.Context, clientID string) (uint64, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.NextSequenceSendKey(clientID))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return 0, false
	}
	return sdk.BigEndianToUint64(bz), true
}

// SetNextSequenceSend writes the next send sequence under the sequence path
func (k *Keeper) SetNextSequenceSend(ctx sdk.Context, clientID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set(hostv2.NextSequenceSendKey(clientID), bigEndianBz); err != nil {
		panic(err)
	}
}

// SetAsyncPacket writes the packet under the async path
func (k *Keeper) SetAsyncPacket(ctx sdk.Context, clientID string, sequence uint64, packet types.Packet) {
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&packet)
	if err := store.Set(types.AsyncPacketKey(clientID, sequence), bz); err != nil {
		panic(err)
	}
}

// GetAsyncPacket fetches the packet from the async path
func (k *Keeper) GetAsyncPacket(ctx sdk.Context, clientID string, sequence uint64) (types.Packet, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.AsyncPacketKey(clientID, sequence))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return types.Packet{}, false
	}
	var packet types.Packet
	k.cdc.MustUnmarshal(bz, &packet)
	return packet, true
}

// DeleteAsyncPacket deletes the packet from the async path
func (k *Keeper) DeleteAsyncPacket(ctx sdk.Context, clientID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.AsyncPacketKey(clientID, sequence)); err != nil {
		panic(err)
	}
}

// extractSequenceFromKey takes the full store key as well as a packet store prefix and extracts
// the encoded sequence number from the key.
//
// This function panics of the provided key once trimmed is larger than 8 bytes as the expected
// sequence byte length is always 8.
func extractSequenceFromKey(key, storePrefix []byte) uint64 {
	sequenceBz := bytes.TrimPrefix(key, storePrefix)
	if len(sequenceBz) > 8 {
		panic("sequence is too long - expected 8 bytes")
	}
	return sdk.BigEndianToUint64(sequenceBz)
}

// GetAllPacketCommitmentsForClient returns all stored PacketCommitments objects for a specified
// client ID.
func (k *Keeper) GetAllPacketCommitmentsForClient(ctx sdk.Context, clientID string) []types.PacketState {
	return k.getAllPacketStateForClient(ctx, clientID, hostv2.PacketCommitmentPrefixKey)
}

// GetAllPacketAcknowledgementsForClient returns all stored PacketAcknowledgements objects for a specified
// client ID.
func (k *Keeper) GetAllPacketAcknowledgementsForClient(ctx sdk.Context, clientID string) []types.PacketState {
	return k.getAllPacketStateForClient(ctx, clientID, hostv2.PacketAcknowledgementPrefixKey)
}

// GetAllPacketReceiptsForClient returns all stored PacketReceipts objects for a specified
// client ID.
func (k *Keeper) GetAllPacketReceiptsForClient(ctx sdk.Context, clientID string) []types.PacketState {
	return k.getAllPacketStateForClient(ctx, clientID, hostv2.PacketReceiptPrefixKey)
}

// GetAllAsyncPacketsForClient returns all stored AsyncPackets objects for a specified
// client ID.
func (k *Keeper) GetAllAsyncPacketsForClient(ctx sdk.Context, clientID string) []types.PacketState {
	return k.getAllPacketStateForClient(ctx, clientID, types.AsyncPacketPrefixKey)
}

// prefixKeyConstructor is a function that constructs a store key for a specific packet store using the provided
// clientID.
type prefixKeyConstructor func(clientID string) []byte

// getAllPacketStateForClient gets all PacketState objects for the specified clientID using a provided
// function for constructing the key prefix for the store.
//
// For example, to get all PacketReceipts for a clientID the hostv2.PacketReceiptPrefixKey function can be
// passed to get the PacketReceipt store key prefix.
func (k *Keeper) getAllPacketStateForClient(ctx sdk.Context, clientID string, prefixFn prefixKeyConstructor) []types.PacketState {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	storePrefix := prefixFn(clientID)
	iterator := storetypes.KVStorePrefixIterator(store, storePrefix)

	var packets []types.PacketState
	for ; iterator.Valid(); iterator.Next() {
		sequence := extractSequenceFromKey(iterator.Key(), storePrefix)
		state := types.NewPacketState(clientID, sequence, iterator.Value())

		packets = append(packets, state)
	}
	return packets
}
