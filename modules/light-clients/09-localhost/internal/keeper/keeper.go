package keeper

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

// Keeper defines the localhost light client module keeper
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	authority string,
) Keeper {
	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	return Keeper{
		cdc:       cdc,
		storeKey:  key,
		authority: authority,
	}
}

// ClientStore returns isolated prefix store for each client so they can read/write in separate
// namespace without being able to read/write other client's data
func (k Keeper) ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore {
	clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
	return prefix.NewStore(ctx.KVStore(k.storeKey), clientPrefix)
}

func (k Keeper) Codec() codec.BinaryCodec {
	return k.cdc
}
