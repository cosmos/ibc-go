package keeper

import (
	"errors"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
)

// Keeper defines the tendermint light client module keeper
type Keeper struct {
	cdc codec.BinaryCodec

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewKeeper(cdc codec.BinaryCodec, authority string) Keeper {
	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	return Keeper{
		cdc:       cdc,
		authority: authority,
	}
}

func (k Keeper) Codec() codec.BinaryCodec {
	return k.cdc
}
