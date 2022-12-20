package multihop_test

import (
	"fmt"
	"testing"

	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
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
	_, paths := ibctesting.CreateLinkedChains(&t.Suite, 5)

	// From chain A to Z, generate and verifyMembership multi-hop proof for A's connectionEnd of on Z
	verifyMembership := func(paths ibctesting.LinkedPaths) error {
		connKey := host.ConnectionKey(paths.A().ConnectionID)
		connEnd := paths.A().GetConnection()
		proofValue := paths.A().Chain.Codec.MustMarshal(&connEnd)

		proofs, err := ibctesting.GenerateMultiHopProof(paths, connKey, proofValue, true)
		t.Require().NoError(err, "failed to generate multi-hop proof for connection")

		keyPath := commitmenttypes.NewMerklePath(string(connKey))
		keyMerklePath, err := commitmenttypes.ApplyPrefix(paths.A().Chain.GetPrefix(), keyPath)
		t.Require().NoError(err, "failed to generate merklepath for key")

		return ibctesting.VerifyMultiHopProofMembership(
			paths.Z(),
			proofs,
			&keyMerklePath,
			proofValue,
		)
	}
	verifyNonMembership := func(paths ibctesting.LinkedPaths) error {
		connKey := host.ConnectionKey("non-existent-connection-id")
		proofs, err := ibctesting.GenerateMultiHopProof(paths, connKey, nil, true)
		t.Require().NoError(err, "failed to generate non-membership multi-hop proof")
		keyPath := commitmenttypes.NewMerklePath(string(connKey))
		keyMerklePath, err := commitmenttypes.ApplyPrefix(paths.A().Chain.GetPrefix(), keyPath)
		t.Require().NoError(err, "failed to generate merklepath for key")
		return ibctesting.VerifyMultiHopProofNonMembership(
			paths.Z(),
			proofs,
			&keyMerklePath,
		)
	}

	t.Require().
		NoError(verifyMembership(paths), "failed to verify multi-hop membership proof for connection end (A -> Z)")
	t.Require().
		NoError(verifyNonMembership(paths), "failed to verify multi-hop non-membership proof for connection end (A -> Z)")

	pathsZ2A := paths.Reverse().UpdateClients()
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
			_, paths := ibctesting.CreateLinkedChains(&t.Suite, 5)
			tc.malleate(paths)
			kvPath := host.ConnectionKey(paths.A().ConnectionID)
			connEnd := paths.A().GetConnection()
			value, err := connEnd.Marshal()
			t.NoError(err)
			_, err = ibctesting.GenerateMultiHopProof(paths, kvPath, value, true)
			t.Require().ErrorContains(err, "failed to verify consensus state proof")

		})
	}
}

// Test that A's consensus state can be verified by chain Z

func (t *proofTestSuite) TestMultiConsensusProof() {
	_, paths := ibctesting.CreateLinkedChains(&t.Suite, 5)

	// From chain A to Z, generate and verify multi-hop proof for A's consensus state on Z
	verify := func(paths ibctesting.LinkedPaths) error {
		consStateProofs, connectionProofs, err := ibctesting.GenerateMultiHopConsensusProof(paths)
		t.NoError(err, "failed to generate multi-hop proof for consensus state")
		return ibctesting.VerifyMultiHopConsensusStateProof(paths.Z(), consStateProofs, connectionProofs)
	}

	// t.NoError(verify(paths), "failed to verify multi-hop proof for consensus state (A -> Z)")
	t.NoError(verify(paths.Reverse().UpdateClients()), "failed to verify multi-hop proof for consensus state (Z -> A)")

}

// Given chain x-y-z, test that x's consensus state can be verified by y at y's height on z.
// This test is how single-hop-ibc verifies various ibc constructs, eg. clients, connections, channels, packets, etc.

func (t *proofTestSuite) TestClientStateProof() {
	_, paths := ibctesting.CreateLinkedChains(&t.Suite, 3)

	// lastClientState is the 2nd to last chain's client state, eg. chain Y's clientState in a connection from chains A
	// to Z
	endZ := paths.Z()
	endY := endZ.Counterparty

	// Test clientStateZY verifies Z has a consState for Y
	verifyConsState := func(endY, endZ *ibctesting.Endpoint) {

		heightZY := endY.GetClientState().GetLatestHeight()
		heightYZ := endZ.GetClientState().GetLatestHeight()
		proofYZ, _ := ibctesting.GetConsStateProof(endZ, heightZY, heightYZ, endZ.ClientID)
		consStateYZ := endZ.GetConsensusState(heightYZ)

		var merkleProofYZ commitmenttypes.MerkleProof
		err := merkleProofYZ.Unmarshal(proofYZ)
		t.NoError(err)

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
	coord, paths := ibctesting.CreateLinkedChains(&t.Suite, N)
	t.Require().Equal(len(coord.Chains), N)
	t.Require().Equal(len(paths), N-1)

	verify := func(
		pathGP, pathPS *ibctesting.Path,
		rootPS exported.Root,
		heightPS exported.Height,
	) {
		heightGP := pathGP.EndpointB.GetClientState().GetLatestHeight()
		consGP, err := ibctesting.GetConsensusState(pathGP.EndpointB, heightGP)
		t.NoError(err)
		proofConsGP, _ := ibctesting.GetConsStateProof(pathGP.EndpointB, heightPS, heightGP, pathGP.EndpointB.ClientID)
		var merkleProofConsGP commitmenttypes.MerkleProof
		err = merkleProofConsGP.Unmarshal(proofConsGP)
		t.NoError(err)

		consStatePrefix, err := ibctesting.GetConsensusStatePrefix(pathGP.EndpointB, heightGP)
		t.NoError(err)
		err = merkleProofConsGP.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
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
			Logf("offset=[%d]\tCons[%s] verifies Cons[%s]\n", offset, pathStr(pathPS), pathStr(pathGP))

		heightPS := pathPS.EndpointB.GetClientState().GetLatestHeight()
		verify(
			pathGP, pathPS,
			pathPS.EndpointB.GetConsensusState(heightPS).GetRoot(),
			heightPS,
		)
	}
}

func pathStr(path *ibctesting.Path) string {
	return fmt.Sprintf("%s-%s", path.EndpointA.Chain.ChainID, path.EndpointB.Chain.ChainID)
}

// output endpoint for debugging
func (t *proofTestSuite) printEndpoint(endpoint *ibctesting.Endpoint) {
	t.T().Logf(
		"self: %s, %s, counter-party: %s, %s, latest height: %d [%s]\n",
		endpoint.ConnectionID, endpoint.ClientID, endpoint.Counterparty.ConnectionID, endpoint.Counterparty.ClientID, endpoint.Chain.LastHeader.GetHeight(), endpoint.Chain.ChainID,
	)
}

func TestProofTestSuite(t *testing.T) {
	suite.Run(t, new(proofTestSuite))
}
