package ibctesting

import (
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// TODO: Extract proof generation so it only depends on minimal params
// GenerateMultiHopProof generate a proof for key path on the source (aka. paths[0].EndpointA) verified on the dest chain (aka.
// paths[len(paths)-1].EndpointB) and all intermediate consensus states.
//
// The first proof can be either a membership proof or a non-membership proof depending on if the key exists on the
// source chain.
// TODO: pass in proof height of key/value pair on A
func GenerateMultiHopProof(paths LinkedPaths, keyPathToProve []byte, expectedVal []byte, doVerify bool) (proofs *channeltypes.MsgMultihopProofs, err error) {
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
		proof, _ := endpointA.Chain.QueryProofAtHeight(keyPathToProve, int64(provenHeight.GetRevisionHeight()))

		// membership proof
		if doVerify {
			prefixedKey, err := commitmenttypes.ApplyPrefix(
				endpointA.Chain.GetPrefix(),
				commitmenttypes.NewMerklePath(string(keyPathToProve)),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to apply prefix to key path: %w", err)
			}
			if len(expectedVal) > 0 {
				var proofKV commitmenttypes.MerkleProof
				if err = endpointA.Chain.Codec.Unmarshal(proof, &proofKV); err != nil {
					return nil, fmt.Errorf("failed to unmarshal proof: %w", err)
				}
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
		}

		proofs.KeyProof = &channeltypes.MultihopProof{
			Proof: proof,
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

		keyPrefixedConsAB, err := getConsensusStatePrefix(self, heightAB)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"failed to get consensus state prefix at height %d and revision %d: %w",
				heightAB.GetRevisionHeight(),
				heightAB.GetRevisionHeight(),
				err,
			)
		}

		// proof of A's consensus state (heightAB) on B at height BC
		consensusProof, _ := getConsStateProof(self, heightBC, heightAB, self.ClientID)

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
		connectionKey, err := getPrefixedConnectionKey(self)
		if err != nil {
			return nil, nil, err
		}

		connectionProof, _ := getConnectionProof(self, heightBC, self.ConnectionID)
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

// getConsStateProof returns the merkle proof of consensusState of self's clientId and at `consensusHeight` stored on self at `selfHeight`.
func getConsStateProof(
	self *Endpoint,
	selfHeight exported.Height,
	consensusHeight exported.Height,
	clientID string,
) ([]byte, exported.Height) {
	consensusKey := host.FullConsensusStateKey(clientID, consensusHeight)
	return self.Chain.QueryProofAtHeight(consensusKey, int64(selfHeight.GetRevisionHeight()))
}

// getConnectionProof returns the proof of a connection at the specified height
func getConnectionProof(
	self *Endpoint,
	selfHeight exported.Height,
	connectionID string,
) ([]byte, exported.Height) {
	connectionKey := host.ConnectionKey(connectionID)
	return self.Chain.QueryProofAtHeight(connectionKey, int64(selfHeight.GetRevisionHeight()))
}

// getConsensusStatePrefix returns the merkle prefix of consensus state of self's counterparty chain at height `consensusHeight` stored on self.
func getConsensusStatePrefix(self *Endpoint, consensusHeight exported.Height) (commitmenttypes.MerklePath, error) {
	keyPath := commitmenttypes.NewMerklePath(host.FullConsensusStatePath(self.ClientID, consensusHeight))
	return commitmenttypes.ApplyPrefix(self.Chain.GetPrefix(), keyPath)
}

// getPrefixedConnectionKey returns the connection prefix associated
func getPrefixedConnectionKey(self *Endpoint) (commitmenttypes.MerklePath, error) {
	keyPath := commitmenttypes.NewMerklePath(host.ConnectionPath(self.ConnectionID))
	return commitmenttypes.ApplyPrefix(self.Chain.GetPrefix(), keyPath)
}
