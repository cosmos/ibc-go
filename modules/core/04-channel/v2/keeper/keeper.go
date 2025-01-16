package keeper

import (
	"context"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectionkeeper "github.com/cosmos/ibc-go/v9/modules/core/03-connection/keeper"
	channelkeeperv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
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
	if len(bz) == 0 {
		return 0, false
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
				return "", clienttypes.CounterpartyInfo{}, channeltypesv1.ErrChannelNotFound
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
