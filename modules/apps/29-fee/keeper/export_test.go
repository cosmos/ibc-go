package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
)

// LegacyTotal is a wrapper for the legacyTotal function for testing.
func LegacyTotal(f types.Fee) sdk.Coins {
	return legacyTotal(f)
}
