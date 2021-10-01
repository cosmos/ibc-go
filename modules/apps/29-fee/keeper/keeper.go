package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	host "github.com/cosmos/ibc-go/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
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
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	bankKeeper    types.BankKeeper
}

// NewKeeper creates a new 29-fee Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper, authKeeper types.AccountKeeper, bankKeeper types.BankKeeper,
) Keeper {

	return Keeper{
		cdc:           cdc,
		storeKey:      key,
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

// ChanCloseInit wraps the channel keeper's function in order to expose it to underlying app.
func (k Keeper) ChanCloseInit(ctx sdk.Context, portID, channelID string, chanCap *capabilitytypes.Capability) error {
	return k.channelKeeper.ChanCloseInit(ctx, portID, channelID, chanCap)
}

// GetChannel wraps IBC ChannelKeeper's GetChannel function
func (k Keeper) GetChannel(ctx sdk.Context, portID, channelID string) (channeltypes.Channel, bool) {
	return k.channelKeeper.GetChannel(ctx, portID, channelID)
}

// GetNextSequenceSend wraps IBC ChannelKeeper's GetNextSequenceSend function
func (k Keeper) GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool) {
	return k.channelKeeper.GetNextSequenceSend(ctx, portID, channelID)
}

// GetFeeAccount returns the ICS29 - fee ModuleAccount
func (k Keeper) GetFeeAccount(ctx sdk.Context) authtypes.ModuleAccountI {
	return k.authKeeper.GetModuleAccount(ctx, types.ModuleName)
}

func (k Keeper) GetFeeModuleAddress() sdk.AccAddress {
	return k.authKeeper.GetModuleAddress(types.ModuleName)
}

// SendPacket wraps IBC ChannelKeeper's SendPacket function
func (k Keeper) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
	return k.channelKeeper.SendPacket(ctx, chanCap, packet)
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

// SetCounterpartyAddress maps the destination chain relayer address to the source relayer address
// The receiving chain must store the mapping from: address -> counterpartyAddress for the given channel
func (k Keeper) SetCounterpartyAddress(ctx sdk.Context, address, counterpartyAddress string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyRelayerAddress(address), []byte(counterpartyAddress))
}

// GetCounterpartyAddress gets the relayer counterparty address given a destination relayer address
func (k Keeper) GetCounterpartyAddress(ctx sdk.Context, address sdk.AccAddress) (sdk.AccAddress, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyRelayerAddress(address.String())

	if !store.Has(key) {
		return []byte{}, false
	}

	return store.Get(key), true
}
