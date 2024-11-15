package types

import (
	"fmt"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
)

// String returns the Hop in the format:
// <portID>/<channelID>
func (h Hop) String() string {
	return fmt.Sprintf("%s/%s", h.PortId, h.ChannelId)
}

// V1HopsToV2Hops converts a slice of v1 Hop to a slice of V2 Hop.
func V1HopsToV2Hops(hops []types.Hop) []Hop {
	v2hops := make([]Hop, len(hops))
	for i, h := range hops {
		v2hops[i] = Hop{PortId: h.PortId, ChannelId: h.ChannelId}
	}
	return v2hops
}

// V2HopsToV1Hops converts a slice of v2 Hop to a slice of V1 Hop.
func V2HopsToV1Hops(hops []Hop) []types.Hop {
	v1hops := make([]types.Hop, len(hops))
	for i, h := range hops {
		v1hops[i] = types.Hop{PortId: h.PortId, ChannelId: h.ChannelId}
	}
	return v1hops
}
