package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
)

// GetEscrowAccount creates a module account to escrow the transferred coins
func (k Keeper) GetEscrowAccount(ctx sdk.Context, sourcePort, sourceChannel string) authtypes.ModuleAccountI {
	// name of the escrow Module Account is derived from the source port and channel ID
	escrowAccountName := fmt.Sprintf("%s/%s", sourcePort, sourceChannel)

	// create escrow address for the tokens as defined by ADR-028
	// https://docs.cosmos.network/main/architecture/adr-028-public-key-addresses
	escrowAddress := types.GetEscrowAddress(sourcePort, sourceChannel)

    // check if account already exists
    if existingAcc := k.authKeeper.GetAccount(ctx, escrowAddress); existingAcc != nil {
        existingAcc, isModuleAccount := existingAcc.(authtypes.ModuleAccountI)
        // use existent account if it's ModuleAccount. Otherwise create a new ModuleAccount
        if isModuleAccount {
            return existingAcc
        }
    }

	baseAcc := authtypes.NewBaseAccountWithAddress(escrowAddress)
	// no special permissions defined for the module account
	escrowModuleAcc := authtypes.NewModuleAccount(baseAcc, escrowAccountName)
	k.authKeeper.SetModuleAccount(ctx, escrowModuleAcc)

	return escrowModuleAcc
}
