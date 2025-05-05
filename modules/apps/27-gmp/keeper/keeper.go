package keeper

import (
	"errors"
	"strings"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
)

// Keeper defines the IBC fungible transfer keeper
type Keeper struct {
	cdc codec.BinaryCodec

	ics4Wrapper porttypes.ICS4Wrapper
	msgRouter   types.MessageRouter

	accountKeeper types.AccountKeeper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string

	// state management
	Schema collections.Schema
	// Accounts is a map of  (ClientID, Sender, Salt) to ICS27Account
	Accounts collections.Map[collections.Triple[string, string, []byte], types.ICS27Account]
}

// NewKeeper creates a new Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, storeService storetypes.KVStoreService,
	accountKeeper types.AccountKeeper, msgRouter types.MessageRouter,
	authority string,
) Keeper {
	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:       cdc,
		authority: authority,
		Accounts:  collections.NewMap(sb, types.AccountsKey, "accounts", collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.BytesKey), codec.CollValue[types.ICS27Account](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}
