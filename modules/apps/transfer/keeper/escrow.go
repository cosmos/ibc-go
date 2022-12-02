package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
)

// GetEscrowAccount returns the escrow account (ModuleAccount) for the corresponding source and port.
// If the account exists but is not a ModuleAccount, the existing account is migrated to this account type. 
// If the escrow account does not exist, this function creates a new module account to escrow the transferred coins
func (k Keeper) GetEscrowAccount(ctx sdk.Context, sourcePort, sourceChannel string) authtypes.ModuleAccountI {
	// name of the escrow Module Account is derived from the source port and channel ID
	accountName := fmt.Sprintf("%s/%s", sourcePort, sourceChannel)

	// create escrow address for the tokens as defined by ADR-028
	// https://docs.cosmos.network/main/architecture/adr-028-public-key-addresses
	address := types.GetEscrowAddress(sourcePort, sourceChannel)

    // check if account already exists
    if existingAcc := k.authKeeper.GetAccount(ctx, address); existingAcc != nil {
        existingAcc, isModuleAccount := existingAcc.(authtypes.ModuleAccountI)
        // use existent account if it's ModuleAccount. Otherwise create a new ModuleAccount
        if isModuleAccount {
            return existingAcc
        }
    }

	baseAcc := authtypes.NewBaseAccountWithAddress(address)
	// no special permissions defined for the module account
	moduleAcc := authtypes.NewModuleAccount(baseAcc, accountName)
	k.authKeeper.SetModuleAccount(ctx, moduleAcc)

	return moduleAcc
}
