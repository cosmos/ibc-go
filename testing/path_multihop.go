package ibctesting

import (
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	"github.com/stretchr/testify/suite"
)

// PathM represents a multihop channel path between two chains.
type PathM struct {
	EndpointA *EndpointM
	EndpointZ *EndpointM
}

// SetChannelOrdered sets the channel order for both endpoints to ORDERED. Default channel is Unordered.
func (path *PathM) SetChannelOrdered() {
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointZ.ChannelConfig.Order = channeltypes.ORDERED
}

// LinkedPaths is a list of linked ibc paths, A -> B -> C -> ... -> Z, where {A,B,C,...,Z} are chains, and A/Z is the first/last chain endpoint.
type LinkedPaths []*Path

// CreateLinkedChains creates `num` chains and set up a Path between each pair of chains
// return the coordinator, the `num` chains, and `num-1` connected Paths
func CreateLinkedChains(
	t *suite.Suite,
	num int,
) (*Coordinator, LinkedPaths) {
	coord := NewCoordinator(t.T(), num)
	paths := make([]*Path, num-1)

	for i := 0; i < num-1; i++ {
		paths[i] = NewPath(coord.GetChain(GetChainID(i+1)), coord.GetChain(GetChainID(i+2)))
	}

	// create connections for each path
	for _, path := range paths {
		path := path
		t.Require().Equal(path.EndpointA.ConnectionID, "")
		t.Require().Equal(path.EndpointB.ConnectionID, "")
		coord.SetupConnections(path)
		t.Require().NotEqual(path.EndpointA.ConnectionID, "")
		t.Require().NotEqual(path.EndpointB.ConnectionID, "")
	}

	return coord, paths
}

// ToPathM converts a LinkedPaths to a PathM where the EndpointA has the same linking paths as the LinkedPaths and
// EndpointZ has the reverse linking paths.
func (paths LinkedPaths) ToPathM() *PathM {
	a, z := NewEndpointMFromLinkedPaths(paths)
	return &PathM{
		EndpointA: &a,
		EndpointZ: &z,
	}
}

// A returns the first chain in the paths, aka. the source chain.
func (paths LinkedPaths) A() *Endpoint {
	return paths[0].EndpointA
}

// Z returns the last chain in the paths, aka. the destination chain.
func (paths LinkedPaths) Z() *Endpoint {
	return paths[len(paths)-1].EndpointB
}

// Reverse a list of paths from chain A to chain Z.
// Return a list of paths from chain Z to chain A, where the endpoints A/B are also swapped.
func (paths LinkedPaths) Reverse() LinkedPaths {
	var reversed LinkedPaths
	for i := range paths {
		orgPath := paths[len(paths)-1-i]
		path := Path{
			EndpointA: orgPath.EndpointB,
			EndpointB: orgPath.EndpointA,
		}
		reversed = append(reversed, &path)
	}
	return reversed
}
