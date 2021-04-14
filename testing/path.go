package ibctesting

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

// TODO create RelayPacket function which relays the packet,
// it will try both chains for the packet so caller doesn't
// need to specify which chain the packet was sent on
