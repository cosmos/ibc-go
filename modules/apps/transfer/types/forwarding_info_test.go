package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestForwardingInfo_Validate(t *testing.T) {
	tests := []struct {
		name           string
		forwardingInfo types.ForwardingInfo
		expError       error
	}{
		{
			"valid msg no hops",
			types.ForwardingInfo{
				Hops: []*types.Hop{},
				Memo: "",
			},
			nil,
		},
		{
			"valid msg nil hops",
			types.ForwardingInfo{
				Hops: nil,
				Memo: "",
			},
			nil,
		},
		{
			"valid msg with hops",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    "transfer",
						ChannelId: "channel-0",
					},
				},
				Memo: "",
			},
			nil,
		},
		{
			"valid msg with memo",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    "transfer",
						ChannelId: "channel-0",
					},
					{
						PortId:    "transfer",
						ChannelId: "channel-1",
					},
				},
				Memo: testMemo1,
			},
			nil,
		},
		{
			"valid msg with max hops",
			types.ForwardingInfo{
				Hops: generateHops(types.MaximumNumberOfForwardingHops),
				Memo: "",
			},
			nil,
		},
		{
			"valid msg with max memo length",
			types.ForwardingInfo{
				Hops: []*types.Hop{},
				Memo: ibctesting.GenerateString(types.MaximumMemoLength),
			},
			nil,
		},
		{
			"invalid msg with too many hops",
			types.ForwardingInfo{
				Hops: generateHops(types.MaximumNumberOfForwardingHops + 1),
				Memo: "",
			},
			types.ErrInvalidForwardingInfo,
		},
		{
			"invalid msg with too long memo",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    "transfer",
						ChannelId: "channel-0",
					},
				},
				Memo: ibctesting.GenerateString(types.MaximumMemoLength + 1),
			},
			types.ErrInvalidMemo,
		},
		{
			"invalid msg with too short hop port ID",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    invalidShortPort,
						ChannelId: "channel-0",
					},
				},
				Memo: "",
			},
			host.ErrInvalidID,
		},
		{
			"invalid msg with too long hop port ID",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    invalidLongPort,
						ChannelId: "channel-0",
					},
				},
				Memo: "",
			},
			host.ErrInvalidID,
		},
		{
			"invalid msg with non-alpha hop port ID",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    invalidPort,
						ChannelId: "channel-0",
					},
				},
				Memo: "",
			},
			host.ErrInvalidID,
		},
		{
			"invalid msg with too long hop channel ID",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    "transfer",
						ChannelId: invalidLongChannel,
					},
				},
				Memo: "",
			},
			host.ErrInvalidID,
		},
		{
			"invalid msg with too short hop channel ID",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    "transfer",
						ChannelId: invalidShortChannel,
					},
				},
				Memo: "",
			},
			host.ErrInvalidID,
		},
		{
			"invalid msg with non-alpha hop channel ID",
			types.ForwardingInfo{
				Hops: []*types.Hop{
					{
						PortId:    "transfer",
						ChannelId: invalidChannel,
					},
				},
				Memo: "",
			},
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
