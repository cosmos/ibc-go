package ibctesting

import (
	"fmt"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint/types"
)

// GenerateMultiHopProof generate a proof for key path on the source (aka. paths[0].EndpointA) verified on the dest chain (aka.
// paths[len(paths)-1].EndpointB) and all intermediate consensus states.
//
// The first proof can be either a membership proof or a non-membership proof depending on if the key exists on the
// source chain.
func GenerateMultiHopProof(paths LinkedPaths, keyPathToProve string) (*channeltypes.MsgConsStateProofs, error) {
	if len(keyPathToProve) == 0 {
		panic("path cannot be empty")
	}

	if len(paths) < 2 {
		panic("paths must have at least two elements")
	}
	var allProofs channeltypes.MsgConsStateProofs
	srcEnd := paths.A()

	// generate proof for key path on the source chain
	{
		// srcEnd.counterparty's proven height on its next connected chain
		provenHeight := srcEnd.Counterparty.GetClientState().GetLatestHeight()
		proof, _ := srcEnd.Chain.QueryProofAtHeight([]byte(keyPathToProve), int64(provenHeight.GetRevisionHeight()))

		var proofKV commitmenttypes.MerkleProof
		if err := srcEnd.Chain.Codec.Unmarshal(proof, &proofKV); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proof: %w", err)
		}
		prefixedKey, err := commitmenttypes.ApplyPrefix(
			srcEnd.Chain.GetPrefix(),
			commitmenttypes.NewMerklePath(keyPathToProve),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to apply prefix to key path: %w", err)
		}
		allProofs.Proofs = append(allProofs.Proofs, &channeltypes.ConsStateProof{
			Proof: &proofKV,
			// state is the same as its consState
			PrefixedKey: &prefixedKey,
		})
	}

	consStateProofs, err := GenerateMultiHopConsensusProof(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to generate consensus proofs: %w", err)
	}
	allProofs.Proofs = append(allProofs.Proofs, consStateProofs.Proofs...)

	return &allProofs, nil
}

// GenerateMultiHopConsensusProof generates a proof of consensus state of paths[0].EndpointA verified on
// paths[len(paths)-1].EndpointB and all intermediate consensus states.
func GenerateMultiHopConsensusProof(paths []*Path) (*channeltypes.MsgConsStateProofs, error) {
	if len(paths) < 2 {
		panic("paths must have at least two elements")
	}
	var consStateProofs channeltypes.MsgConsStateProofs

	// iterate all but the last path
	for i := 0; i < len(paths)-1; i++ {
		path, nextPath := paths[i], paths[i+1]
		// self is where the proof is queried and generated
		self := path.EndpointB

		heightAB := path.EndpointB.GetClientState().GetLatestHeight()
		heightBC := nextPath.EndpointB.GetClientState().GetLatestHeight()
		consStateAB, found := self.Chain.GetConsensusState(self.ClientID, heightAB)
		if !found {
			return nil, fmt.Errorf(
				"consensus state not found for height %s on chain %s",
				heightAB,
				self.Chain.ChainID,
			)
		}

		keyPrefixedConsAB, err := GetConsensusStatePrefix(self, heightAB)
		if err != nil {
			return nil, fmt.Errorf("failed to get consensus state prefix at height %d and revision %d: %w", heightAB.GetRevisionHeight(), heightAB.GetRevisionHeight(), err)
		}
		proofConsAB, err := GetConsStateProof(self, heightBC, heightAB, self.ClientID)
		if err != nil {
			return nil, fmt.Errorf("failed to get proof for consensus state on chain %s: %w", self.Chain.ChainID, err)
		}
		consStateABBytes, err := self.Chain.Codec.MarshalInterface(consStateAB)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal consensus state: %w", err)
		}
		// ensure consStateAB is verified by consStateBC, where self is chain B
		if err := proofConsAB.VerifyMembership(
			GetProofSpec(self),
			nextPath.EndpointB.GetConsensusState(heightBC).GetRoot(),
			keyPrefixedConsAB,
			consStateABBytes,
		); err != nil {
			return nil, fmt.Errorf(
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
		consensusStateWithHeight := types.NewConsensusStateWithHeight(types.NewHeight(heightAB.GetRevisionNumber(), heightAB.GetRevisionHeight()),
			consStateAB)
		consStateProofs.Proofs = append(consStateProofs.Proofs, &channeltypes.ConsStateProof{
			Proof:          &proofConsAB,
			ConsensusState: &consensusStateWithHeight,
			PrefixedKey:    &keyPrefixedConsAB,
		})
	}
	return &consStateProofs, nil
}

// VerifyMultiHopConsensusStateProof verifies the consensus state of paths[0].EndpointA on paths[len(paths)-1].EndpointB.
func VerifyMultiHopConsensusStateProof(endpoint *Endpoint, proofs []*channeltypes.ConsStateProof) error {
	lastConsstate := endpoint.GetConsensusState(endpoint.GetClientState().GetLatestHeight())
	for i := len(proofs) - 1; i >= 0; i-- {
		consStateProof := proofs[i]
		consStateBz, err := endpoint.Chain.Codec.MarshalInterface(consStateProof.ConsensusState)
		if err != nil {
			return fmt.Errorf("failed to marshal consensus state: %w", err)
		}
		if err = consStateProof.Proof.VerifyMembership(
			GetProofSpec(endpoint),
			lastConsstate.GetRoot(),
			consStateProof.PrefixedKey,
			consStateBz,
		); err != nil {
			return fmt.Errorf("failed to verify proof on chain '%s': %w", endpoint.Chain.ChainID, err)
		}
		if err = endpoint.Chain.Codec.UnpackAny(consStateProof.ConsensusState.ConsensusState, &lastConsstate); err != nil {
			return fmt.Errorf("failed to unpack consesnsus state: %w", err)
		}
	}
	return nil
}

// VerifyMultiHopProofMembership verifies a multihop membership proof including all intermediate state proofs.
func VerifyMultiHopProofMembership(endpoint *Endpoint, proofs *channeltypes.MsgConsStateProofs, value []byte) error {
	if len(proofs.Proofs) < 2 {
		return fmt.Errorf(
			"proof must have at least two elements where the first one is the proof for the key and the rest are for the consensus states",
		)
	}
	if err := VerifyMultiHopConsensusStateProof(endpoint, proofs.Proofs[1:]); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}
	keyValueProof := proofs.Proofs[0]
	var secondConsState exported.ConsensusState
	if err := endpoint.Chain.Codec.UnpackAny(proofs.Proofs[1].ConsensusState.ConsensusState, &secondConsState); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	return keyValueProof.Proof.VerifyMembership(
		GetProofSpec(endpoint),
		secondConsState.GetRoot(),
		keyValueProof.PrefixedKey,
		value,
	)
}

// VerifyMultiHopProofNonMembership verifies a multihop proof of non-membership including all intermediate state proofs.
func VerifyMultiHopProofNonMembership(endpoint *Endpoint, proofs *channeltypes.MsgConsStateProofs) error {
	if len(proofs.Proofs) < 2 {
		return fmt.Errorf(
			"proof must have at least two elements where the first one is the proof for the key and the rest are for the consensus states",
		)
	}
	if err := VerifyMultiHopConsensusStateProof(endpoint, proofs.Proofs[1:]); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}
	keyValueProof := proofs.Proofs[0]
	var secondConsState exported.ConsensusState
	if err := endpoint.Chain.Codec.UnpackAny(proofs.Proofs[1].ConsensusState.ConsensusState, &secondConsState); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	err := keyValueProof.Proof.VerifyNonMembership(
		GetProofSpec(endpoint),
		secondConsState.GetRoot(),
		keyValueProof.PrefixedKey,
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
) (merkleProof commitmenttypes.MerkleProof, err error) {
	consensusKey := host.FullConsensusStateKey(clientID, consensusHeight)
	proof, _ := self.Chain.QueryProofAtHeight(consensusKey, int64(selfHeight.GetRevisionHeight()))
	err = self.Chain.Codec.Unmarshal(proof, &merkleProof)
	return
}

// GetConsensusStatePrefix returns the merkle prefix of consensus state of self's counterparty chain at height `consensusHeight` stored on self.
func GetConsensusStatePrefix(self *Endpoint, consensusHeight exported.Height) (commitmenttypes.MerklePath, error) {
	keyPath := commitmenttypes.NewMerklePath(
		host.FullConsensusStatePath(self.ClientID, consensusHeight),
	)
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
		path := paths[len(paths)-1-i]
		path.EndpointA.UpdateClient()
		path.EndpointA, path.EndpointB = path.EndpointB, path.EndpointA
		reversed = append(reversed, path)
	}
	return reversed
}
