package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// GetSDKAmount returns the amount as a cosmos-sdk math.Int.
func (m *QueryTotalEscrowForDenomResponse) GetSDKAmount() (math.Int, error) {
	amount, ok := math.NewIntFromString(m.Amount)
	if !ok {
		return math.ZeroInt(), fmt.Errorf(`unable to convert string "%s" to int`, m.Amount)
	}
	return math.NewInt(amount.Int64()), nil
}
