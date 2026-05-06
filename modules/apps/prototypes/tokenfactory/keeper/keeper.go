package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/tokenfactory/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper

	Schema      collections.Schema
	ParamsStore collections.Item[types.Params]

	DenomAuthorityMetadataStore collections.Map[string, types.DenomAuthorityMetadata]
	CreatorPrefixStore          collections.Map[collections.Pair[string, string], bool]
}

// NewKeeper creates a new tokenfactory Keeper instance
func NewKeeper(cdc codec.BinaryCodec, storeService store.KVStoreService, accountKeeper types.AccountKeeper, bankKeeper types.BankKeeper) Keeper {
	if accountKeeper == nil {
		panic("accountKeeper cannot be nil")
	}
	if bankKeeper == nil {
		panic("bankKeeper cannot be nil")
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:           cdc,
		storeService:  storeService,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,

		ParamsStore:                 collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		DenomAuthorityMetadataStore: collections.NewMap(sb, types.DenomAuthorityMetadataPrefix, "denom_authority_metadata", collections.StringKey, codec.CollValue[types.DenomAuthorityMetadata](cdc)),
		CreatorPrefixStore:          collections.NewMap(sb, types.CreatorPrefixKey, "creator_prefix_store", collections.PairKeyCodec(collections.StringKey, collections.StringKey), collections.BoolValue),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the x/tokenfactory module's authority.
func (Keeper) GetAuthority() string {
	return types.ModuleName
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
