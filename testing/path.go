package ibctesting

// Path
type Path struct {
	Name      string // TODO: remove?
	EndpointA *Endpoint
	EndpointB *Endpoint
}

func NewPath(name string, chainA, chainB *TestChain) *Path {
	endpointA := NewEndpoint(chainA)
	endpointB := NewEndpoint(chainB)

	endpointA.Counterparty = endpointB
	endpointB.Counterparty = endpointA

	return &Path{
		Name:      name,
		EndpointA: endpointA,
		EndpointB: endpointB,
	}
}
