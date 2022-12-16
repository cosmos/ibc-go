package ibctesting

import (
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// GenerateMultiHopProof generate a proof for key path on the source (aka. paths[0].EndpointA) verified on the dest chain (aka.
// paths[len(paths)-1].EndpointB) and all intermediate consensus states.
//
// The first proof can be either a membership proof or a non-membership proof depending on if the key exists on the
// source chain.
// TODO: pass in proof height of key/value pair on A
func GenerateMultiHopProof(paths LinkedPaths, keyPathToProve []byte, expectedVal []byte) (proofs *channeltypes.MsgMultihopProofs, err error) {
	if len(keyPathToProve) == 0 {
		panic("path cannot be empty")
	}

	if len(paths) < 2 {
		panic("paths must have at least two elements")
	}
	endpointA := paths.A()

	proofs = &channeltypes.MsgMultihopProofs{}
	// generate proof for key path on the source chain
	{
		endpointB := endpointA.Counterparty
		heightBC := endpointB.GetClientState().GetLatestHeight()
		// srcEnd.counterparty's proven height on its next connected chain
		provenHeight := endpointB.GetClientState().GetLatestHeight()
		proof, _ := endpointA.Chain.QueryProofAtHeight([]byte(keyPathToProve), int64(provenHeight.GetRevisionHeight()))
		var proofKV commitmenttypes.MerkleProof
		if err = endpointA.Chain.Codec.Unmarshal(proof, &proofKV); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proof: %w", err)
		}

		prefixedKey, err := commitmenttypes.ApplyPrefix(
			endpointA.Chain.GetPrefix(),
			commitmenttypes.NewMerklePath(string(keyPathToProve)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to apply prefix to key path: %w", err)
		}

		// membership proof
		if len(expectedVal) > 0 {
			// check expected val
			if err = proofKV.VerifyMembership(
				commitmenttypes.GetSDKSpecs(),
				endpointB.GetConsensusState(heightBC).GetRoot(),
				prefixedKey,
				expectedVal,
			); err != nil {
				return nil, fmt.Errorf(
					"failed to verify keyval proof of [%s] on [%s] with [%s].ConsState on [%s]: %w\nconsider update [%s]'s client on [%s]",
					endpointB.Chain.ChainID,
					endpointA.Chain.ChainID,
					endpointA.Chain.ChainID,
					endpointB.Chain.ChainID,
					err,
					endpointA.Chain.ChainID,
					endpointB.Chain.ChainID,
				)
			}
		}
		// TODO: verify non-membership proof?

		proofs.KeyProof = &channeltypes.MultihopProof{
			Proof:       proof,
			Value:       nil,
			PrefixedKey: &prefixedKey,
		}
	}

	proofs.ConsensusProofs, proofs.ConnectionProofs, err = GenerateMultiHopConsensusProof(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to generate consensus proofs: %w", err)
	}

	return
}

// GenerateMultiHopConsensusProof generates a proof of consensus state of paths[0].EndpointA verified on
// paths[len(paths)-1].EndpointB and all intermediate consensus states.
// TODO: Would it be beneficial to batch the consensus state and connection proofs?
func GenerateMultiHopConsensusProof(
	paths []*Path,
) ([]*channeltypes.MultihopProof, []*channeltypes.MultihopProof, error) {
	if len(paths) < 2 {
		panic("paths must have at least two elements")
	}

	var consStateProofs []*channeltypes.MultihopProof
	var connectionProofs []*channeltypes.MultihopProof

	// iterate all but the last path
	for i := 0; i < len(paths)-1; i++ {
		path, nextPath := paths[i], paths[i+1]

		self := path.EndpointB // self is where the proof is queried and generated
		next := nextPath.EndpointB

		heightAB := self.GetClientState().GetLatestHeight() // height of A on B
		heightBC := next.GetClientState().GetLatestHeight() // height of B on C

		// consensus state of A on B at height AB which is the height of A's client state on B
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
			return nil, nil, fmt.Errorf(
				"failed to get consensus state prefix at height %d and revision %d: %w",
				heightAB.GetRevisionHeight(),
				heightAB.GetRevisionHeight(),
				err,
			)
		}

		// proof of A's consensus state (heightAB) on B at height BC
		consensusProof, _ := GetConsStateProof(self, heightBC, heightAB, self.ClientID)

		var consensusStateMerkleProof commitmenttypes.MerkleProof
		if err := self.Chain.Codec.Unmarshal(consensusProof, &consensusStateMerkleProof); err != nil {
			return nil, nil, fmt.Errorf(
				"failed to get proof for consensus state on chain %s: %w",
				self.Chain.ChainID,
				err,
			)
		}

		value, err := self.Chain.Codec.MarshalInterface(consStateAB)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal consensus state: %w", err)
		}

		// ensure consStateAB is verified by consStateBC, where self is chain B
		if err := consensusStateMerkleProof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			next.GetConsensusState(heightBC).GetRoot(),
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
			return nil, nil, fmt.Errorf(
				"failed to get proof for consensus state on chain %s: %w",
				self.Chain.ChainID,
				err,
			)
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
			commitmenttypes.GetSDKSpecs(),
			next.GetConsensusState(heightBC).GetRoot(),
			connectionKey,
			value,
		); err != nil {
			return nil, nil, fmt.Errorf(
				"failed to verify connection proof of [%s] on [%s] with [%s].ConnectionEnd on [%s]: %w\nconsider update [%s]'s client on [%s]",
				self.Chain.ChainID,
				self.Chain.ChainID,
				self.Chain.ChainID,
				next.Chain.ChainID,
				err,
				self.Chain.ChainID,
				next.Chain.ChainID,
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
func VerifyMultiHopConsensusStateProof(
	endpoint *Endpoint,
	consensusProofs []*channeltypes.MultihopProof,
	connectionProofs []*channeltypes.MultihopProof,
) error {
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
			commitmenttypes.GetSDKSpecs(),
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
			commitmenttypes.GetSDKSpecs(),
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
func VerifyMultiHopProofMembership(
	endpoint *Endpoint,
	proofs *channeltypes.MsgMultihopProofs,
	expectedVal []byte,
) error {
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
		commitmenttypes.GetSDKSpecs(),
		secondConsState.GetRoot(),
		*proofs.KeyProof.PrefixedKey,
		expectedVal,
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
		commitmenttypes.GetSDKSpecs(),
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
	connectionKey := host.ConnectionKey(connectionID)
	return self.Chain.QueryProofAtHeight(connectionKey, int64(selfHeight.GetRevisionHeight()))
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
