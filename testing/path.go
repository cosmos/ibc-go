package ibctesting

import (
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

// Path
type Path struct {
	Name      string
	EndpointA *Endpoint
	EndpointB *Endpoint

	ChannelOrder channeltypes.Order
}

func NewPath(name string, chainA, chainB *TestChain) *Path {
	endpointA := NewEndpoint(chainA)
	endpointB := NewEndpoint(chainB)

	endpointA.Counterparty = endpointB
	endpointB.Counterparty = endpointA

	return &Path{
		Name:         name,
		EndpointA:    endpointA,
		EndpointB:    endpointB,
		ChannelOrder: channeltypes.UNORDERED,
	}
}
