package multihop

import (
	"fmt"
	"math"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
)

// VerifyDelayPeriodPassed will ensure that at least delayTimePeriod amount of time and delayBlockPeriod number of blocks have passed
// since consensus state was submitted before allowing verification to continue.
func VerifyDelayPeriodPassed(
	ctx sdk.Context,
	store sdk.KVStore,
	proofHeight exported.Height,
	timeDelay uint64,
	expectedTimePerBlock uint64,
) error {
	// get time and block delays
	blockDelay := getBlockDelay(ctx, timeDelay, expectedTimePerBlock)
	return tmclient.VerifyDelayPeriodPassed(ctx, store, proofHeight, timeDelay, blockDelay)
}

// getBlockDelay calculates the block delay period from the time delay of the connection
// and the maximum expected time per block.
func getBlockDelay(ctx sdk.Context, timeDelay uint64, expectedTimePerBlock uint64) uint64 {
	// expectedTimePerBlock should never be zero, however if it is then return a 0 block delay for safety
	// as the expectedTimePerBlock parameter was not set.
	if expectedTimePerBlock == 0 {
		return 0
	}
	return uint64(math.Ceil(float64(timeDelay) / float64(expectedTimePerBlock)))
}

// VerifyMultihopProof verifies a multihop proof. A nil value indicates a non-inclusion proof (proof of absence).
func VerifyMultihopProof(
	cdc codec.BinaryCodec,
	consensusState exported.ConsensusState,
	connectionHops []string,
	proofs *channeltypes.MsgMultihopProofs,
	prefix exported.Prefix,
	key string,
	value []byte,
) error {

	// verify proof lengths
	if len(proofs.ConnectionProofs) < 1 || len(proofs.ConsensusProofs) < 1 || len(proofs.ClientProofs) < 1 {
		return fmt.Errorf("the number of connection (%d), consensus (%d), and client (%d) proofs must be > 0",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs), len(proofs.ClientProofs))
	}

	if len(proofs.ConsensusProofs) != len(proofs.ConnectionProofs) {
		return fmt.Errorf("the number of connection (%d) and consensus (%d) proofs must be equal",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}

	if len(proofs.ConsensusProofs) != len(proofs.ClientProofs) {
		return fmt.Errorf("the number of client (%d) and consensus (%d) proofs must be equal",
			len(proofs.ClientProofs), len(proofs.ConsensusProofs))
	}

	// verify connection states and ordering
	if err := verifyConnectionStates(cdc, proofs.ConnectionProofs, connectionHops); err != nil {
		return err
	}

	// verify client states are not frozen
	if err := verifyClientStates(cdc, proofs.ClientProofs, proofs.ConsensusProofs); err != nil {
		return err
	}

	// verify intermediate consensus and connection states from destination --> source
	if err := verifyIntermediateStateProofs(cdc, consensusState, proofs.ConsensusProofs, proofs.ConnectionProofs, proofs.ClientProofs); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}

	// verify the keyproof on source chain's consensus state.
	return verifyKeyValueProof(cdc, consensusState, proofs, prefix, key, value)
}

// verifyConnectionState verifies that the provided connections match the connectionHops field of the channel and are in OPEN state
func verifyConnectionStates(cdc codec.BinaryCodec, connectionProofData []*channeltypes.MultihopProof, connectionHops []string) error {
	if len(connectionProofData) != len(connectionHops)-1 {
		return sdkerrors.Wrapf(connectiontypes.ErrInvalidLengthConnection,
			"connectionHops length (%d) must match the connectionProofData length (%d)",
			len(connectionHops)-1, len(connectionProofData))
	}

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

// verifyClientStates verifies that the provided clientstates are not frozen/expired
// and that the client id for the client state matches the consensus state.
func verifyClientStates(
	cdc codec.BinaryCodec,
	clientProofData []*channeltypes.MultihopProof,
	consensusProofData []*channeltypes.MultihopProof,
) error {
	for i, data := range clientProofData {
		var clientState exported.ClientState
		if err := cdc.UnmarshalInterface(data.Value, &clientState); err != nil {
			return fmt.Errorf("failed to unpack client state: %w", err)
		}
		var consensusState exported.ConsensusState
		if err := cdc.UnmarshalInterface(consensusProofData[i].Value, &consensusState); err != nil {
			return fmt.Errorf("failed to unpack consesnsus state: %w", err)
		}
		if len(consensusProofData[i].PrefixedKey.KeyPath) < 2 || len(clientProofData[i].PrefixedKey.KeyPath) < 2 {
			return fmt.Errorf("consensus and client proof prefixe length must be > 1")
		}
		consensusParts := strings.Split(consensusProofData[i].PrefixedKey.KeyPath[1], "/")
		clientParts := strings.Split(clientProofData[i].PrefixedKey.KeyPath[1], "/")
		if len(consensusParts) < 2 || len(clientParts) < 2 {
			return fmt.Errorf("consensus or client proof prefix component too short")
		}

		// verify the client ids match
		if consensusParts[1] != clientParts[1] {
			return fmt.Errorf("consensus (%s) and client (%s) ids must match", consensusParts[1], clientParts[1])
		}

		// clients can not be frozen
		if clientState.ClientType() == exported.Tendermint {
			cs, ok := clientState.(*tmclient.ClientState)
			if ok && cs.FrozenHeight != clienttypes.Height(types.NewHeight(0, 0)) {
				return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "Multihop client frozen")
			}
		}
	}
	return nil
}

// verifyIntermediateStateProofs verifies the intermediate consensus, connection, client states in the multi-hop proof.
func verifyIntermediateStateProofs(
	cdc codec.BinaryCodec,
	consensusState exported.ConsensusState,
	consensusProofs []*channeltypes.MultihopProof,
	connectionProofs []*channeltypes.MultihopProof,
	clientProofs []*channeltypes.MultihopProof,
) error {
	var consState exported.ConsensusState
	for i := len(consensusProofs) - 1; i >= 0; i-- {
		consensusProof := consensusProofs[i]
		connectionProof := connectionProofs[i]
		clientProof := clientProofs[i]
		if err := cdc.UnmarshalInterface(consensusProof.Value, &consState); err != nil {
			return fmt.Errorf("failed to unpack consesnsus state: %w", err)
		}

		// prove consensus state
		var proof commitmenttypes.MerkleProof
		if err := cdc.Unmarshal(consensusProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal consensus state proof: %w", err)
		}

		if err := proof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			consensusState.GetRoot(),
			*consensusProof.PrefixedKey,
			consensusProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify consensus proof: %w", err)
		}

		// prove connection state
		proof.Reset()
		if err := cdc.Unmarshal(connectionProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal connection state proof: %w", err)
		}

		if err := proof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			consensusState.GetRoot(),
			*connectionProof.PrefixedKey,
			connectionProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify connection proof: %w", err)
		}

		// prove client state
		proof.Reset()
		if err := cdc.Unmarshal(clientProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal cilent state proof: %w", err)
		}

		if err := proof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			consensusState.GetRoot(),
			*clientProof.PrefixedKey,
			clientProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify client proof: %w", err)
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
