package keeper

import (
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// Keeper defines the IBC non fungible transfer keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	ics4Wrapper   types.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	nftKeeper     types.NFTKeeper
	authkeeper    types.AccountKeeper
	// bankKeeper    types.BankKeeper
	scopedKeeper capabilitykeeper.ScopedKeeper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+host.ModuleName+"-"+types.ModuleName)
}

// GetPort returns the portID for the transfer module.
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(types.PortKey))
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return k.scopedKeeper.AuthenticateCapability(ctx, cap, name)
}

// ClaimCapability allows the nft-transfer module that can claim a capability that IBC module
// passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// MustUnmarshalClassTrace attempts to decode and return an ClassTrace object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustUnmarshalClassTrace(bz []byte) types.ClassTrace {
	var classTrace types.ClassTrace
	k.cdc.MustUnmarshal(bz, &classTrace)
	return classTrace
}

// MustMarshalClassTrace attempts to decode and return an ClassTrace object from
// raw encoded bytes. It panics on error.
func (k Keeper) MustMarshalClassTrace(classTrace types.ClassTrace) []byte {
	return k.cdc.MustMarshal(&classTrace)
}

// UnmarshalClassTrace attempts to decode and return an ClassTrace object from
// raw encoded bytes.
func (k Keeper) UnmarshalClassTrace(bz []byte) (types.ClassTrace, error) {
	var classTrace types.ClassTrace
	if err := k.cdc.Unmarshal(bz, &classTrace); err != nil {
		return types.ClassTrace{}, err
	}

	return classTrace, nil
}

// SetEscrowAddress attempts to save a account to auth module
func (k Keeper) SetEscrowAddress(ctx sdk.Context, portID, channelID string) {
	// create the escrow address for the tokens
	escrowAddress := types.GetEscrowAddress(portID, channelID)
	if !k.authkeeper.HasAccount(ctx, escrowAddress) {
		acc := k.authkeeper.NewAccountWithAddress(ctx, escrowAddress)
		k.authkeeper.SetAccount(ctx, acc)
	}
}
