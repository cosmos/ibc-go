package multihop

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	connectiontypes "github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
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
	value []byte,
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
		value,
	)
}

// GetExpectedCounterpartyChannelBytes returns a counterparty multihop channel as bytes for multihop proofs
// TODO: refactor this to avoid needing to unmarshal the multihop proof message twice (here and again in VerifyMultihopProof)
func GetExpectedCounterpartyChannelBytes(
	portID string,
	channelID string,
	state channeltypes.State,
	ordering channeltypes.Order,
	version string,
	cdc codec.BinaryCodec,
	lastConnection *connectiontypes.ConnectionEnd,
	proof []byte,
) ([]byte, error) {
	var proofs channeltypes.MsgMultihopProofs
	if err := cdc.Unmarshal(proof, &proofs); err != nil {
		return nil, err
	}
	var counterpartyHops []string

	for _, connData := range proofs.ConnectionProofs {
		var connectionEnd connectiontypes.ConnectionEnd
		if err := cdc.Unmarshal(connData.Value, &connectionEnd); err != nil {
			return nil, err
		}
		counterpartyHops = append(counterpartyHops, connectionEnd.GetCounterparty().GetConnectionID())
	}
	counterpartyHops = append(counterpartyHops, lastConnection.GetCounterparty().GetConnectionID())

	counterparty := channeltypes.NewCounterparty(portID, channelID)
	expectedChannel := channeltypes.NewChannel(
		state, ordering, counterparty,
		counterpartyHops, version,
	)
	value, err := expectedChannel.Marshal()
	if err != nil {
		return nil, err
	}
	return value, err
}

// VerifyMultihopProof verifies a multihop proof
func VerifyMultihopProof(cdc codec.BinaryCodec, consensusState exported.ConsensusState, connectionHops []string, proof []byte, value []byte) error {
	var proofs channeltypes.MsgMultihopProofs
	if err := cdc.Unmarshal(proof, &proofs); err != nil {
		return err
	}

	// check all connections are in OPEN state and that the connection IDs match and are in the right order
	for i, connData := range proofs.ConnectionProofs {
		var connectionEnd connectiontypes.ConnectionEnd
		if err := cdc.Unmarshal(connData.Value, &connectionEnd); err != nil {
			return err
		}

		// Verify the first N-1 connectionHops (last hop already verified above)
		// 1. check the connectionHop values match the proofs and are in the same order.
		parts := strings.Split(connData.PrefixedKey.GetKeyPath()[len(connData.PrefixedKey.KeyPath)-1], "/")

		// fmt.Printf("parts[len(parts)-1]: %s\n", parts[len(parts)-1])
		// fmt.Printf("channel.ConnectionHops[%d]: %s\n", i+1, channel.ConnectionHops[i+1])
		// fmt.Printf("connectionEnd.Counterparty.ConnectionId: %s\n", connectionEnd.Counterparty.ConnectionId)
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

	fmt.Printf("proof value check: %x\n", value)
	// verify each consensus state and connection state starting going from Z --> A
	// finally verify the keyproof on A within B's verified view of A's consensus state.
	return VerifyMultiHopProofMembership(consensusState, cdc, &proofs, value)
}
