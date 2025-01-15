package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectionkeeper "github.com/cosmos/ibc-go/v9/modules/core/03-connection/keeper"
	channelkeeperv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmentv2types "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// Keeper defines the channel keeper v2.
type Keeper struct {
	appmodule.Environment

	cdc          codec.BinaryCodec
	ClientKeeper types.ClientKeeper
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
	env appmodule.Environment,
	clientKeeper types.ClientKeeper,
	channelKeeperV1 *channelkeeperv1.Keeper,
	connectionKeeper *connectionkeeper.Keeper,
) *Keeper {
	return &Keeper{
		Environment:      env,
		cdc:              cdc,
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

// channelStore returns the KV store under which channels are stored.
func (k Keeper) channelStore(ctx context.Context) storetypes.KVStore {
	channelPrefix := []byte(fmt.Sprintf("%s/", types.ChannelPrefix))
	return prefix.NewStore(runtime.KVStoreAdapter(k.KVStoreService.OpenKVStore(ctx)), channelPrefix)
}

// creatorStore returns the KV store under which creators are stored.
func (k Keeper) creatorStore(ctx context.Context) storetypes.KVStore {
	creatorPrefix := []byte(fmt.Sprintf("%s/", types.CreatorPrefix))
	return prefix.NewStore(runtime.KVStoreAdapter(k.KVStoreService.OpenKVStore(ctx)), creatorPrefix)
}

// SetChannel sets the Channel for a given channel identifier.
func (k *Keeper) SetChannel(ctx context.Context, clientID string, channel types.Channel) {
	bz := k.cdc.MustMarshal(&channel)
	k.channelStore(ctx).Set([]byte(clientID), bz)
}

// GetChannel gets the Channel for a given channel identifier.
func (k *Keeper) GetChannel(ctx context.Context, clientID string) (types.Channel, bool) {
	store := k.channelStore(ctx)
	bz := store.Get([]byte(clientID))
	if len(bz) == 0 {
		return types.Channel{}, false
	}

	var channel types.Channel
	k.cdc.MustUnmarshal(bz, &channel)
	return channel, true
}

// HasChannel returns true if a Channel exists for a given channel identifier, otherwise false.
func (k *Keeper) HasChannel(ctx context.Context, clientID string) bool {
	store := k.channelStore(ctx)
	return store.Has([]byte(clientID))
}

// GetCreator returns the creator of the channel.
func (k *Keeper) GetCreator(ctx context.Context, clientID string) (string, bool) {
	bz := k.creatorStore(ctx).Get([]byte(clientID))
	if len(bz) == 0 {
		return "", false
	}

	return string(bz), true
}

// SetCreator sets the creator of the channel.
func (k *Keeper) SetCreator(ctx context.Context, clientID, creator string) {
	k.creatorStore(ctx).Set([]byte(clientID), []byte(creator))
}

// DeleteCreator deletes the creator associated with the channel.
func (k *Keeper) DeleteCreator(ctx context.Context, clientID string) {
	k.creatorStore(ctx).Delete([]byte(clientID))
}

// GetPacketReceipt returns the packet receipt from the packet receipt path based on the clientID and sequence.
func (k *Keeper) GetPacketReceipt(ctx context.Context, clientID string, sequence uint64) ([]byte, bool) {
	store := k.KVStoreService.OpenKVStore(ctx)
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
func (k *Keeper) HasPacketReceipt(ctx context.Context, clientID string, sequence uint64) bool {
	store := k.KVStoreService.OpenKVStore(ctx)
	has, err := store.Has(hostv2.PacketReceiptKey(clientID, sequence))
	if err != nil {
		panic(err)
	}

	return has
}

// SetPacketReceipt writes the packet receipt under the receipt path
// This is a public path that is standardized by the IBC V2 specification.
func (k *Keeper) SetPacketReceipt(ctx context.Context, clientID string, sequence uint64) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketReceiptKey(clientID, sequence), []byte{byte(2)}); err != nil {
		panic(err)
	}
}

// GetPacketAcknowledgement fetches the packet acknowledgement from the store.
func (k *Keeper) GetPacketAcknowledgement(ctx context.Context, clientID string, sequence uint64) []byte {
	store := k.KVStoreService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.PacketAcknowledgementKey(clientID, sequence))
	if err != nil {
		panic(err)
	}
	return bz
}

// SetPacketAcknowledgement writes the acknowledgement hash under the acknowledgement path
// This is a public path that is standardized by the IBC V2 specification.
func (k *Keeper) SetPacketAcknowledgement(ctx context.Context, clientID string, sequence uint64, ackHash []byte) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketAcknowledgementKey(clientID, sequence), ackHash); err != nil {
		panic(err)
	}
}

// HasPacketAcknowledgement checks if the packet ack hash is already on the store.
func (k *Keeper) HasPacketAcknowledgement(ctx context.Context, clientID string, sequence uint64) bool {
	return len(k.GetPacketAcknowledgement(ctx, clientID, sequence)) > 0
}

// GetPacketCommitment returns the packet commitment hash under the commitment path.
func (k *Keeper) GetPacketCommitment(ctx context.Context, clientID string, sequence uint64) []byte {
	store := k.KVStoreService.OpenKVStore(ctx)
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
func (k *Keeper) SetPacketCommitment(ctx context.Context, clientID string, sequence uint64, commitment []byte) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Set(hostv2.PacketCommitmentKey(clientID, sequence), commitment); err != nil {
		panic(err)
	}
}

// DeletePacketCommitment deletes the packet commitment hash under the commitment path.
func (k *Keeper) DeletePacketCommitment(ctx context.Context, clientID string, sequence uint64) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Delete(hostv2.PacketCommitmentKey(clientID, sequence)); err != nil {
		panic(err)
	}
}

// GetNextSequenceSend returns the next send sequence from the sequence path
func (k *Keeper) GetNextSequenceSend(ctx context.Context, clientID string) (uint64, bool) {
	store := k.KVStoreService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.NextSequenceSendKey(clientID))
	if err != nil {
		panic(err)
	}
	// initialize sequence to 1 if it does not exist
	if len(bz) == 0 {
		k.SetNextSequenceSend(ctx, clientID, 1)
		return 1, true
	}
	return sdk.BigEndianToUint64(bz), true
}

// SetNextSequenceSend writes the next send sequence under the sequence path
func (k *Keeper) SetNextSequenceSend(ctx context.Context, clientID string, sequence uint64) {
	store := k.KVStoreService.OpenKVStore(ctx)
	bigEndianBz := sdk.Uint64ToBigEndian(sequence)
	if err := store.Set(hostv2.NextSequenceSendKey(clientID), bigEndianBz); err != nil {
		panic(err)
	}
}

// aliasV1Channel returns a version 2 channel for the given port and channel ID
// by converting the channel into a version 2 channel.
func (k *Keeper) aliasV1Channel(ctx context.Context, portID, clientID string) (types.Channel, bool) {
	channel, ok := k.channelKeeperV1.GetChannel(ctx, portID, clientID)
	if !ok {
		return types.Channel{}, false
	}
	// Do not allow channel to be converted into a version 2 channel
	// if the channel is not OPEN or if it is not UNORDERED
	if channel.State != channeltypesv1.OPEN || channel.Ordering != channeltypesv1.UNORDERED {
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

// resolveV2Identifiers returns the client identifier and the counterpartyInfo for the client given the packetId
// Note: For fresh eureka channels, the client identifier and packet identifier are the same.
// For aliased channels, the packet identifier will be the original channel ID and the counterpartyInfo will be constructed from the channel
func (k *Keeper) resolveV2Identifiers(ctx context.Context, portId string, packetId string) (string, clienttypes.CounterpartyInfo, error) {
	counterpartyInfo, ok := k.ClientKeeper.GetClientCounterparty(ctx, packetId)
	if !ok {
		channel, ok := k.channelKeeperV1.GetChannel(ctx, portId, packetId)
		if ok {
			connection, ok := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
			if !ok {
				// should never happen since the connection should exist if the channel exists
				return "", clienttypes.CounterpartyInfo{}, types.ErrInvalidChannel
			}
			// convert v1 merkle prefix into the v2 path format
			merklePrefix := [][]byte{connection.Counterparty.Prefix.KeyPrefix, []byte("")}
			// create the counterparty info, here we set the counterparty client Id to the the counterparty channel id
			// this is because we want to preserve the original identifiers that are used to write provable paths to each other
			counterpartyInfo = clienttypes.NewCounterpartyInfo(merklePrefix, channel.Counterparty.ChannelId)
			return connection.ClientId, counterpartyInfo, nil
		} else {
			// neither client nor channel exists so return client not found error
			return "", clienttypes.CounterpartyInfo{}, clienttypes.ErrClientNotFound
		}
	}
	return packetId, counterpartyInfo, nil
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
