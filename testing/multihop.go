package ibctesting

import (
	"fmt"

	ics23 "github.com/confio/ics23/go"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint/types"
	"github.com/stretchr/testify/suite"
)

// GenerateMultiHopProof generate a proof for key path on the source (aka. paths[0].EndpointA) verified on the dest chain (aka.
// paths[len(paths)-1].EndpointB) and all intermediate consensus states.
//
// The first proof can be either a membership proof or a non-membership proof depending on if the key exists on the
// source chain.
func GenerateMultiHopProof(paths LinkedPaths, keyPathToProve string, expectedVal []byte) (*channeltypes.MsgMultihopProofs, error) {
	if len(keyPathToProve) == 0 {
		panic("path cannot be empty")
	}

	if len(paths) < 2 {
		panic("paths must have at least two elements")
	}
	srcEnd := paths.A()

	var proofs channeltypes.MsgMultihopProofs
	// generate proof for key path on the source chain
	{
		nextPath := paths[0].EndpointB
		heightBC := nextPath.GetClientState().GetLatestHeight()
		fmt.Printf("heightBC: %d\n", heightBC.GetRevisionHeight())
		// srcEnd.counterparty's proven height on its next connected chain
		provenHeight := srcEnd.Counterparty.GetClientState().GetLatestHeight()
		fmt.Printf("provenHeight: %d\n", provenHeight.GetRevisionHeight())
		proof, _ := srcEnd.Chain.QueryProofAtHeight([]byte(keyPathToProve), int64(provenHeight.GetRevisionHeight()))
		var proofKV commitmenttypes.MerkleProof
		if err := srcEnd.Chain.Codec.Unmarshal(proof, &proofKV); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proof: %w", err)
		}

		prefixedKey, err := commitmenttypes.ApplyPrefix(
			srcEnd.Chain.GetPrefix(),
			commitmenttypes.NewMerklePath(keyPathToProve),
		)

		// membership proof
		if len(expectedVal) > 0 {

			fmt.Printf("VERIFYING MEMBERSHIP\n")
			fmt.Printf("nextPath.GetConsensusState(heightBC).GetRoot(): %x\n",
				nextPath.GetConsensusState(heightBC).GetRoot().GetHash())
			fmt.Printf("key: %s\n", prefixedKey.String())
			fmt.Printf("val: %x\n", expectedVal)

			// check expected val
			if err := proofKV.VerifyMembership(
				GetProofSpec(srcEnd),
				nextPath.GetConsensusState(heightBC).GetRoot(),
				prefixedKey,
				expectedVal,
			); err != nil {
				return nil, fmt.Errorf(
					"failed to verify consensus state proof of [%s] on [%s] with [%s].ConsState on [%s]: %w\nconsider update [%s]'s client on [%s]",
					srcEnd.Counterparty.Chain.ChainID,
					srcEnd.Chain.ChainID,
					srcEnd.Chain.ChainID,
					nextPath.Chain.ChainID,
					err,
					srcEnd.Chain.ChainID,
					nextPath.Chain.ChainID,
				)
			}
		}
		// TODO: make sure the the keypath value actually exists
		// TODO: also add flag for membership proof or non-membership proof

		if err != nil {
			return nil, fmt.Errorf("failed to apply prefix to key path: %w", err)
		}

		proofs.KeyProof = &channeltypes.MultihopProof{
			Proof:       proof,
			Value:       expectedVal,
			PrefixedKey: &prefixedKey,
		}
	}

	consStateProofs, connectionProofs, err := GenerateMultiHopConsensusProof(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to generate consensus proofs: %w", err)
	}
	proofs.ConsensusProofs = consStateProofs
	proofs.ConnectionProofs = connectionProofs

	return &proofs, nil
}

// GenerateMultiHopConsensusProof generates a proof of consensus state of paths[0].EndpointA verified on
// paths[len(paths)-1].EndpointB and all intermediate consensus states.
// TODO: Would it be beneficial to batch the consensus state and connection proofs?
func GenerateMultiHopConsensusProof(paths []*Path) ([]*channeltypes.MultihopProof, []*channeltypes.MultihopProof, error) {
	if len(paths) < 2 {
		panic("paths must have at least two elements")
	}

	var consStateProofs []*channeltypes.MultihopProof
	var connectionProofs []*channeltypes.MultihopProof

	// iterate all but the last path
	for i := 0; i < len(paths)-1; i++ {
		path, nextPath := paths[i], paths[i+1]
		// self is where the proof is queried and generated
		self := path.EndpointB

		heightAB := path.EndpointB.GetClientState().GetLatestHeight()
		heightBC := nextPath.EndpointB.GetClientState().GetLatestHeight()
		consStateAB, found := self.Chain.GetConsensusState(self.ClientID, heightAB)
		if !found {
			return nil, nil, fmt.Errorf(
				"consensus state not found for height %s on chain %s",
				heightAB,
				self.Chain.ChainID,
			)
		}

		keyPrefixedConsAB, err := GetConsensusStatePrefix(self, heightAB)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get consensus state prefix at height %d and revision %d: %w", heightAB.GetRevisionHeight(), heightAB.GetRevisionHeight(), err)
		}
		consensusProof, _ := GetConsStateProof(self, heightBC, heightAB, self.ClientID)

		var consensusStateMerkleProof commitmenttypes.MerkleProof
		if err := self.Chain.Codec.Unmarshal(consensusProof, &consensusStateMerkleProof); err != nil {
			return nil, nil, fmt.Errorf("failed to get proof for consensus state on chain %s: %w", self.Chain.ChainID, err)
		}

		value, err := self.Chain.Codec.MarshalInterface(consStateAB)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal consensus state: %w", err)
		}

		// fmt.Printf("nextPath.EndpointB.GetConsensusState(heightBC).GetRoot(): %x\n", nextPath.EndpointB.GetConsensusState(heightBC).GetRoot())
		// fmt.Printf("keyPrefixedConsAB: %s\n", keyPrefixedConsAB.String())
		// fmt.Printf("value: %x\n", value)
		// ensure consStateAB is verified by consStateBC, where self is chain B
		if err := consensusStateMerkleProof.VerifyMembership(
			GetProofSpec(self),
			nextPath.EndpointB.GetConsensusState(heightBC).GetRoot(),
			keyPrefixedConsAB,
			value,
		); err != nil {
			return nil, nil, fmt.Errorf(
				"failed to verify consensus state proof of [%s] on [%s] with [%s].ConsState on [%s]: %w\nconsider update [%s]'s client on [%s]",
				self.Counterparty.Chain.ChainID,
				self.Chain.ChainID,
				self.Chain.ChainID,
				nextPath.EndpointB.Chain.ChainID,
				err,
				self.Chain.ChainID,
				nextPath.EndpointB.Chain.ChainID,
			)
		}

		consStateProofs = append(consStateProofs, &channeltypes.MultihopProof{
			Proof:       consensusProof,
			Value:       value,
			PrefixedKey: &keyPrefixedConsAB,
		})

		// now to connection proof verification
		connectionKey, err := GetPrefixedConnectionKey(self)
		if err != nil {
			return nil, nil, err
		}
		connectionProof, _ := GetConnectionProof(self, heightBC, self.ConnectionID)
		var connectionMerkleProof commitmenttypes.MerkleProof
		if err := self.Chain.Codec.Unmarshal(connectionProof, &connectionMerkleProof); err != nil {
			return nil, nil, fmt.Errorf("failed to get proof for consensus state on chain %s: %w", self.Chain.ChainID, err)
		}

		connection := self.GetConnection()
		value, err = connection.Marshal()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal connection end: %w", err)
		}

		// fmt.Printf("nextPath.EndpointB.GetConsensusState(heightBC).GetRoot(): %x\n", nextPath.EndpointB.GetConsensusState(heightBC).GetRoot())
		// fmt.Printf("connectionProof.PrefixedKey: %s\n", connectionKey.String())
		// fmt.Printf("value: %x\n", value)
		// ensure consStateAB is verified by consStateBC, where self is chain B
		if err := connectionMerkleProof.VerifyMembership(
			GetProofSpec(self),
			nextPath.EndpointB.GetConsensusState(heightBC).GetRoot(),
			connectionKey,
			value,
		); err != nil {
			return nil, nil, fmt.Errorf(
				"failed to verify connection proof of [%s] on [%s] with [%s].ConnectionEnd on [%s]: %w\nconsider update [%s]'s client on [%s]",
				self.Counterparty.Chain.ChainID,
				self.Chain.ChainID,
				self.Chain.ChainID,
				nextPath.EndpointB.Chain.ChainID,
				err,
				self.Chain.ChainID,
				nextPath.EndpointB.Chain.ChainID,
			)
		}

		connectionProofs = append(connectionProofs, &channeltypes.MultihopProof{
			Proof:       connectionProof,
			Value:       value,
			PrefixedKey: &connectionKey,
		})
	}

	return consStateProofs, connectionProofs, nil
}

// VerifyMultiHopConsensusStateProof verifies the consensus state of paths[0].EndpointA on paths[len(paths)-1].EndpointB.
func VerifyMultiHopConsensusStateProof(endpoint *Endpoint, consensusProofs []*channeltypes.MultihopProof, connectionProofs []*channeltypes.MultihopProof) error {
	lastConsstate := endpoint.GetConsensusState(endpoint.GetClientState().GetLatestHeight())
	var consState exported.ConsensusState
	//var connectionEnd connectiontypes.ConnectionEnd
	for i := len(consensusProofs) - 1; i >= 0; i-- {
		consStateProof := consensusProofs[i]
		connectionProof := connectionProofs[i]
		if err := endpoint.Chain.Codec.UnmarshalInterface(consStateProof.Value, &consState); err != nil {
			return fmt.Errorf("failed to unpack consesnsus state: %w", err)
		}

		var proof commitmenttypes.MerkleProof
		if err := endpoint.Chain.Codec.Unmarshal(consStateProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal consensus state proof: %w", err)
		}

		if err := proof.VerifyMembership(
			GetProofSpec(endpoint),
			lastConsstate.GetRoot(),
			*consStateProof.PrefixedKey,
			consStateProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify consensus proof on chain '%s': %w", endpoint.Chain.ChainID, err)
		}

		proof.Reset()
		if err := endpoint.Chain.Codec.Unmarshal(connectionProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal connection proof: %w", err)
		}

		// fmt.Printf("root: %x\n", lastConsstate.GetRoot())
		// fmt.Printf("connectionProof.PrefixedKey: %s\n", connectionProof.PrefixedKey.String())
		// fmt.Printf("value: %x\n", connectionProof.Value)
		if err := proof.VerifyMembership(
			GetProofSpec(endpoint),
			lastConsstate.GetRoot(),
			*connectionProof.PrefixedKey,
			connectionProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify connection proof on chain '%s': %w", endpoint.Chain.ChainID, err)
		}

		lastConsstate = consState
	}
	return nil
}

// VerifyMultiHopProofMembership verifies a multihop membership proof including all intermediate state proofs.
func VerifyMultiHopProofMembership(endpoint *Endpoint, proofs *channeltypes.MsgMultihopProofs) error {
	if len(proofs.ConsensusProofs) < 1 {
		return fmt.Errorf(
			"proof must have at least two elements where the first one is the proof for the key and the rest are for the consensus states",
		)
	}
	if len(proofs.ConsensusProofs) != len(proofs.ConnectionProofs) {
		return fmt.Errorf("the number of connection (%d) and consensus (%d) proofs must be equal",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}
	if err := VerifyMultiHopConsensusStateProof(endpoint, proofs.ConsensusProofs, proofs.ConnectionProofs); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}
	var keyProof commitmenttypes.MerkleProof
	if err := endpoint.Chain.Codec.Unmarshal(proofs.KeyProof.Proof, &keyProof); err != nil {
		return fmt.Errorf("failed to unmarshal key proof: %w", err)
	}
	var secondConsState exported.ConsensusState
	if err := endpoint.Chain.Codec.UnmarshalInterface(proofs.ConsensusProofs[0].Value, &secondConsState); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	return keyProof.VerifyMembership(
		GetProofSpec(endpoint),
		secondConsState.GetRoot(),
		*proofs.KeyProof.PrefixedKey,
		proofs.KeyProof.Value,
	)
}

// VerifyMultiHopProofNonMembership verifies a multihop proof of non-membership including all intermediate state proofs.
func VerifyMultiHopProofNonMembership(endpoint *Endpoint, proofs *channeltypes.MsgMultihopProofs) error {
	if len(proofs.ConsensusProofs) < 1 {
		return fmt.Errorf(
			"proof must have at least two elements where the first one is the proof for the key and the rest are for the consensus states",
		)
	}
	if len(proofs.ConsensusProofs) != len(proofs.ConnectionProofs) {
		return fmt.Errorf("the number of connection (%d) and consensus (%d) proofs must be equal",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}
	if err := VerifyMultiHopConsensusStateProof(endpoint, proofs.ConsensusProofs, proofs.ConnectionProofs); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}
	var keyProof commitmenttypes.MerkleProof
	if err := endpoint.Chain.Codec.Unmarshal(proofs.KeyProof.Proof, &keyProof); err != nil {
		return fmt.Errorf("failed to unmarshal key proof: %w", err)
	}
	var secondConsState exported.ConsensusState
	if err := endpoint.Chain.Codec.UnmarshalInterface(proofs.ConsensusProofs[0].Value, &secondConsState); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	err := keyProof.VerifyNonMembership(
		GetProofSpec(endpoint),
		secondConsState.GetRoot(),
		*proofs.KeyProof.PrefixedKey,
	)
	return err
}

// GetConsensusState returns the consensus state of self's counterparty chain stored on self, where height is according to the counterparty.
func GetConsensusState(self *Endpoint, height exported.Height) ([]byte, error) {
	consensusState := self.GetConsensusState(height)
	return self.Counterparty.Chain.Codec.MarshalInterface(consensusState)
}

// GetConsensusStateProof returns the consensus state proof for the state of self's counterparty chain stored on self, where height is the latest
// self client height.
func GetConsensusStateProof(self *Endpoint) commitmenttypes.MerkleProof {
	proofBz, _ := self.Chain.QueryConsensusStateProof(self.ClientID)
	var proof commitmenttypes.MerkleProof
	self.Chain.Codec.MustUnmarshal(proofBz, &proof)
	return proof
}

// GetConsStateProof returns the merkle proof of consensusState of self's clientId and at `consensusHeight` stored on self at `selfHeight`.
func GetConsStateProof(
	self *Endpoint,
	selfHeight exported.Height,
	consensusHeight exported.Height,
	clientID string,
) ([]byte, exported.Height) {
	consensusKey := host.FullConsensusStateKey(clientID, consensusHeight)
	return self.Chain.QueryProofAtHeight(consensusKey, int64(selfHeight.GetRevisionHeight()))
}

// GetConnectionProof returns the proof of a connection at the specified height
func GetConnectionProof(
	self *Endpoint,
	selfHeight exported.Height,
	connectionID string,
) ([]byte, exported.Height) {
	consensusKey := host.ConnectionKey(connectionID)
	return self.Chain.QueryProofAtHeight(consensusKey, int64(selfHeight.GetRevisionHeight()))
}

// GetConsensusStatePrefix returns the merkle prefix of consensus state of self's counterparty chain at height `consensusHeight` stored on self.
func GetConsensusStatePrefix(self *Endpoint, consensusHeight exported.Height) (commitmenttypes.MerklePath, error) {
	keyPath := commitmenttypes.NewMerklePath(host.FullConsensusStatePath(self.ClientID, consensusHeight))
	return commitmenttypes.ApplyPrefix(self.Chain.GetPrefix(), keyPath)
}

// GetPrefixedConnectionKey returns the connection prefix associated
func GetPrefixedConnectionKey(self *Endpoint) (commitmenttypes.MerklePath, error) {
	keyPath := commitmenttypes.NewMerklePath(host.ConnectionPath(self.ConnectionID))
	return commitmenttypes.ApplyPrefix(self.Chain.GetPrefix(), keyPath)
}

// GetProofSpec returns self counterparty's ProofSpec
func GetProofSpec(self *Endpoint) []*ics23.ProofSpec {
	tmclient := self.GetClientState().(*ibctmtypes.ClientState)
	return tmclient.GetProofSpecs()
}

// LinkedPaths is a list of linked ibc paths, A -> B -> C -> ... -> Z, where {A,B,C,...,Z} are chains, and A/Z is the first/last chain endpoint.
type LinkedPaths []*Path

// Last returns the last Path in LinkedPaths.
func (paths LinkedPaths) Last() *Path {
	return paths[len(paths)-1]
}

// First returns the first Path in LinkedPaths.
func (paths LinkedPaths) First() *Path {
	return paths[0]
}

// A returns the first chain in the paths, aka. the source chain.
func (paths LinkedPaths) A() *Endpoint {
	return paths.First().EndpointA
}

// Z returns the last chain in the paths, aka. the destination chain.
func (paths LinkedPaths) Z() *Endpoint {
	return paths.Last().EndpointB
}

// Reverse a list of paths from chain A to chain Z.
// Return a list of paths from chain Z to chain A, where the endpoints A/B are also swapped.
func (paths LinkedPaths) Reverse() LinkedPaths {
	var reversed LinkedPaths
	for i := range paths {
		// Ensure Z's client on Y, Y's client on X, etc. are all updated
		orgPath := paths[len(paths)-1-i]
		path := Path{
			EndpointA: orgPath.EndpointB,
			EndpointB: orgPath.EndpointA,
		}
		reversed = append(reversed, &path)
	}
	return reversed
}

// UpdateClients iterates through each chain in the path and calls UpdateClient
func (paths LinkedPaths) UpdateClients() LinkedPaths {
	for _, path := range paths {
		if path.EndpointB.ClientID != "" {
			if err := path.EndpointB.UpdateClient(); err != nil {
				panic(err)
			}
		}
	}
	return paths
}

// GetConnectionHops returns connection IDs on {A, B,... Y}
func (paths LinkedPaths) GetConnectionHops() (connectionHops []string) {
	for _, path := range paths {
		connectionHops = append(connectionHops, path.EndpointA.ConnectionID)
	}
	return
}

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

	// for _, chain := range coord.Chains {
	// 	coord.CommitNBlocks(chain, 2)
	// }

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
