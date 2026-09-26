// SPDX-License-Identifier: Apache-2.0

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuthenticateTx is exported for testing purposes.
func (k *Keeper) AuthenticateTx(ctx sdk.Context, account sdk.AccountI, msgs []sdk.Msg) error {
	return k.authenticateTx(ctx, account, msgs)
}
