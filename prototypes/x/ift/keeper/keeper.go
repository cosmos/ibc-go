package keeper

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/sandbox-ledger/x/ift/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService
	addressCodec address.Codec
	authority    string

	accountKeeper      types.AccountKeeper
	tokenFactoryKeeper types.TokenFactoryKeeper
	gmpKeeper          types.GMPKeeper
	msgRouter          types.MessageRouter
	ibcClientKeeper    types.IBCClientKeeper
	ibcClientV2Keeper  types.IBCClientV2Keeper

	Schema               collections.Schema
	ParamsStore          collections.Item[types.Params]
	IFTBridgeStore       collections.Map[collections.Pair[string, string], types.IFTBridge]
	PendingTransferStore collections.Map[collections.Pair[string, uint64], types.PendingTransfer] // keyed by (clientID, sequence)
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	addressCodec address.Codec,
	authority string,
	accountKeeper types.AccountKeeper,
	tokenFactoryKeeper types.TokenFactoryKeeper,
	gmpKeeper types.GMPKeeper,
	msgRouter types.MessageRouter,
	ibcClientKeeper types.IBCClientKeeper,
	ibcClientV2Keeper types.IBCClientV2Keeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:                cdc,
		storeService:       storeService,
		addressCodec:       addressCodec,
		authority:          authority,
		accountKeeper:      accountKeeper,
		tokenFactoryKeeper: tokenFactoryKeeper,
		gmpKeeper:          gmpKeeper,
		msgRouter:          msgRouter,
		ibcClientKeeper:    ibcClientKeeper,
		ibcClientV2Keeper:  ibcClientV2Keeper,
		ParamsStore:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		IFTBridgeStore:     collections.NewMap(sb, types.IFTBridgePrefix, "ift_bridges", collections.PairKeyCodec(collections.StringKey, collections.StringKey), codec.CollValue[types.IFTBridge](cdc)),
		PendingTransferStore: collections.NewMap(sb, types.PendingTransferKey, "pending_transfers",
			collections.PairKeyCodec(collections.StringKey, collections.Uint64Key),
			codec.CollValue[types.PendingTransfer](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

func (Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) GetModuleAddress() sdk.AccAddress {
	return k.accountKeeper.GetModuleAddress(types.ModuleName)
}

// SetPendingTransfer saves the pending transfer keyed by (clientID, sequence)
func (k Keeper) SetPendingTransfer(ctx context.Context, clientID string, sequence uint64, pending types.PendingTransfer) error {
	return k.PendingTransferStore.Set(ctx, collections.Join(clientID, sequence), pending)
}

// RemovePendingTransfer removes the pending transfer by (clientID, sequence)
func (k Keeper) RemovePendingTransfer(ctx context.Context, clientID string, sequence uint64) error {
	return k.PendingTransferStore.Remove(ctx, collections.Join(clientID, sequence))
}

// HasPendingTransfersForBridge checks if there are any pending transfers for a specific bridge.
// Uses prefix range to efficiently walk only entries for the given clientID.
func (k Keeper) HasPendingTransfersForBridge(ctx context.Context, denom, clientID string) (bool, error) {
	rng := collections.NewPrefixedPairRange[string, uint64](clientID)
	var found bool
	err := k.PendingTransferStore.Walk(ctx, rng, func(_ collections.Pair[string, uint64], pending types.PendingTransfer) (bool, error) {
		if pending.Denom == denom {
			found = true
			return true, nil // stop iteration
		}
		return false, nil
	})
	return found, err
}

// GetPendingTransferByClientSequence looks up a pending transfer by (clientID, sequence).
func (k Keeper) GetPendingTransferByClientSequence(ctx context.Context, clientID string, sequence uint64) (types.PendingTransfer, bool, error) {
	pending, err := k.PendingTransferStore.Get(ctx, collections.Join(clientID, sequence))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.PendingTransfer{}, false, nil
		}
		return types.PendingTransfer{}, false, err
	}
	return pending, true, nil
}
