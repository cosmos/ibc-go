package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
)

// RegisterInterchainAccount attempts to create a new account using the provided address and stores it in state keyed by the provided port identifier
// If an account for the provided address already exists this function returns early (no-op)
func (k Keeper) RegisterInterchainAccount(ctx sdk.Context, accAddr sdk.AccAddress, controllerPortID string) {
	if acc := k.accountKeeper.GetAccount(ctx, accAddr); acc != nil {
		return
	}

	interchainAccount := types.NewInterchainAccount(
		authtypes.NewBaseAccountWithAddress(accAddr),
		controllerPortID,
	)

	k.accountKeeper.NewAccount(ctx, interchainAccount)
	k.accountKeeper.SetAccount(ctx, interchainAccount)

	k.icaKeeper.SetInterchainAccountAddress(ctx, types.ModuleName, controllerPortID, interchainAccount.Address)
}
