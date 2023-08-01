package keeper

import (
	"fmt"
	"strings"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// Keeper defines the IBC fungible transfer keeper
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	authKeeper    types.AccountKeeper
	bankKeeper    types.BankKeeper
	scopedKeeper  exported.ScopedKeeper
}

// NewKeeper creates a new IBC transfer Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key storetypes.StoreKey, paramSpace paramtypes.Subspace,
	ics4Wrapper porttypes.ICS4Wrapper, channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	authKeeper types.AccountKeeper, bankKeeper types.BankKeeper, scopedKeeper exported.ScopedKeeper,
) Keeper {
	// ensure ibc transfer module account is set
	if addr := authKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the IBC transfer module account has not been set")
	}

	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      key,
		paramSpace:    paramSpace,
		ics4Wrapper:   ics4Wrapper,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		authKeeper:    authKeeper,
		bankKeeper:    bankKeeper,
		scopedKeeper:  scopedKeeper,
	}
}

// WithICS4Wrapper sets the ICS4Wrapper. This function may be used after
// the keepers creation to set the middleware which is above this module
// in the IBC application stack.
func (k *Keeper) WithICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	k.ics4Wrapper = wrapper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"-"+types.ModuleName)
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

// GetDenomTrace retreives the full identifiers trace and base denomination from the store.
func (k Keeper) GetDenomTrace(ctx sdk.Context, denomTraceHash tmbytes.HexBytes) (types.DenomTrace, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomTraceKey)
	bz := store.Get(denomTraceHash)
	if len(bz) == 0 {
		return types.DenomTrace{}, false
	}

	denomTrace := k.MustUnmarshalDenomTrace(bz)
	return denomTrace, true
}

// HasDenomTrace checks if a the key with the given denomination trace hash exists on the store.
func (k Keeper) HasDenomTrace(ctx sdk.Context, denomTraceHash tmbytes.HexBytes) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomTraceKey)
	return store.Has(denomTraceHash)
}

// SetDenomTrace sets a new {trace hash -> denom trace} pair to the store.
func (k Keeper) SetDenomTrace(ctx sdk.Context, denomTrace types.DenomTrace) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomTraceKey)
	bz := k.MustMarshalDenomTrace(denomTrace)
	store.Set(denomTrace.Hash(), bz)
}

// GetAllDenomTraces returns the trace information for all the denominations.
func (k Keeper) GetAllDenomTraces(ctx sdk.Context) types.Traces {
	traces := types.Traces{}
	k.IterateDenomTraces(ctx, func(denomTrace types.DenomTrace) bool {
		traces = append(traces, denomTrace)
		return false
	})

	return traces.Sort()
}

// IterateDenomTraces iterates over the denomination traces in the store
// and performs a callback function.
func (k Keeper) IterateDenomTraces(ctx sdk.Context, cb func(denomTrace types.DenomTrace) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.DenomTraceKey)

	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		denomTrace := k.MustUnmarshalDenomTrace(iterator.Value())
		if cb(denomTrace) {
			break
		}
	}
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
		return sdk.NewCoin(denom, sdk.ZeroInt())
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
		panic(fmt.Sprintf("amount cannot be negative: %s", coin.Amount))
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
func (k Keeper) IterateTokensInEscrow(ctx sdk.Context, prefix []byte, cb func(denomEscrow sdk.Coin) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, prefix)

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
