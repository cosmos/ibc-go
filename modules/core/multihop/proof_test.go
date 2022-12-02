package multihop_test

import (
	"fmt"
	"testing"

	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	"github.com/stretchr/testify/suite"
)

type proofTestSuite struct {
	suite.Suite
}

// Test that 1. key/value can be verified by A's consensus state; and 2. that A's consState can be verified by chain Z.
//
// This effectively tests that the multi-hop proof of a source chain's key/value pair is correctly verified by the
// destination chain.
func (t *proofTestSuite) TestMultiHopProof() {
	_, _, paths := t.createLinkedChains(5)

	// From chain A to Z, generate and verifyMembership multi-hop proof for A's connectionEnd of on Z
	verifyMembership := func(paths ibctesting.LinkedPaths) error {
		connPath := host.ConnectionPath(paths.A().ConnectionID)
		proofs, err := ibctesting.GenerateMultiHopProof(paths, connPath)
		t.Require().NoError(err, "failed to generate multi-hop proof for connection")
		connEnd := paths.A().GetConnection()
		return ibctesting.VerifyMultiHopProofMembership(
			paths.Z(),
			proofs,
			paths.A().Chain.Codec.MustMarshal(&connEnd),
		)
	}
	verifyNonMembership := func(paths ibctesting.LinkedPaths) error {
		connPath := host.ConnectionPath("non-existent-connection-id")
		proofs, err := ibctesting.GenerateMultiHopProof(paths, connPath)
		t.Require().NoError(err, "failed to generate non-membership multi-hop proof")
		return ibctesting.VerifyMultiHopProofNonMembership(
			paths.Z(),
			proofs,
		)
	}

	t.Require().
		NoError(verifyMembership(paths), "failed to verify multi-hop membership proof for connection end (A -> Z)")
	t.Require().
		NoError(verifyNonMembership(paths), "failed to verify multi-hop non-membership proof for connection end (A -> Z)")

	pathsZ2A := paths.Reverse()
	t.Require().
		NoError(verifyMembership(pathsZ2A), "failed to verify multi-hop membership proof for connection end (Z -> A)")
	t.Require().
		NoError(verifyNonMembership(pathsZ2A), "failed to verify multi-hop non-membership proof for connection end (Z -> A)")
}

func (t *proofTestSuite) TestMultiHopProofNegative() {
	testCases := []struct {
		name     string
		malleate func(paths ibctesting.LinkedPaths)
	}{
		{"invalid consState proof", func(paths ibctesting.LinkedPaths) {
			// update heightAB, but do not propogate to following chains
			paths[0].EndpointB.UpdateClient()
		}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func() {
			_, _, paths := t.createLinkedChains(5)
			tc.malleate(paths)
			kvPath := host.ConnectionPath(paths.A().ConnectionID)
			_, err := ibctesting.GenerateMultiHopProof(paths, kvPath)
			t.Require().ErrorContains(err, "failed to verify consensus state proof")

		})
	}
}

// Test that A's consensus state can be verified by chain Z
func (t *proofTestSuite) TestMultiConsensusProof() {
	_, _, paths := t.createLinkedChains(5)

	// From chain A to Z, generate and verify multi-hop proof for A's consensus state on Z
	verify := func(paths ibctesting.LinkedPaths) error {
		consStateProofs, err := ibctesting.GenerateMultiHopConsensusProof(paths)
		t.Require().NoError(err, "failed to generate multi-hop proof for consensus state")
		return ibctesting.VerifyMultiHopConsensusStateProof(paths.Z(), consStateProofs)
	}

	t.Require().
		NoError(verify(paths), "failed to verify multi-hop proof for consensus state (A -> Z)")
	t.Require().
		NoError(verify(paths.Reverse()), "failed to verify multi-hop proof for consensus state (Z -> A)")

}

// Given chain x-y-z, test that x's consensus state can be verified by y at y's height on z.
// This test is how single-hop-ibc verifies various ibc constructs, eg. clients, connections, channels, packets, etc.
func (t *proofTestSuite) TestClientStateProof() {
	_, _, paths := t.createLinkedChains(3)

	// lastClientState is the 2nd to last chain's client state, eg. chain Y's clientState in a connection from chains A
	// to Z
	endZ := paths.Z()
	endY := endZ.Counterparty

	// Test clientStateZY verifies Z has a consState for Y
	verifyConsState := func(endY, endZ *ibctesting.Endpoint) {

		heightZY := endY.GetClientState().GetLatestHeight()
		heightYZ := endZ.GetClientState().GetLatestHeight()
		merkleProofYZ, err := ibctesting.GetConsStateProof(endZ, heightZY, heightYZ, endZ.ClientID)
		t.NoError(err)
		consStateYZ := endZ.GetConsensusState(heightYZ)

		// Test that clientStateZ/Y can verify chainZ has a consensus state for Z's clientID, consensus{State,Height,Proof}Y/Z
		err = endY.GetClientState().VerifyClientConsensusState(
			endY.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(
				endY.Chain.GetContext(),
				endY.ClientID,
			),
			endY.Chain.Codec,
			heightZY, // client state height
			endY.Counterparty.ClientID,
			heightYZ, // consensus state height
			endY.GetConnection().Counterparty.GetPrefix(),
			endZ.Chain.Codec.MustMarshal(&merkleProofYZ),
			consStateYZ,
		)
		t.Require().NoError(err, "failed to verify consensus state proof")
	}

	verifyConsState(endY, endZ)
	// below fails due to one sided update causing merkle root mismtach
	// testVerifyClientConsensusState(endZ, endY)

	// This works because the client state is updated on both sides
	endZ.UpdateClient()
	verifyConsState(endZ, endY)
}

// Suppose we have three chains G(grandpa), P(parent), S(self) and two paths G-P and P-C,
// TestGrandpaConsStateProof verifies that G's consensus state on P is proven by P's consensus state on S, or
// consPS.Verify(consGP), `heightPS` is the height of P's consensus state on S,
// `rootPS` is the merkle root of P's consensus state at heightSP on S.
func (t *proofTestSuite) TestGrandpaConsStateProof() {
	// Create N chains and N-1 Paths like a linked list
	N := 5
	_, chains, paths := t.createLinkedChains(N)
	t.Require().Equal(len(chains), N)
	t.Require().Equal(len(paths), N-1)

	verify := func(
		pathGP, pathPS *ibctesting.Path,
		rootPS exported.Root,
		heightPS exported.Height,
	) {
		heightGP := pathGP.EndpointB.GetClientState().GetLatestHeight()
		consGP, err := ibctesting.GetConsensusState(pathGP.EndpointB, heightGP)
		t.NoError(err)
		proofConsGP, err := ibctesting.GetConsStateProof(pathGP.EndpointB, heightPS, heightGP, pathGP.EndpointB.ClientID)
		t.NoError(err)
		consStatePrefix, err := ibctesting.GetConsensusStatePrefix(pathGP.EndpointB, heightGP)
		t.NoError(err)
		err = proofConsGP.VerifyMembership(
			ibctesting.GetProofSpec(pathGP.EndpointB),
			rootPS,
			consStatePrefix,
			consGP,
		)
		t.Require().NoErrorf(err, "failed to verify grandpa consensus state proof")
	}

	// A-Y's consensusState stored on B-Z
	consStates := make([]exported.ConsensusState, len(paths))
	for i, path := range paths {
		latestHeight := path.EndpointB.GetClientState().GetLatestHeight()
		cs := path.EndpointB.GetConsensusState(latestHeight)
		t.Require().NotNil(cs)
		consStates[i] = cs
	}

	// Suppose a list of chains from A to Z,
	// when offset == 2, verify that X's consState on Y is proven by Y's consState on Z, or
	// consStateYZ.Verify(consStateXY);
	// when offset == 3, consStateXY.Verify(consStateWX); and so on until offset == N-1,
	// consStateBC.Verify(consStateAB) is the last verification.
	for offset := 2; offset < N; offset++ {
		pathPS := paths[N-offset]
		pathGP := paths[N-offset-1]
		t.T().
			Logf("offset=[%d]\tCons[%s] verifies Cons[%s]\n", offset, t.PathStr(pathPS), t.PathStr(pathGP))

		heightPS := pathPS.EndpointB.GetClientState().GetLatestHeight()
		verify(
			pathGP, pathPS,
			pathPS.EndpointB.GetConsensusState(heightPS).GetRoot(),
			heightPS,
		)
	}
}

// create `num` chains and set up a Path between each pair of chains
// return the coordinator, the `num` chains, and `num-1` connected Paths
func (t *proofTestSuite) createLinkedChains(
	num int,
) (*ibctesting.Coordinator, []*ibctesting.TestChain, ibctesting.LinkedPaths) {
	coord, chains := t.createChains(num)
	paths := make([]*ibctesting.Path, num-1)

	for i := 0; i < num-1; i++ {
		paths[i] = ibctesting.NewPath(chains[i], chains[i+1])
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

	return coord, chains, paths
}

func (t *proofTestSuite) PathStr(path *ibctesting.Path) string {
	return fmt.Sprintf("%s-%s", path.EndpointA.Chain.ChainID, path.EndpointB.Chain.ChainID)
}

// output endpoint for debugging
func (t *proofTestSuite) printEndpoint(endpoint *ibctesting.Endpoint) {
	t.T().Logf(
		"self: %s, %s, counter-party: %s, %s, latest height: %d [%s]\n",
		endpoint.ConnectionID, endpoint.ClientID, endpoint.Counterparty.ConnectionID, endpoint.Counterparty.ClientID, endpoint.Chain.LastHeader.GetHeight(), endpoint.Chain.ChainID,
	)
}

// CreateChains creates numChains test chains and returns a coordinator along with the array of chains.
func (t *proofTestSuite) createChains(numChains int) (*ibctesting.Coordinator, []*ibctesting.TestChain) {
	var chains []*ibctesting.TestChain
	coord := ibctesting.NewCoordinator(t.T(), numChains)
	for i := 0; i < numChains; i++ {
		// chainId starts at 1, eg. testchain1, testchain2, ...
		chains = append(chains, coord.GetChain(ibctesting.GetChainID(i+1)))
	}
	return coord, chains
}

func TestProofTestSuite(t *testing.T) {
	suite.Run(t, new(proofTestSuite))
}
