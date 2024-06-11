package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

var validHop = &types.Hop{
	PortId:    "transfer",
	ChannelId: "channel-0",
}

func TestForwardingInfo_Validate(t *testing.T) {
	tests := []struct {
		name           string
		forwardingInfo *types.ForwardingInfo
		expError       error
	}{
		{
			"valid msg no hops",
			types.NewForwardingInfo(""),
			nil,
		},
		{
			"valid msg with hops",
			types.NewForwardingInfo("", validHop),
			nil,
		},
		{
			"valid msg with memo",
			types.NewForwardingInfo(testMemo1, validHop, validHop),
			nil,
		},
		{
			"valid msg with max hops",
			types.NewForwardingInfo("", generateHops(types.MaximumNumberOfForwardingHops)...),
			nil,
		},
		{
			"valid msg with max memo length",
			types.NewForwardingInfo(ibctesting.GenerateString(types.MaximumMemoLength), validHop),
			nil,
		},
		{
			"invalid msg with too many hops",
			types.NewForwardingInfo("", generateHops(types.MaximumNumberOfForwardingHops+1)...),
			types.ErrInvalidForwardingInfo,
		},
		{
			"invalid msg with too long memo",
			types.NewForwardingInfo(ibctesting.GenerateString(types.MaximumMemoLength+1), validHop),
			types.ErrInvalidMemo,
		},
		{
			"invalid msg with too short hop port ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    invalidShortPort,
					ChannelId: "channel-0",
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid msg with too long hop port ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    invalidLongPort,
					ChannelId: "channel-0",
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid msg with non-alpha hop port ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    invalidPort,
					ChannelId: "channel-0",
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid msg with too long hop channel ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    "transfer",
					ChannelId: invalidLongChannel,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid msg with too short hop channel ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    "transfer",
					ChannelId: invalidShortChannel,
				},
			),
			host.ErrInvalidID,
		},
		{
			"invalid msg with non-alpha hop channel ID",
			types.NewForwardingInfo(
				"",
				&types.Hop{
					PortId:    "transfer",
					ChannelId: invalidChannel,
				},
			),
			host.ErrInvalidID,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc

			err := tc.forwardingInfo.Validate()

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
			PortId:    "transfer",
			ChannelId: "channel-0",
		}
	}
	return hops
}
