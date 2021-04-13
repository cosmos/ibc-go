package ibctesting

// Path
type Path struct {
	EndpointA *Endpoint
	EndpointB *Endpoint
}

func NewPath(chainA, chainB *TestChain) *Path {
	endpointA := NewEndpoint(chainA)
	endpointB := NewEndpoint(chainB)

	endpointA.Counterparty = endpointB
	endpointB.Counterparty = endpointA

	return &Path{
		EndpointA: endpointA,
		EndpointB: endpointB,
	}
}
