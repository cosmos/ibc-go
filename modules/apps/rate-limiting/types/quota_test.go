package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

func TestCheckExceedsQuota(t *testing.T) {
	totalValue := sdkmath.NewInt(100)
	amountUnderThreshold := sdkmath.NewInt(5)
	amountOverThreshold := sdkmath.NewInt(15)
	quota := types.Quota{
		MaxPercentRecv: sdkmath.NewInt(10),
		MaxPercentSend: sdkmath.NewInt(10),
		DurationHours:  uint64(1),
	}

	tests := []struct {
		name       string
		direction  types.PacketDirection
		amount     sdkmath.Int
		totalValue sdkmath.Int
		exceeded   bool
	}{
		{
			name:       "inflow exceeded threshold",
			direction:  types.PACKET_RECV,
			amount:     amountOverThreshold,
			totalValue: totalValue,
			exceeded:   true,
		},
		{
			name:       "inflow did not exceed threshold",
			direction:  types.PACKET_RECV,
			amount:     amountUnderThreshold,
			totalValue: totalValue,
			exceeded:   false,
		},
		{
			name:       "outflow exceeded threshold",
			direction:  types.PACKET_SEND,
			amount:     amountOverThreshold,
			totalValue: totalValue,
			exceeded:   true,
		},
		{
			name:       "outflow did not exceed threshold",
			direction:  types.PACKET_SEND,
			amount:     amountUnderThreshold,
			totalValue: totalValue,
			exceeded:   false,
		},
		{
			name:       "zero channel value send",
			direction:  types.PACKET_SEND,
			amount:     amountOverThreshold,
			totalValue: sdkmath.ZeroInt(),
			exceeded:   false,
		},
		{
			name:       "zero channel value recv",
			direction:  types.PACKET_RECV,
			amount:     amountOverThreshold,
			totalValue: sdkmath.ZeroInt(),
			exceeded:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := quota.CheckExceedsQuota(test.direction, test.amount, test.totalValue)
			require.Equal(t, res, test.exceeded, "test: %s", test.name)
		})
	}
}
