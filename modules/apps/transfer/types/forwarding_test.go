package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

var validHop = types.Hop{
	PortId:    types.PortID,
	ChannelId: ibctesting.FirstChannelID,
}

func TestForwarding_Validate(t *testing.T) {
	tests := []struct {
		name       string
		forwarding *types.Forwarding
		expError   error
	}{
		{
			"valid forwarding with no hops",
			types.NewForwarding(""),
			nil,
		},
		{
			"valid forwarding with hops",
			types.NewForwarding("", validHop),
			nil,
		},
		{
			"valid forwarding with memo",
			types.NewForwarding(testMemo1, validHop, validHop),
			nil,
		},
		{
			"valid forwarding with max hops",
			types.NewForwarding("", generateHops(types.MaximumNumberOfForwardingHops)...),
			nil,
		},
		{
			"valid forwarding with max memo length",
			types.NewForwarding(ibctesting.GenerateString(types.MaximumMemoLength), validHop),
			nil,
		},
		{
			"invalid forwarding with too many hops",
			types.NewForwarding("", generateHops(types.MaximumNumberOfForwardingHops+1)...),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too long memo",
			types.NewForwarding(ibctesting.GenerateString(types.MaximumMemoLength+1), validHop),
			types.ErrInvalidMemo,
		},
		{
			"invalid forwarding with empty hops and specified memo",
			types.NewForwarding("memo"),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too short hop port ID",
			types.NewForwarding(
				"",
				types.Hop{
					PortId:    invalidShortPort,
					ChannelId: ibctesting.FirstChannelID,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding with too long hop port ID",
			types.NewForwarding(
				"",
				types.Hop{
					PortId:    invalidLongPort,
					ChannelId: ibctesting.FirstChannelID,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding with non-alpha hop port ID",
			types.NewForwarding(
				"",
				types.Hop{
					PortId:    invalidPort,
					ChannelId: ibctesting.FirstChannelID,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding with too long hop channel ID",
			types.NewForwarding(
				"",
				types.Hop{
					PortId:    types.PortID,
					ChannelId: invalidLongChannel,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding with too short hop channel ID",
			types.NewForwarding(
				"",
				types.Hop{
					PortId:    types.PortID,
					ChannelId: invalidShortChannel,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding with non-alpha hop channel ID",
			types.NewForwarding(
				"",
				types.Hop{
					PortId:    types.PortID,
					ChannelId: invalidChannel,
				},
			),
			host.ErrInvalidID,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc

			err := tc.forwarding.Validate()

			expPass := tc.expError == nil
			if expPass {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}

func generateHops(n int) []types.Hop {
	hops := make([]types.Hop, n)
	for i := 0; i < n; i++ {
		hops[i] = types.Hop{
			PortId:    types.PortID,
			ChannelId: ibctesting.FirstChannelID,
		}
	}
	return hops
}
