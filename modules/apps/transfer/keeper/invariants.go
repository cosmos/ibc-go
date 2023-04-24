package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

// RegisterInvariants registers all transfer invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k *Keeper) {
	ir.RegisterRoute(types.ModuleName, "total-escrow-per-denom",
		TotalEscrowPerDenomInvariants(k))
}

// AllInvariants runs all invariants of the transfer module.
func AllInvariants(k *Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		return TotalEscrowPerDenomInvariants(k)(ctx)
	}
}

// TotalEscrowPerDenomInvariants checks that the total amount escrowed for
// each denom is not smaller than the amount stored in the state entry.
func TotalEscrowPerDenomInvariants(k *Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg                     string
			broken                  bool
			totalEscrowedInAccounts sdk.Coins
		)

		totalEscrowedInState := k.GetAllTotalEscrowed(ctx)

		portID := k.GetPort(ctx)
		transferChannels := k.channelKeeper.GetAllChannelsWithPortPrefix(ctx, portID)
		for _, channel := range transferChannels {
			escrowAddress := types.GetEscrowAddress(portID, channel.ChannelId)
			escrowBalances := k.bankKeeper.GetAllBalances(ctx, escrowAddress)

			totalEscrowedInAccounts = totalEscrowedInAccounts.Add(escrowBalances...)
		}

		for _, expectedEscrow := range totalEscrowedInState {
			if found, actualEscrow := totalEscrowedInAccounts.Find(expectedEscrow.GetDenom()); found {
				if expectedEscrow.Amount.GT(actualEscrow.Amount) {
					broken = true
					msg += fmt.Sprintf("\tdenom: %s, actual escrow (%s) is < expected escrow (%s)\n", expectedEscrow.GetDenom(), actualEscrow.Amount, expectedEscrow.Amount)
				}
			}
		}

		if broken {
			// the total amount for each denom in escrow should be >= the amount stored in state for each denom
			return sdk.FormatInvariant(
				types.ModuleName,
				"total escrow per denom invariance",
				fmt.Sprintf("found denom(s) with total escrow amount lower than expected:\n%s", msg)), broken
		}

		return "", broken
	}
}
