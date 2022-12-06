package multihop

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint/types"
)

// VerifyMultiHopConsensusStateProof verifies the consensus state of paths[0].EndpointA on paths[len(paths)-1].EndpointB.
func VerifyMultiHopConsensusStateProof(
	consensusState exported.ConsensusState,
	clientState exported.ClientState,
	cdc codec.BinaryCodec,
	proofs []*channeltypes.ConsStateProof,
) error {
	tmclient := clientState.(*ibctmtypes.ClientState)
	var consState exported.ConsensusState
	for i := len(proofs) - 1; i >= 0; i-- {
		consStateProof := proofs[i]
		if err := cdc.UnpackAny(consStateProof.ConsensusState.ConsensusState, &consState); err != nil {
			return fmt.Errorf("failed to unpack consesnsus state: %w", err)
		}
		consStateBz, err := cdc.MarshalInterface(consState)
		if err != nil {
			return fmt.Errorf("failed to marshal consensus state: %w", err)
		}

		if err = consStateProof.Proof.VerifyMembership(
			tmclient.GetProofSpecs(),
			consensusState.GetRoot(),
			*consStateProof.PrefixedKey,
			consStateBz,
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
	clientState exported.ClientState,
	cdc codec.BinaryCodec,
	proofs *channeltypes.MsgConsStateProofs,
	value []byte,
) error {
	if len(proofs.Proofs) < 2 {
		return fmt.Errorf(
			"proof must have at least two elements where the first one is the proof for the key and the rest are for the consensus states",
		)
	}
	if err := VerifyMultiHopConsensusStateProof(consensusState, clientState, cdc, proofs.Proofs[1:]); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}
	keyValueProof := proofs.Proofs[0]
	var secondConsState exported.ConsensusState
	if err := cdc.UnpackAny(proofs.Proofs[1].ConsensusState.ConsensusState, &secondConsState); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	tmclient := clientState.(*ibctmtypes.ClientState)
	fmt.Printf("secondConsState.root: %x\n", secondConsState.GetRoot().GetHash())
	fmt.Printf("key: %s\n", keyValueProof.PrefixedKey.String())
	fmt.Printf("val: %x\n", value)
	return keyValueProof.Proof.VerifyMembership(
		tmclient.GetProofSpecs(),
		secondConsState.GetRoot(),
		*keyValueProof.PrefixedKey,
		value,
	)
}
