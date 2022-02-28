package keeper

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// Middleware must implement types.ChannelKeeper and types.PortKeeper expected interfaces
// so that it can wrap IBC channel and port logic for underlying application.
var (
	_ types.ChannelKeeper = Keeper{}
	_ types.PortKeeper    = Keeper{}
)

// Keeper defines the IBC fungible transfer keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	authKeeper    types.AccountKeeper
	ics4Wrapper   types.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	bankKeeper    types.BankKeeper
}

// NewKeeper creates a new 29-fee Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	ics4Wrapper types.ICS4Wrapper, channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper, authKeeper types.AccountKeeper, bankKeeper types.BankKeeper,
) Keeper {

	return Keeper{
		cdc:           cdc,
		storeKey:      key,
		ics4Wrapper:   ics4Wrapper,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		authKeeper:    authKeeper,
		bankKeeper:    bankKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+host.ModuleName+"-"+types.ModuleName)
}

// BindPort defines a wrapper function for the port Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability {
	return k.portKeeper.BindPort(ctx, portID)
}

// GetChannel wraps IBC ChannelKeeper's GetChannel function
func (k Keeper) GetChannel(ctx sdk.Context, portID, channelID string) (channeltypes.Channel, bool) {
	return k.channelKeeper.GetChannel(ctx, portID, channelID)
}

// GetNextSequenceSend wraps IBC ChannelKeeper's GetNextSequenceSend function
func (k Keeper) GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool) {
	return k.channelKeeper.GetNextSequenceSend(ctx, portID, channelID)
}

// GetFeeAccount returns the ICS29 Fee ModuleAccount address
func (k Keeper) GetFeeModuleAddress() sdk.AccAddress {
	return k.authKeeper.GetModuleAddress(types.ModuleName)
}

// SetFeeEnabled sets a flag to determine if fee handling logic should run for the given channel
// identified by channel and port identifiers.
func (k Keeper) SetFeeEnabled(ctx sdk.Context, portID, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.FeeEnabledKey(portID, channelID), []byte{1})
}

// DeleteFeeEnabled deletes the fee enabled flag for a given portID and channelID
func (k Keeper) DeleteFeeEnabled(ctx sdk.Context, portID, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.FeeEnabledKey(portID, channelID))
}

// IsFeeEnabled returns whether fee handling logic should be run for the given port. It will check the
// fee enabled flag for the given port and channel identifiers
func (k Keeper) IsFeeEnabled(ctx sdk.Context, portID, channelID string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Get(types.FeeEnabledKey(portID, channelID)) != nil
}

// GetAllFeeEnabledChannels returns a list of all ics29 enabled channels containing portID & channelID that are stored in state
func (k Keeper) GetAllFeeEnabledChannels(ctx sdk.Context) []types.FeeEnabledChannel {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.FeeEnabledKeyPrefix))
	defer iterator.Close()

	var enabledChArr []types.FeeEnabledChannel
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		ch := types.FeeEnabledChannel{
			PortId:    keySplit[1],
			ChannelId: keySplit[2],
		}

		enabledChArr = append(enabledChArr, ch)
	}

	return enabledChArr
}

// DisableAllChannels will disable the fee module for all channels.
// Only called if the module enters into an invalid state
// e.g. ModuleAccount has insufficient balance to refund users.
// In this case, chain developers should investigate the issue, fix it,
// and then re-enable the fee module in a coordinated upgrade.
func (k Keeper) DisableAllChannels(ctx sdk.Context) {
	channels := k.GetAllFeeEnabledChannels(ctx)

	for _, channel := range channels {
		k.DeleteFeeEnabled(ctx, channel.PortId, channel.ChannelId)
	}
}

// SetCounterpartyAddress maps the destination chain relayer address to the source relayer address
// The receiving chain must store the mapping from: address -> counterpartyAddress for the given channel
func (k Keeper) SetCounterpartyAddress(ctx sdk.Context, address, counterpartyAddress, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyCounterpartyRelayer(address, channelID), []byte(counterpartyAddress))
}

// GetCounterpartyAddress gets the relayer counterparty address given a destination relayer address
func (k Keeper) GetCounterpartyAddress(ctx sdk.Context, address, channelID string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyCounterpartyRelayer(address, channelID)

	if !store.Has(key) {
		return "", false
	}

	addr := string(store.Get(key))
	return addr, true
}

// GetAllRelayerAddresses returns all registered relayer addresses
func (k Keeper) GetAllRelayerAddresses(ctx sdk.Context) []types.RegisteredRelayerAddress {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.CounterpartyRelayerAddressKeyPrefix))
	defer iterator.Close()

	var registeredAddrArr []types.RegisteredRelayerAddress
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		addr := types.RegisteredRelayerAddress{
			Address:             keySplit[1],
			CounterpartyAddress: string(iterator.Value()),
			ChannelId:           keySplit[2],
		}

		registeredAddrArr = append(registeredAddrArr, addr)
	}

	return registeredAddrArr
}

// SetRelayerAddressForAsyncAck sets the forward relayer address during OnRecvPacket in case of async acknowledgement
func (k Keeper) SetRelayerAddressForAsyncAck(ctx sdk.Context, packetId channeltypes.PacketId, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyForwardRelayerAddress(packetId), []byte(address))
}

// GetRelayerAddressForAsyncAck gets forward relayer address for a particular packet
func (k Keeper) GetRelayerAddressForAsyncAck(ctx sdk.Context, packetId channeltypes.PacketId) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyForwardRelayerAddress(packetId)
	if !store.Has(key) {
		return "", false
	}

	addr := string(store.Get(key))
	return addr, true
}

// GetAllForwardRelayerAddresses returns all forward relayer addresses stored for async acknowledgements
func (k Keeper) GetAllForwardRelayerAddresses(ctx sdk.Context) []types.ForwardRelayerAddress {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.ForwardRelayerPrefix))
	defer iterator.Close()

	var forwardRelayerAddr []types.ForwardRelayerAddress
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		seq, err := strconv.ParseUint(keySplit[3], 0, 64)
		if err != nil {
			panic("failed to parse packet sequence in forward relayer address mapping")
		}

		packetId := channeltypes.NewPacketId(keySplit[2], keySplit[1], seq)

		addr := types.ForwardRelayerAddress{
			Address:  string(iterator.Value()),
			PacketId: packetId,
		}

		forwardRelayerAddr = append(forwardRelayerAddr, addr)
	}

	return forwardRelayerAddr
}

// Deletes the forwardRelayerAddr associated with the packetId
func (k Keeper) DeleteForwardRelayerAddress(ctx sdk.Context, packetId channeltypes.PacketId) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyForwardRelayerAddress(packetId)
	store.Delete(key)
}

// Stores a Fee for a given packet in state
func (k Keeper) SetFeeInEscrow(ctx sdk.Context, fee types.IdentifiedPacketFee) {
	store := ctx.KVStore(k.storeKey)
	bz := k.MustMarshalFee(&fee)
	store.Set(types.KeyFeeInEscrow(fee.PacketId), bz)
}

// Gets a Fee for a given packet
func (k Keeper) GetFeeInEscrow(ctx sdk.Context, packetId channeltypes.PacketId) (types.IdentifiedPacketFee, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyFeeInEscrow(packetId)
	bz := store.Get(key)
	if bz == nil {
		return types.IdentifiedPacketFee{}, false
	}
	fee := k.MustUnmarshalFee(bz)

	return fee, true
}

// GetFeesInEscrow returns all escrowed packet fees for a given packetID
func (k Keeper) GetFeesInEscrow(ctx sdk.Context, packetID channeltypes.PacketId) (types.PacketFees, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyFeesInEscrow(packetID)
	bz := store.Get(key)
	if bz == nil {
		return types.PacketFees{}, false
	}

	return k.MustUnmarshalFees(bz), true
}

// HasFeesInEscrow returns true if packet fees exist for the provided packetID
func (k Keeper) HasFeesInEscrow(ctx sdk.Context, packetID channeltypes.PacketId) bool {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyFeesInEscrow(packetID)

	return store.Has(key)
}

// SetFeesInEscrow sets the given packet fees in escrow keyed by the packet identifier
func (k Keeper) SetFeesInEscrow(ctx sdk.Context, packetID channeltypes.PacketId, fees types.PacketFees) {
	store := ctx.KVStore(k.storeKey)
	bz := k.MustMarshalFees(fees)
	store.Set(types.KeyFeesInEscrow(packetID), bz)
}

// DeleteFeesInEscrow deletes the fee associated with the given packetID
func (k Keeper) DeleteFeesInEscrow(ctx sdk.Context, packetID channeltypes.PacketId) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyFeesInEscrow(packetID)
	store.Delete(key)
}

// IteratePacketFeesInEscrow iterates over all the fees on the given channel currently escrowed and calls the provided callback
// if the callback returns true, then iteration is stopped.
func (k Keeper) IteratePacketFeesInEscrow(ctx sdk.Context, portID, channelID string, cb func(packetFees types.PacketFees) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.KeyFeesInEscrowChannelPrefix(portID, channelID))

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		packetFees := k.MustUnmarshalFees(iterator.Value())
		if cb(packetFees) {
			break
		}
	}
}

// IterateChannelFeesInEscrow iterates over all the fees on the given channel currently escrowed and calls the provided callback
// if the callback returns true, then iteration is stopped.
func (k Keeper) IterateChannelFeesInEscrow(ctx sdk.Context, portID, channelID string, cb func(identifiedFee types.IdentifiedPacketFee) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.KeyFeeInEscrowChannelPrefix(portID, channelID))

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		identifiedFee := k.MustUnmarshalFee(iterator.Value())
		if cb(identifiedFee) {
			break
		}
	}
}

// Deletes the fee associated with the given packetId
func (k Keeper) DeleteFeeInEscrow(ctx sdk.Context, packetId channeltypes.PacketId) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyFeeInEscrow(packetId)
	store.Delete(key)
}

// HasFeeInEscrow returns true if there is a Fee still to be escrowed for a given packet
func (k Keeper) HasFeeInEscrow(ctx sdk.Context, packetId channeltypes.PacketId) bool {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyFeeInEscrow(packetId)

	return store.Has(key)
}

// GetAllIdentifiedPacketFees returns a list of all IdentifiedPacketFees that are stored in state
func (k Keeper) GetAllIdentifiedPacketFees(ctx sdk.Context) []types.IdentifiedPacketFees {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.FeesInEscrowPrefix))
	defer iterator.Close()

	var identifiedFees []types.IdentifiedPacketFees
	for ; iterator.Valid(); iterator.Next() {
		packetID, err := types.ParseKeyFeesInEscrow(string(iterator.Key()))
		if err != nil {
			panic(err)
		}

		feesInEscrow := k.MustUnmarshalFees(iterator.Value())

		identifiedFee := types.IdentifiedPacketFees{
			PacketId:   packetID,
			PacketFees: feesInEscrow.PacketFees,
		}

		identifiedFees = append(identifiedFees, identifiedFee)
	}

	return identifiedFees
}

// MustMarshalFee attempts to encode a Fee object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalFee(fee *types.IdentifiedPacketFee) []byte {
	return k.cdc.MustMarshal(fee)
}

// MustUnmarshalFee attempts to decode and return a Fee object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalFee(bz []byte) types.IdentifiedPacketFee {
	var fee types.IdentifiedPacketFee
	k.cdc.MustUnmarshal(bz, &fee)
	return fee
}

// MustMarshalFees attempts to encode a Fee object and returns the
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalFees(fees types.PacketFees) []byte {
	return k.cdc.MustMarshal(&fees)
}

// MustUnmarshalFees attempts to decode and return a Fee object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalFees(bz []byte) types.PacketFees {
	var fees types.PacketFees
	k.cdc.MustUnmarshal(bz, &fees)
	return fees
}
