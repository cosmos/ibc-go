package celestia

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// VerifyClientMessage implements exported.ClientState.
func (*ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) error {
	panic("unimplemented")
}

// UpdateState implements exported.ClientState.
func (*ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	panic("unimplemented")
}

// CheckForMisbehaviour implements exported.ClientState.
func (*ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) bool {
	panic("unimplemented")
}

// UpdateStateOnMisbehaviour implements exported.ClientState.
func (*ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) {
	panic("unimplemented")
}
