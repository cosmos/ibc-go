package keeper

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// Keeper defines the IBC fungible transfer keeper
type Keeper struct {
	storeKey       storetypes.StoreKey
	cdc            codec.BinaryCodec
	legacySubspace types.ParamSubspace

	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	authKeeper    types.AccountKeeper
	bankKeeper    types.BankKeeper
	scopedKeeper  exported.ScopedKeeper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper creates a new IBC transfer Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	legacySubspace types.ParamSubspace,
	ics4Wrapper porttypes.ICS4Wrapper,
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	authKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	scopedKeeper exported.ScopedKeeper,
	authority string,
) Keeper {
	// ensure ibc transfer module account is set
	if addr := authKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(errors.New("the IBC transfer module account has not been set"))
	}

	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	return Keeper{
		cdc:            cdc,
		storeKey:       key,
		legacySubspace: legacySubspace,
		ics4Wrapper:    ics4Wrapper,
		channelKeeper:  channelKeeper,
		portKeeper:     portKeeper,
		authKeeper:     authKeeper,
		bankKeeper:     bankKeeper,
		scopedKeeper:   scopedKeeper,
		authority:      authority,
	}
}

// WithICS4Wrapper sets the ICS4Wrapper. This function may be used after
// the keepers creation to set the middleware which is above this module
// in the IBC application stack.
func (k *Keeper) WithICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	k.ics4Wrapper = wrapper
}

// GetICS4Wrapper returns the ICS4Wrapper.
func (k Keeper) GetICS4Wrapper() porttypes.ICS4Wrapper {
	return k.ics4Wrapper
}

// GetAuthority returns the transfer module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"-"+types.ModuleName)
}

// hasCapability checks if the transfer module owns the port capability for the desired port
func (k Keeper) hasCapability(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the port Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	capability := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, capability, host.PortPath(portID))
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

// GetParams returns the current transfer module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.ParamsKey))
	if bz == nil { // only panic on unset params and not on empty params
		panic(errors.New("transfer params are not set in store"))
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the transfer module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set([]byte(types.ParamsKey), bz)
}

// GetDenom retrieves the denom from store given the hash of the denom.
func (k Keeper) GetDenom(ctx sdk.Context, denomHash cmtbytes.HexBytes) (types.Denom, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomKey)
	bz := store.Get(denomHash)
	if len(bz) == 0 {
		return types.Denom{}, false
	}

	var denom types.Denom
	k.cdc.MustUnmarshal(bz, &denom)

	return denom, true
}

// HasDenom checks if a the key with the given denomination hash exists on the store.
func (k Keeper) HasDenom(ctx sdk.Context, denomHash cmtbytes.HexBytes) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomKey)
	return store.Has(denomHash)
}

// SetDenom sets a new {denom hash -> denom } pair to the store.
// This allows for reverse lookup of the denom given the hash.
func (k Keeper) SetDenom(ctx sdk.Context, denom types.Denom) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomKey)
	bz := k.cdc.MustMarshal(&denom)
	store.Set(denom.Hash(), bz)
}

// GetAllDenoms returns all the denominations.
func (k Keeper) GetAllDenoms(ctx sdk.Context) types.Denoms {
	denoms := types.Denoms{}
	k.IterateDenoms(ctx, func(denom types.Denom) bool {
		denoms = append(denoms, denom)
		return false
	})

	return denoms.Sort()
}

// IterateDenoms iterates over the denominations in the store and performs a callback function.
func (k Keeper) IterateDenoms(ctx sdk.Context, cb func(denom types.Denom) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.DenomKey)

	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		var denom types.Denom
		k.cdc.MustUnmarshal(iterator.Value(), &denom)

		if cb(denom) {
			break
		}
	}
}

// setDenomMetadata sets an IBC token's denomination metadata
func (k Keeper) setDenomMetadata(ctx sdk.Context, denom types.Denom) {
	metadata := banktypes.Metadata{
		Description: fmt.Sprintf("IBC token from %s", denom.Path()),
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom.Base,
				Exponent: 0,
			},
		},
		// Setting base as IBC hash denom since bank keepers's SetDenomMetadata uses
		// Base as key path and the IBC hash is what gives this token uniqueness
		// on the executing chain
		Base:    denom.IBCDenom(),
		Display: denom.Path(),
		Name:    fmt.Sprintf("%s IBC token", denom.Path()),
		Symbol:  strings.ToUpper(denom.Base),
	}

	k.bankKeeper.SetDenomMetaData(ctx, metadata)
}

// GetTotalEscrowForDenom gets the total amount of source chain tokens that
// are in escrow, keyed by the denomination.
//
// NOTE: if there is no value stored in state for the provided denom then a new Coin is returned for the denom with an initial value of zero.
// This accommodates callers to simply call `Add()` on the returned Coin as an empty Coin literal (e.g. sdk.Coin{}) will trigger a panic due to the absence of a denom.
func (k Keeper) GetTotalEscrowForDenom(ctx sdk.Context, denom string) sdk.Coin {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalEscrowForDenomKey(denom))
	if len(bz) == 0 {
		return sdk.NewCoin(denom, sdkmath.ZeroInt())
	}

	amount := sdk.IntProto{}
	k.cdc.MustUnmarshal(bz, &amount)

	return sdk.NewCoin(denom, amount.Int)
}

// SetTotalEscrowForDenom stores the total amount of source chain tokens that are in escrow.
// Amount is stored in state if and only if it is not equal to zero. The function will panic
// if the amount is negative.
func (k Keeper) SetTotalEscrowForDenom(ctx sdk.Context, coin sdk.Coin) {
	if coin.Amount.IsNegative() {
		panic(fmt.Errorf("amount cannot be negative: %s", coin.Amount))
	}

	store := ctx.KVStore(k.storeKey)
	key := types.TotalEscrowForDenomKey(coin.Denom)

	if coin.Amount.IsZero() {
		store.Delete(key) // delete the key since Cosmos SDK x/bank module will prune any non-zero balances
		return
	}

	bz := k.cdc.MustMarshal(&sdk.IntProto{Int: coin.Amount})
	store.Set(key, bz)
}

// GetAllTotalEscrowed returns the escrow information for all the denominations.
func (k Keeper) GetAllTotalEscrowed(ctx sdk.Context) sdk.Coins {
	var escrows sdk.Coins
	k.IterateTokensInEscrow(ctx, []byte(types.KeyTotalEscrowPrefix), func(denomEscrow sdk.Coin) bool {
		escrows = escrows.Add(denomEscrow)
		return false
	})

	return escrows
}

// IterateTokensInEscrow iterates over the denomination escrows in the store
// and performs a callback function. Denominations for which an invalid value
// (i.e. not integer) is stored, will be skipped.
func (k Keeper) IterateTokensInEscrow(ctx sdk.Context, storeprefix []byte, cb func(denomEscrow sdk.Coin) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, storeprefix)

	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		denom := strings.TrimPrefix(string(iterator.Key()), fmt.Sprintf("%s/", types.KeyTotalEscrowPrefix))
		if strings.TrimSpace(denom) == "" {
			continue // denom is empty
		}

		amount := sdk.IntProto{}
		if err := k.cdc.Unmarshal(iterator.Value(), &amount); err != nil {
			continue // total escrow amount cannot be unmarshalled to integer
		}

		denomEscrow := sdk.NewCoin(denom, amount.Int)
		if cb(denomEscrow) {
			break
		}
	}
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

// setForwardedPacket sets the forwarded packet in the store.
func (k Keeper) setForwardedPacket(ctx sdk.Context, portID, channelID string, sequence uint64, packet channeltypes.Packet) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&packet)
	store.Set(types.PacketForwardKey(portID, channelID, sequence), bz)
}

// getForwardedPacket gets the forwarded packet from the store.
func (k Keeper) getForwardedPacket(ctx sdk.Context, portID, channelID string, sequence uint64) (channeltypes.Packet, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PacketForwardKey(portID, channelID, sequence))
	if bz == nil {
		return channeltypes.Packet{}, false
	}

	var storedPacket channeltypes.Packet
	k.cdc.MustUnmarshal(bz, &storedPacket)

	return storedPacket, true
}

// deleteForwardedPacket deletes the forwarded packet from the store.
func (k Keeper) deleteForwardedPacket(ctx sdk.Context, portID, channelID string, sequence uint64) {
	store := ctx.KVStore(k.storeKey)
	packetKey := types.PacketForwardKey(portID, channelID, sequence)

	store.Delete(packetKey)
}

// IsBlockedAddr checks if the given address is allowed to send or receive tokens.
// The module account is always allowed to send and receive tokens.
func (k Keeper) isBlockedAddr(addr sdk.AccAddress) bool {
	moduleAddr := k.authKeeper.GetModuleAddress(types.ModuleName)
	if addr.Equals(moduleAddr) {
		return false
	}

	return k.bankKeeper.BlockedAddr(addr)
}
