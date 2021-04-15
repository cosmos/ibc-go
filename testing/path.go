package ibctesting

import (
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

// Path
type Path struct {
	EndpointA *Endpoint
	EndpointB *Endpoint
}

func NewPath(chainA, chainB *TestChain) *Path {
	endpointA := NewDefaultEndpoint(chainA)
	endpointB := NewDefaultEndpoint(chainB)

	endpointA.Counterparty = endpointB
	endpointB.Counterparty = endpointA

	return &Path{
		EndpointA: endpointA,
		EndpointB: endpointB,
	}
}

// SetChannelOrdered sets the channel order for both endpoints to ORDERED.
func (path *Path) SetChannelOrdered() {
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
}

// TODO create RelayPacket function which relays the packet,
// it will try both chains for the packet so caller doesn't
// need to specify which chain the packet was sent on
