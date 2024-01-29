package tendermint

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type LightClientModule struct {
	cdc codec.BinaryCodec
	prefixStoreKey storetypes.StoreKey
}

// Initialize checks that the initial consensus state is an 07-tendermint consensus state and
// sets the client state, consensus state and associated metadata in the provided client store.
func (lcm LightClientModule) Initialize(ctx sdk.Context, clientID, clientState conensusState []byte) error {
	// validate args
	// - client id
	// client/consensus state if len == 0

	// unpack client/consensus states

	// get store key
	lcm.getStoreKey(clientID)

	return clientState.Initialize(ctx, lcm.cdc, storeKey, consensusState)
}

