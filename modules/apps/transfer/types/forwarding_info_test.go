package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

var validHop = &types.Hop{
	PortId:    types.PortID,
	ChannelId: ibctesting.FirstChannelID,
}

func TestForwardingInfo_Validate(t *testing.T) {
	tests := []struct {
		name           string
		forwardingInfo *types.ForwardingInfo
		expError       error
	}{
		{
			"valid forwarding info with no hops",
			types.NewForwardingInfo(""),
			nil,
		},
		{
			"valid forwarding info with hops",
			types.NewForwardingInfo("", validHop),
			nil,
		},
		{
			"valid forwarding info with memo",
			types.NewForwardingInfo(testMemo1, validHop, validHop),
			nil,
		},
		{
			"valid forwarding info with max hops",
			types.NewForwardingInfo("", generateHops(types.MaximumNumberOfForwardingHops)...),
			nil,
		},
		{
			"valid forwarding info with max memo length",
			types.NewForwardingInfo(ibctesting.GenerateString(types.MaximumMemoLength), validHop),
			nil,
		},
		{
			"invalid forwarding info with too many hops",
			types.NewForwardingInfo("", generateHops(types.MaximumNumberOfForwardingHops+1)...),
			types.ErrInvalidForwardingInfo,
		},
		{
			"invalid forwarding info with too long memo",
			types.NewForwardingInfo(ibctesting.GenerateString(types.MaximumMemoLength+1), validHop),
			types.ErrInvalidMemo,
		},
		{
			"invalid forwarding info with too short hop port ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    invalidShortPort,
					ChannelId: ibctesting.FirstChannelID,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding info with too long hop port ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    invalidLongPort,
					ChannelId: ibctesting.FirstChannelID,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding info with non-alpha hop port ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    invalidPort,
					ChannelId: ibctesting.FirstChannelID,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding info with too long hop channel ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    types.PortID,
					ChannelId: invalidLongChannel,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding info with too short hop channel ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    types.PortID,
					ChannelId: invalidShortChannel,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid forwarding info with non-alpha hop channel ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
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

			err := tc.forwardingInfo.ValidateBasic()

			expPass := tc.expError == nil
			if expPass {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}

func generateHops(n int) []*types.Hop {
	hops := make([]*types.Hop, n)
	for i := 0; i < n; i++ {
		hops[i] = &types.Hop{
			PortId:    types.PortID,
			ChannelId: ibctesting.FirstChannelID,
		}
	}
	return hops
}
