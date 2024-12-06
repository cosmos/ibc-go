package keeper

import (
	"context"
	"fmt"

	corestore "cosmossdk.io/core/store"
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
	channelPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyChannelPrefix, channelID))
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), channelPrefix)
}

// SetChannel sets the Channel for a given channel identifier.
func (k *Keeper) SetChannel(ctx context.Context, channelID string, channel types.Channel) {
	bz := k.cdc.MustMarshal(&channel)
	k.ChannelStore(ctx, channelID).Set([]byte(types.ChannelKey), bz)
}

// GetChannel gets the Channel for a given channel identifier.
func (k *Keeper) GetChannel(ctx context.Context, channelID string) (types.Channel, bool) {
	store := k.ChannelStore(ctx, channelID)
	bz := store.Get([]byte(types.ChannelKey))
	if len(bz) == 0 {
		return types.Channel{}, false
	}

	var channel types.Channel
	k.cdc.MustUnmarshal(bz, &channel)
	return channel, true
}

// HasChannel returns true if a Channel exists for a given channel identifier, otherwise false.
func (k *Keeper) HasChannel(ctx context.Context, channelID string) bool {
	store := k.ChannelStore(ctx, channelID)
	return store.Has([]byte(types.ChannelKey))
}

// GetCreator returns the creator of the channel.
func (k *Keeper) GetCreator(ctx context.Context, channelID string) (string, bool) {
	bz := k.ChannelStore(ctx, channelID).Get([]byte(types.CreatorKey))
	if len(bz) == 0 {
		return "", false
	}

	return string(bz), true
}

// SetCreator sets the creator of the channel.
func (k *Keeper) SetCreator(ctx context.Context, channelID, creator string) {
	k.ChannelStore(ctx, channelID).Set([]byte(types.CreatorKey), []byte(creator))
}

// DeleteCreator deletes the creator associated with the channel.
func (k *Keeper) DeleteCreator(ctx context.Context, channelID string) {
	k.ChannelStore(ctx, channelID).Delete([]byte(types.CreatorKey))
}

// GetPacketReceipt returns the packet receipt from the packet receipt path based on the channelID and sequence.
func (k *Keeper) GetPacketReceipt(ctx context.Context, channelID string, sequence uint64) ([]byte, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.PacketReceiptKey(channelID, sequence))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return nil, false
	}
	return bz, true
}

// HasPacketRceipt returns true if the packet receipt exists, otherwise false.
func (k *Keeper) HasPacketReceipt(ctx context.Context, channelID string, sequence uint64) bool {
	store := k.storeService.OpenKVStore(ctx)
	has, err := store.Has(hostv2.PacketReceiptKey(channelID, sequence))
	if err != nil {
		panic(err)
	}

	return has
}

// SetPacketReceipt writes the packet receipt under the receipt path
// This is a public path that is standardized by the IBC V2 specification.
func (k *Keeper) SetPacketReceipt(ctx context.Context, channelID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketReceiptKey(channelID, sequence), []byte{byte(2)}); err != nil {
		panic(err)
	}
}

// GetPacketAcknowledgement fetches the packet acknowledgement from the store.
func (k *Keeper) GetPacketAcknowledgement(ctx context.Context, channelID string, sequence uint64) []byte {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.PacketAcknowledgementKey(channelID, sequence))
	if err != nil {
		panic(err)
	}
	return bz
}

// SetPacketAcknowledgement writes the acknowledgement hash under the acknowledgement path
// This is a public path that is standardized by the IBC V2 specification.
func (k *Keeper) SetPacketAcknowledgement(ctx context.Context, channelID string, sequence uint64, ackHash []byte) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketAcknowledgementKey(channelID, sequence), ackHash); err != nil {
		panic(err)
	}
}

// HasPacketAcknowledgement check if the packet ack hash is already on the store.
func (k *Keeper) HasPacketAcknowledgement(ctx context.Context, channelID string, sequence uint64) bool {
	return len(k.GetPacketAcknowledgement(ctx, channelID, sequence)) > 0
}

// GetPacketCommitment returns the packet commitment hash under the commitment path.
func (k *Keeper) GetPacketCommitment(ctx context.Context, channelID string, sequence uint64) []byte {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.PacketCommitmentKey(channelID, sequence))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return nil
	}
	return bz
}

// SetPacketCommitment writes the commitment hash under the commitment path.
func (k *Keeper) SetPacketCommitment(ctx context.Context, channelID string, sequence uint64, commitment []byte) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketCommitmentKey(channelID, sequence), commitment); err != nil {
		panic(err)
	}
}

// DeletePacketCommitment deletes the packet commitment hash under the commitment path.
func (k *Keeper) DeletePacketCommitment(ctx context.Context, channelID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(hostv2.PacketCommitmentKey(channelID, sequence)); err != nil {
		panic(err)
	}
}

// GetNextSequenceSend returns the next send sequence from the sequence path
func (k *Keeper) GetNextSequenceSend(ctx context.Context, channelID string) (uint64, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.NextSequenceSendKey(channelID))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return 0, false
	}
	return sdk.BigEndianToUint64(bz), true
}

// SetNextSequenceSend writes the next send sequence under the sequence path
func (k *Keeper) SetNextSequenceSend(ctx context.Context, channelID string, sequence uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set(hostv2.NextSequenceSendKey(channelID), bigEndianBz); err != nil {
		panic(err)
	}
}

// aliasV1Channel returns a version 2 channel for the given port and channel ID
// by converting the channel into a version 2 channel.
func (k *Keeper) aliasV1Channel(ctx context.Context, portID, channelID string) (types.Channel, bool) {
	channel, ok := k.channelKeeperV1.GetChannel(ctx, portID, channelID)
	if !ok {
		return types.Channel{}, false
	}
	// Do not allow channel to be converted into a version 2 channel
	// if the channel is not OPEN or if it is ORDERED
	if channel.State != channeltypesv1.OPEN || channel.Ordering == channeltypesv1.ORDERED {
		return types.Channel{}, false
	}
	connection, ok := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !ok {
		return types.Channel{}, false
	}
	merklePathPrefix := commitmentv2types.NewMerklePath(connection.Counterparty.Prefix.KeyPrefix, []byte(""))

	channelv2 := types.Channel{
		CounterpartyChannelId: channel.Counterparty.ChannelId,
		ClientId:              connection.ClientId,
		MerklePathPrefix:      merklePathPrefix,
	}
	return channelv2, true
}

// convertV1Channel attempts to retrieve a v1 channel from the channel keeper if it exists, then converts it
// to a v2 counterparty and stores it in the v2 channel keeper for future use
func (k *Keeper) convertV1Channel(ctx context.Context, port, id string) (types.Channel, bool) {
	if channel, ok := k.aliasV1Channel(ctx, port, id); ok {
		// we can key on just the channel here since channel ids are globally unique
		k.SetChannel(ctx, id, channel)
		return channel, true
	}

	return types.Channel{}, false
}
