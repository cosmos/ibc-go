package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
)

// createInterchainAccount creates a new interchain account. An address is generated using the host connectionID, the controller portID,
// and block dependent information. An error is returned if an account already exists for the generated account.
// An interchain account type is set in the account keeper and the interchain account address mapping is updated.
func (k Keeper) createInterchainAccount(ctx context.Context, connectionID, controllerPortID string) (sdk.AccAddress, error) {
	accAddress := icatypes.GenerateAddress(k.HeaderService.HeaderInfo(ctx), connectionID, controllerPortID)

	if acc := k.authKeeper.GetAccount(ctx, accAddress); acc != nil {
		return nil, errorsmod.Wrapf(icatypes.ErrAccountAlreadyExist, "existing account for newly generated interchain account address %s", accAddress)
	}

	interchainAccount := icatypes.NewInterchainAccount(
		authtypes.NewBaseAccountWithAddress(accAddress),
		controllerPortID,
	)

	k.authKeeper.NewAccount(ctx, interchainAccount)
	k.authKeeper.SetAccount(ctx, interchainAccount)

	k.SetInterchainAccountAddress(ctx, connectionID, controllerPortID, interchainAccount.Address)

	return accAddress, nil
}
