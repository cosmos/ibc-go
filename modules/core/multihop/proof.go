package multihop

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint/types"
	//ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

// ConsStateProof includes data necessary for verifying that A's consensus state on B is proven by B's
// consensus state on C given chains A-B-C. The proof is queried from chain B, and the state represents
// chain A's consensus state on B. The `prefixedKey` is the key of the A's consensus state on chain B.
type ConsStateProof struct {
	Proof       commitmenttypes.MerkleProof
	State       exported.ConsensusState
	PrefixedKey commitmenttypes.MerklePath
}

// VerifyMultiHopConsensusStateProofv2 verifies the consensus state of paths[0].EndpointA on paths[len(paths)-1].EndpointB.
func VerifyMultiHopConsensusStateProof(consensusState exported.ConsensusState, clientState exported.ClientState, cdc codec.BinaryCodec, proofs []*ConsStateProof) error {
	tmclient := clientState.(*ibctmtypes.ClientState)

	for i := len(proofs) - 1; i >= 0; i-- {
		consStateProof := proofs[i]
		consStateBz, err := cdc.MarshalInterface(consStateProof.State)
		if err != nil {
			return fmt.Errorf("failed to marshal consensus state: %w", err)
		}

		if err = consStateProof.Proof.VerifyMembership(
			tmclient.GetProofSpecs(),
			consensusState.GetRoot(),
			consStateProof.PrefixedKey,
			consStateBz,
		); err != nil {
			return fmt.Errorf("failed to verify proof: %w", err)
		}
		consensusState = consStateProof.State
	}
	return nil
}

// VerifyMultiHopProofMembershipv2 verifies a multihop membership proof including all intermediate state proofs.
func VerifyMultiHopProofMembership(consensusState exported.ConsensusState, clientState exported.ClientState, cdc codec.BinaryCodec, proofs []*ConsStateProof, value []byte) error {
	if len(proofs) < 2 {
		return fmt.Errorf(
			"proof must have at least two elements where the first one is the proof for the key and the rest are for the consensus states",
		)
	}
	if err := VerifyMultiHopConsensusStateProof(consensusState, clientState, cdc, proofs[1:]); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}
	keyValueProof := proofs[0]
	secondConsState := proofs[1].State
	tmclient := clientState.(*ibctmtypes.ClientState)
	err := keyValueProof.Proof.VerifyMembership(
		tmclient.GetProofSpecs(),
		secondConsState.GetRoot(),
		keyValueProof.PrefixedKey,
		value,
	)
	return err
}
