package multihop

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// VerifyMultihopProof verifies a multihop proof. A nil value indicates a non-inclusion proof (proof of absence).
func VerifyMultihopProof(
	cdc codec.BinaryCodec,
	consensusState exported.ConsensusState,
	connectionHops []string,
	proof []byte,
	prefix exported.Prefix,
	key string,
	value []byte,
) error {
	var proofs channeltypes.MsgMultihopProofs
	if err := cdc.Unmarshal(proof, &proofs); err != nil {
		return err
	}

	// verify proof lengths
	if len(proofs.ConnectionProofs) < 1 || len(proofs.ConsensusProofs) < 1 {
		return fmt.Errorf("the number of connection (%d) consensus (%d) proofs must be > 0",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}

	if len(proofs.ConsensusProofs) != len(proofs.ConnectionProofs) {
		return fmt.Errorf("the number of connection (%d) and consensus (%d) proofs must be equal",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}

	// verify connection states and ordering
	if err := verifyConnectionStates(cdc, proofs.ConnectionProofs, connectionHops); err != nil {
		return err
	}

	// verify intermediate consensus and connection states from destination --> source
	if err := verifyConsensusAndConnectionStateProofs(consensusState, cdc, proofs.ConsensusProofs, proofs.ConnectionProofs); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}

	// verify the keyproof on source chain's consensus state.
	return verifyKeyValueProof(cdc, consensusState, &proofs, prefix, key, value)
}

// verifyConnectionState verifies that the provided connections match the connectionHops field of the channel and are in OPEN state
func verifyConnectionStates(cdc codec.BinaryCodec, connectionProofData []*channeltypes.MultihopProof, connectionHops []string) error {
	// check all connections are in OPEN state and that the connection IDs match and are in the right order
	for i, connData := range connectionProofData {
		var connectionEnd connectiontypes.ConnectionEnd
		if err := cdc.Unmarshal(connData.Value, &connectionEnd); err != nil {
			return err
		}

		// Verify the rest of the connectionHops (first hop already verified)
		// 1. check the connectionHop values match the proofs and are in the same order.
		parts := strings.Split(connData.PrefixedKey.GetKeyPath()[len(connData.PrefixedKey.KeyPath)-1], "/")
		if parts[len(parts)-1] != connectionHops[i+1] {
			return sdkerrors.Wrapf(
				connectiontypes.ErrConnectionPath,
				"connectionHops (%s) does not match connection proof hop (%s) for hop %d",
				connectionHops[i+1], parts[len(parts)-1], i)
		}

		// 2. check that the connectionEnd's are in the OPEN state.
		if connectionEnd.GetState() != int32(connectiontypes.OPEN) {
			return sdkerrors.Wrapf(
				connectiontypes.ErrInvalidConnectionState,
				"connection state is not OPEN for connectionID=%s (got %s)",
				connectionEnd.Counterparty.ConnectionId,
				connectiontypes.State(connectionEnd.GetState()).String(),
			)
		}
	}

	return nil
}

// verifyConsensusAndConnectionStateProofs verifies the consensus and connection states in the multi-hop proof.
func verifyConsensusAndConnectionStateProofs(
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
			connectionProof.Value, // this should be from connectionHops
		); err != nil {
			return fmt.Errorf("failed to verify proof: %w", err)
		}

		consensusState = consState
	}
	return nil
}

// verifyKeyValueProof verifies a multihop membership proof including all intermediate state proofs.
// If the value is "nil" then a proof of non-membership is verified.
func verifyKeyValueProof(
	cdc codec.BinaryCodec,
	consensusState exported.ConsensusState,
	proofs *channeltypes.MsgMultihopProofs,
	prefix exported.Prefix,
	key string,
	value []byte,
) error {
	prefixedKey, err := commitmenttypes.ApplyPrefix(prefix, commitmenttypes.NewMerklePath(key))
	if err != nil {
		return err
	}

	var keyProof commitmenttypes.MerkleProof
	if err := cdc.Unmarshal(proofs.KeyProof.Proof, &keyProof); err != nil {
		return fmt.Errorf("failed to unmarshal key proof: %w", err)
	}
	var secondConsState exported.ConsensusState
	if err := cdc.UnmarshalInterface(proofs.ConsensusProofs[0].Value, &secondConsState); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}

	if value == nil {
		return keyProof.VerifyNonMembership(
			commitmenttypes.GetSDKSpecs(),
			secondConsState.GetRoot(),
			prefixedKey,
		)
	} else {
		return keyProof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			secondConsState.GetRoot(),
			prefixedKey,
			value,
		)
	}
}
