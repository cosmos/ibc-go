package multihop

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// VerifyMultiHopConsensusStateProof verifies the consensus state of paths[0].EndpointA on paths[len(paths)-1].EndpointB.
func VerifyMultiHopConsensusStateProof(
	consensusState exported.ConsensusState,
	cdc codec.BinaryCodec,
	consensusProofs []*channeltypes.MultihopProof,
	connectionProofs []*channeltypes.MultihopProof,
) error {
	var consState exported.ConsensusState
	for i := len(consensusProofs) - 1; i >= 0; i-- {
		consStateProof := consensusProofs[i]
		connectionProof := connectionProofs[i]
		if err := cdc.UnmarshalInterface(consStateProof.Value, &consState); err != nil {
			return fmt.Errorf("failed to unpack consesnsus state: %w", err)
		}

		var proof commitmenttypes.MerkleProof
		if err := cdc.Unmarshal(consStateProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal consensus state proof: %w", err)
		}

		if err := proof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			consensusState.GetRoot(),
			*consStateProof.PrefixedKey,
			consStateProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify proof: %w", err)
		}

		proof.Reset()
		if err := cdc.Unmarshal(connectionProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal consensus state proof: %w", err)
		}

		if err := proof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			consensusState.GetRoot(),
			*connectionProof.PrefixedKey,
			connectionProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify proof: %w", err)
		}

		consensusState = consState
	}
	return nil
}

// VerifyMultiHopProofMembership verifies a multihop membership proof including all intermediate state proofs.
func VerifyMultiHopProofMembership(
	consensusState exported.ConsensusState,
	cdc codec.BinaryCodec,
	proofs *channeltypes.MsgMultihopProofs,
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
	if err := VerifyMultiHopConsensusStateProof(consensusState, cdc, proofs.ConsensusProofs, proofs.ConnectionProofs); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}
	var keyProof commitmenttypes.MerkleProof
	if err := cdc.Unmarshal(proofs.KeyProof.Proof, &keyProof); err != nil {
		return fmt.Errorf("failed to unmarshal key proof: %w", err)
	}
	var secondConsState exported.ConsensusState
	if err := cdc.UnmarshalInterface(proofs.ConsensusProofs[0].Value, &secondConsState); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	fmt.Printf("secondConsState.root: %x\n", secondConsState.GetRoot().GetHash())
	fmt.Printf("key: %s\n", proofs.KeyProof.PrefixedKey.String())
	fmt.Printf("val: %x\n", proofs.KeyProof.Value)
	return keyProof.VerifyMembership(
		commitmenttypes.GetSDKSpecs(),
		secondConsState.GetRoot(),
		*proofs.KeyProof.PrefixedKey,
		proofs.KeyProof.Value,
	)
}
