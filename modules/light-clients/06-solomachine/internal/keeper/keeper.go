package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// Keeper defines the 06-solomachine Keeper.
// TODO(damian): delete the keeper and put the codec on the LightClientModule?
type Keeper struct {
	cdc codec.BinaryCodec
}

// NewKeeper creates and returns a new 06-solomachine keeper.
func NewKeeper(cdc codec.BinaryCodec) Keeper {
	return Keeper{
		cdc: cdc,
	}
}

// Codec returns the keeper codec.
func (k Keeper) Codec() codec.BinaryCodec {
	return k.cdc
}
