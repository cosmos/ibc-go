package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
)

var _ types.IbcTransferHooks = MultiIbcHooks{}

// MultiIbcHooks combine multiple evm hooks, all hook functions are run in array sequence
type MultiIbcHooks []types.IbcTransferHooks

// NewMultiIbcHooks combine multiple evm hooks
func NewMultiIbcHooks(hooks ...types.IbcTransferHooks) MultiIbcHooks {
	return hooks
}

// AfterRecvPacket delegate the call to underlying hooks
func (mh MultiIbcHooks) AfterRecvPacket(ctx sdk.Context) error {
	for i := range mh {
		if err := mh[i].AfterRecvPacket(ctx); err != nil {
			return sdkerrors.Wrapf(err, "EVM hook %T failed", mh[i])
		}
	}
	return nil
}
