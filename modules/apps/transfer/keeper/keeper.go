package keeper

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// Keeper defines the IBC fungible transfer keeper
type Keeper struct {
	storeService   corestore.KVStoreService
	cdc            codec.BinaryCodec
	legacySubspace types.ParamSubspace
	addressCodec   address.Codec

	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	msgRouter     types.MessageRouter
	AuthKeeper    types.AccountKeeper
	BankKeeper    types.BankKeeper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper creates a new IBC transfer Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService corestore.KVStoreService,
	legacySubspace types.ParamSubspace,
	ics4Wrapper porttypes.ICS4Wrapper,
	channelKeeper types.ChannelKeeper,
	msgRouter types.MessageRouter,
	authKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	// ensure ibc transfer module account is set
	if addr := authKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(errors.New("the IBC transfer module account has not been set"))
	}

	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	addressCodec := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	return Keeper{
		cdc:            cdc,
		storeService:   storeService,
		legacySubspace: legacySubspace,
		addressCodec:   addressCodec,
		ics4Wrapper:    ics4Wrapper,
		channelKeeper:  channelKeeper,
		msgRouter:      msgRouter,
		AuthKeeper:     authKeeper,
		BankKeeper:     bankKeeper,
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

// SetAddressCodec sets the address codec used by the keeper.
func (k *Keeper) SetAddressCodec(addressCodec address.Codec) {
	k.addressCodec = addressCodec
}

// GetAddressCodec returns the address codec used by the keeper.
func (k *Keeper) GetAddressCodec() address.Codec {
	return k.addressCodec
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"-"+types.ModuleName)
}

// GetPort returns the portID for the transfer module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.PortKey)
	if err != nil {
		panic(err)
	}
	return string(bz)
}

// SetPort sets the portID for the transfer module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.PortKey, []byte(portID)); err != nil {
		panic(err)
	}
}

// GetParams returns the current transfer module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get([]byte(types.ParamsKey))
	if err != nil {
		panic(err)
	}
	if bz == nil { // only panic on unset params and not on empty params
		panic(errors.New("transfer params are not set in store"))
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the transfer module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&params)
	if err := store.Set([]byte(types.ParamsKey), bz); err != nil {
		panic(err)
	}
}

// GetDenom retrieves the denom from store given the hash of the denom.
func (k Keeper) GetDenom(ctx sdk.Context, denomHash cmtbytes.HexBytes) (types.Denom, bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DenomKey)
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
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DenomKey)
	return store.Has(denomHash)
}

// SetDenom sets a new {denom hash -> denom } pair to the store.
// This allows for reverse lookup of the denom given the hash.
func (k Keeper) SetDenom(ctx sdk.Context, denom types.Denom) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.DenomKey)
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
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, types.DenomKey)

	defer sdk.LogDeferred(k.Logger(ctx), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		var denom types.Denom
		k.cdc.MustUnmarshal(iterator.Value(), &denom)

		if cb(denom) {
			break
		}
	}
}

// SetDenomMetadata sets an IBC token's denomination metadata
func (k Keeper) SetDenomMetadata(ctx sdk.Context, denom types.Denom) {
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

	k.BankKeeper.SetDenomMetaData(ctx, metadata)
}

// GetTotalEscrowForDenom gets the total amount of source chain tokens that
// are in escrow, keyed by the denomination.
//
// NOTE: if there is no value stored in state for the provided denom then a new Coin is returned for the denom with an initial value of zero.
// This accommodates callers to simply call `Add()` on the returned Coin as an empty Coin literal (e.g. sdk.Coin{}) will trigger a panic due to the absence of a denom.
func (k Keeper) GetTotalEscrowForDenom(ctx sdk.Context, denom string) sdk.Coin {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.TotalEscrowForDenomKey(denom))
	if err != nil {
		panic(err)
	}
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

	store := k.storeService.OpenKVStore(ctx)
	key := types.TotalEscrowForDenomKey(coin.Denom)

	if coin.Amount.IsZero() {
		if err := store.Delete(key); err != nil { // delete the key since Cosmos SDK x/bank module will prune any non-zero balances
			panic(err)
		}
		return
	}

	bz := k.cdc.MustMarshal(&sdk.IntProto{Int: coin.Amount})
	if err := store.Set(key, bz); err != nil {
		panic(err)
	}
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
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, storeprefix)

	defer sdk.LogDeferred(k.Logger(ctx), func() error { return iterator.Close() })
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

// IsBlockedAddr checks if the given address is allowed to send or receive tokens.
// The module account is always allowed to send and receive tokens.
func (k Keeper) IsBlockedAddr(addr sdk.AccAddress) bool {
	moduleAddr := k.AuthKeeper.GetModuleAddress(types.ModuleName)
	if addr.Equals(moduleAddr) {
		return false
	}

	return k.BankKeeper.BlockedAddr(addr)
}
