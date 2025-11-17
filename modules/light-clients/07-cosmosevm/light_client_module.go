package cosmosevm

import (
	// "fmt"
	//
	// errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	// sdk "github.com/cosmos/cosmos-sdk/types"
	//
	// clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	// ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	// "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	cdc           codec.BinaryCodec
	clientKeeper ClientKeeper
}
