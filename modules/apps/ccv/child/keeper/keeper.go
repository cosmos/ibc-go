package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/child/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/tendermint/tendermint/libs/log"
)

// Keeper defines the Cross-Chain Validation Child Keeper
type Keeper struct {
	storeKey         sdk.StoreKey
	cdc              codec.BinaryCodec
	scopedKeeper     capabilitykeeper.ScopedKeeper
	channelKeeper    ccv.ChannelKeeper
	portKeeper       ccv.PortKeeper
	connectionKeeper ccv.ConnectionKeeper
	clientKeeper     ccv.ClientKeeper
}

// NewKeeper creates a new Child Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, scopedKeeper capabilitykeeper.ScopedKeeper,
	channelKeeper ccv.ChannelKeeper, portKeeper ccv.PortKeeper,
	connectionKeeper ccv.ConnectionKeeper, clientKeeper ccv.ClientKeeper,
) Keeper {
	return Keeper{
		cdc:              cdc,
		storeKey:         key,
		scopedKeeper:     scopedKeeper,
		channelKeeper:    channelKeeper,
		portKeeper:       portKeeper,
		connectionKeeper: connectionKeeper,
		clientKeeper:     clientKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+host.ModuleName+"-"+types.ModuleName)
}

// ChanCloseInit defines a wrapper function for the channel Keeper's function
// in order to expose it to the ICS20 transfer handler.
func (k Keeper) ChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	capName := host.ChannelCapabilityPath(portID, channelID)
	chanCap, ok := k.scopedKeeper.GetCapability(ctx, capName)
	if !ok {
		return sdkerrors.Wrapf(channeltypes.ErrChannelCapabilityNotFound, "could not retrieve channel capability at: %s", capName)
	}
	return k.channelKeeper.ChanCloseInit(ctx, portID, channelID, chanCap)
}

// IsBound checks if the transfer module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the ort Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	cap := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}

// GetPort returns the portID for the transfer module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(types.PortKey))
}

// SetPort sets the portID for the transfer module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.PortKey, []byte(portID))
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return k.scopedKeeper.AuthenticateCapability(ctx, cap, name)
}

// ClaimCapability allows the transfer module that can claim a capability that IBC module
// passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// SetChannelStatus sets the status of a CCV channel with the given status
func (k Keeper) SetChannelStatus(ctx sdk.Context, channelID string, status ccv.Status) {
	store := ctx.KVStore(k.storeKey)
	store.Set(ccv.ChannelStatusKey(channelID), []byte{byte(status)})
}

// GetChannelStatus gets the status of a CCV channel
func (k Keeper) GetChannelStatus(ctx sdk.Context, channelID string) ccv.Status {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(ccv.ChannelStatusKey(channelID))
	if bz == nil {
		return ccv.Uninitialized
	}
	return ccv.Status(bz[0])
}

// SetParentClient sets the parent clientID that is validating the chain.
// Set in InitGenesis
func (k Keeper) SetParentClient(ctx sdk.Context, clientID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ParentClientKey(), []byte(clientID))
}

// GetParentClient gets the parent clientID that is validating the chain.
func (k Keeper) GetParentClient(ctx sdk.Context) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	clientIdBytes := store.Get(types.ParentClientKey())
	if clientIdBytes == nil {
		return "", false
	}
	return string(clientIdBytes), true
}

// SetParentChannel sets the parent channelID that is validating the chain.
func (k Keeper) SetParentChannel(ctx sdk.Context, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ParentChannelKey(), []byte(channelID))
}

// GetParentChannel gets the parent channelID that is validating the chain.
func (k Keeper) GetParentChannel(ctx sdk.Context) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	channelIdBytes := store.Get(types.ParentChannelKey())
	if channelIdBytes == nil {
		return "", false
	}
	return string(channelIdBytes), true
}

// SetPendingChanges sets the pending validator set change packet that haven't been flushed to ABCI
func (k Keeper) SetPendingChanges(ctx sdk.Context, updates ccv.ValidatorSetChangePacketData) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := updates.Marshal()
	if err != nil {
		return err
	}
	store.Set(types.PendingChangesKey(), bz)
	return nil
}

// GetPendingChanges gets the pending changes that haven't been flushed over ABCI
func (k Keeper) GetPendingChanges(ctx sdk.Context) (*ccv.ValidatorSetChangePacketData, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingChangesKey())
	if bz == nil {
		return nil, false
	}
	var data ccv.ValidatorSetChangePacketData
	data.Unmarshal(bz)
	return &data, true
}

// DeletePendingChanges deletes the pending changes after they've been flushed to ABCI
func (k Keeper) DeletePendingChanges(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PendingChangesKey())
}

// IterateUnbondingTime iterates through the unbonding times set in the store
func (k Keeper) IterateUnbondingTime(ctx sdk.Context, cb func(seq, timeNs uint64) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.UnbondingTimePrefix))

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		seqBytes := iterator.Key()[len([]byte(types.UnbondingTimePrefix)):]
		seq := binary.BigEndian.Uint64(seqBytes)

		timeNs := binary.BigEndian.Uint64(iterator.Value())

		if cb(seq, timeNs) {
			break
		}
	}
}

// SetUnbondingTime sets the unbonding time for a given received packet sequence
func (k Keeper) SetUnbondingTime(ctx sdk.Context, sequence, unbondingTime uint64) {
	store := ctx.KVStore(k.storeKey)
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, unbondingTime)
	store.Set(types.UnbondingTimeKey(sequence), timeBytes)
}

// GetUnbondingTime gets the unbonding time for a given received packet sequence
func (k Keeper) GetUnbondingTime(ctx sdk.Context, sequence uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.UnbondingTimeKey(sequence))
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// DeleteUnbondingTime deletes the unbonding time
func (k Keeper) DeleteUnbondingTime(ctx sdk.Context, sequence uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.UnbondingTimeKey(sequence))
}

// IterateUnbondingPacket iterates through the unbonding packets set in the store
func (k Keeper) IterateUnbondingPacket(ctx sdk.Context, cb func(seq uint64, packet channeltypes.Packet) bool) error {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.UnbondingPacketPrefix))

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		seqBytes := iterator.Key()[len([]byte(types.UnbondingPacketPrefix)):]
		seq := binary.BigEndian.Uint64(seqBytes)

		var packet channeltypes.Packet
		err := packet.Unmarshal(iterator.Value())
		if err != nil {
			return err
		}

		if cb(seq, packet) {
			break
		}
	}
	return nil
}

// SetUnbondingPacket sets the unbonding packet for a given received packet sequence
func (k Keeper) SetUnbondingPacket(ctx sdk.Context, sequence uint64, packet channeltypes.Packet) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := packet.Marshal()
	if err != nil {
		return err
	}
	store.Set(types.UnbondingPacketKey(sequence), bz)
	return nil
}

// GetUnbondingPacket gets the unbonding packet for a given received packet sequence
func (k Keeper) GetUnbondingPacket(ctx sdk.Context, sequence uint64) (*channeltypes.Packet, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.UnbondingPacketKey(sequence))
	if bz == nil {
		return nil, sdkerrors.Wrapf(channeltypes.ErrInvalidPacket, "packet does not exist at sequence: %d", sequence)
	}
	var packet channeltypes.Packet
	err := packet.Unmarshal(bz)
	if err != nil {
		return nil, err
	}
	return &packet, nil
}

// DeleteUnbondingPacket deletes the unbonding packet
func (k Keeper) DeleteUnbondingPacket(ctx sdk.Context, sequence uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.UnbondingPacketKey(sequence))
}

// VerifyParentChain verifies that the chain trying to connect on the channel handshake
// is the expected parent chain.
func (k Keeper) VerifyParentChain(ctx sdk.Context, channelID string) error {
	// Verify CCV channel is in Initialized state
	status := k.GetChannelStatus(ctx, channelID)
	if status != ccv.Initializing {
		return sdkerrors.Wrap(ccv.ErrInvalidStatus, "CCV channel status must be in Initializing state")
	}
	// Retrieve the underlying client state.
	channel, ok := k.channelKeeper.GetChannel(ctx, types.PortID, channelID)
	if !ok {
		return sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "channel not found for channel ID: %s", channelID)
	}
	if len(channel.ConnectionHops) == 1 {
		return sdkerrors.Wrap(channeltypes.ErrTooManyConnectionHops, "must have direct connection to parent chain")
	}
	connectionID := channel.ConnectionHops[0]
	conn, ok := k.connectionKeeper.GetConnection(ctx, connectionID)
	if !ok {
		return sdkerrors.Wrapf(conntypes.ErrConnectionNotFound, "connection not found for connection ID: %s", connectionID)
	}
	// Verify that client id is expected clientID
	expectedClientId, ok := k.GetParentClient(ctx)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "could not find parent client id")
	}
	if expectedClientId == conn.ClientId {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "invalid client: %s, channel must be built on top of client: %s", conn.ClientId, expectedClientId)
	}

	return nil
}
