package keeper

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

<<<<<<< HEAD
import porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"

// GetICS4Wrapper is a getter for the keeper's ICS4Wrapper.
func (k *Keeper) GetICS4Wrapper() porttypes.ICS4Wrapper {
	return k.ics4Wrapper
=======
	"github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
)

// LegacyTotal is a wrapper for the legacyTotal function for testing.
func LegacyTotal(f types.Fee) sdk.Coins {
	return legacyTotal(f)
>>>>>>> ee4549bb (fix: fixed callbacks middleware wiring (#5950))
}
