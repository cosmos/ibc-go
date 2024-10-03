package keeper

import (
	"context"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// Keeper defines the channel keeper v2.
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService corestore.KVStoreService
	ClientKeeper types.ClientKeeper
}

// NewKeeper creates a new channel v2 keeper
func NewKeeper(cdc codec.BinaryCodec, storeService corestore.KVStoreService, clientKeeper types.ClientKeeper) *Keeper {
	return &Keeper{
		cdc:          cdc,
		storeService: storeService,
		ClientKeeper: clientKeeper,
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
