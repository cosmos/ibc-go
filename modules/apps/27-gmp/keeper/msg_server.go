package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

var _ types.MsgServer = (*Keeper)(nil)

// SendCall defines the handler for the MsgSendCall message.
func (k Keeper) SendCall(ctx context.Context, msg *types.MsgSendCall) (*types.MsgSendCallResponse, error) {
	// TODO: Add logic
	panic("not implemented")
}
