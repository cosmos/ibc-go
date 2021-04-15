package types

import (
	"encoding/binary"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

const KeyIterateConsensusStatePrefix = "iterateConsensusStates"

var (
	// KeyProcessedTime is appended to consensus state key to store the processed time
	KeyProcessedTime = []byte("/processedTime")
	KeyIteration     = []byte("/iterationKey")
)

// SetConsensusState stores the consensus state at the given height.
func SetConsensusState(clientStore sdk.KVStore, cdc codec.BinaryMarshaler, consensusState *ConsensusState, height exported.Height) {
	key := host.ConsensusStateKey(height)
	val := clienttypes.MustMarshalConsensusState(cdc, consensusState)
	clientStore.Set(key, val)
}

// GetConsensusState retrieves the consensus state from the client prefixed
// store. An error is returned if the consensus state does not exist.
func GetConsensusState(store sdk.KVStore, cdc codec.BinaryMarshaler, height exported.Height) (*ConsensusState, error) {
	bz := store.Get(host.ConsensusStateKey(height))
	if bz == nil {
		return nil, sdkerrors.Wrapf(
			clienttypes.ErrConsensusStateNotFound,
			"consensus state does not exist for height %s", height,
		)
	}

	consensusStateI, err := clienttypes.UnmarshalConsensusState(cdc, bz)
	if err != nil {
		return nil, sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "unmarshal error: %v", err)
	}

	consensusState, ok := consensusStateI.(*ConsensusState)
	if !ok {
		return nil, sdkerrors.Wrapf(
			clienttypes.ErrInvalidConsensus,
			"invalid consensus type %T, expected %T", consensusState, &ConsensusState{},
		)
	}

	return consensusState, nil
}

// IterateProcessedTime iterates through the prefix store and applies the callback.
// If the cb returns true, then iterator will close and stop.
func IterateProcessedTime(store sdk.KVStore, cb func(key, val []byte) bool) {
	iterator := sdk.KVStorePrefixIterator(store, []byte(host.KeyConsensusStatePrefix))

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		// processed time key in prefix store has format: "consensusState/<height>/processedTime"
		if len(keySplit) != 3 || keySplit[2] != "processedTime" {
			// ignore all consensus state keys
			continue
		}

		if cb(iterator.Key(), iterator.Value()) {
			break
		}
	}
}

// ProcessedTime Store code

// ProcessedTimeKey returns the key under which the processed time will be stored in the client store.
func ProcessedTimeKey(height exported.Height) []byte {
	return append(host.ConsensusStateKey(height), KeyProcessedTime...)
}

// SetProcessedTime stores the time at which a header was processed and the corresponding consensus state was created.
// This is useful when validating whether a packet has reached the specified delay period in the tendermint client's
// verification functions
func SetProcessedTime(clientStore sdk.KVStore, height exported.Height, timeNs uint64) {
	key := ProcessedTimeKey(height)
	val := sdk.Uint64ToBigEndian(timeNs)
	clientStore.Set(key, val)
}

// GetProcessedTime gets the time (in nanoseconds) at which this chain received and processed a tendermint header.
// This is used to validate that a received packet has passed the delay period.
func GetProcessedTime(clientStore sdk.KVStore, height exported.Height) (uint64, bool) {
	key := ProcessedTimeKey(height)
	bz := clientStore.Get(key)
	if bz == nil {
		return 0, false
	}
	return sdk.BigEndianToUint64(bz), true
}

// Iteration Code

// IterationKey returns the key under which the consensus state key will be stored.
// The iteration key is a BigEndian representation of the consensus state key to support efficient iteration.
func IterationKey(height exported.Height) []byte {
	heightBytes := bigEndianHeightBytes(height)
	return append([]byte(KeyIterateConsensusStatePrefix), heightBytes...)
}

// SetIterationKey stores the consensus state key under a key that is more efficient for ordered iteration
func SetIterationKey(clientStore sdk.KVStore, height exported.Height) {
	key := IterationKey(height)
	val := host.ConsensusStateKey(height)
	clientStore.Set(key, val)
}

func IterateConsensusStateAscending(clientStore sdk.KVStore, cdc codec.BinaryMarshaler,
	cb func(cs ConsensusState) (stop bool)) error {

	iterator := sdk.KVStorePrefixIterator(clientStore, []byte(KeyIterateConsensusStatePrefix))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		csKey := iterator.Value()
		bz := clientStore.Get(csKey)

		consensusStateI, err := clienttypes.UnmarshalConsensusState(cdc, bz)
		if err != nil {
			return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "unmarshal error: %v", err)
		}

		consensusState, ok := consensusStateI.(*ConsensusState)
		if !ok {
			return sdkerrors.Wrapf(
				clienttypes.ErrInvalidConsensus,
				"invalid consensus type %T, expected %T", consensusState, &ConsensusState{},
			)
		}

		if cb(*consensusState) {
			break
		}
	}
	return nil
}

func IterateConsensusStateDescending(clientStore sdk.KVStore, cdc codec.BinaryMarshaler,
	cb func(cs ConsensusState) (stop bool)) error {

	iterator := sdk.KVStorePrefixIterator(clientStore, []byte(KeyIterateConsensusStatePrefix))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		csKey := iterator.Value()
		bz := clientStore.Get(csKey)

		consensusStateI, err := clienttypes.UnmarshalConsensusState(cdc, bz)
		if err != nil {
			return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "unmarshal error: %v", err)
		}

		consensusState, ok := consensusStateI.(*ConsensusState)
		if !ok {
			return sdkerrors.Wrapf(
				clienttypes.ErrInvalidConsensus,
				"invalid consensus type %T, expected %T", consensusState, &ConsensusState{},
			)
		}

		if cb(*consensusState) {
			break
		}
	}
	return nil
}

func GetNextConsensusState(clientStore sdk.KVStore, cdc codec.BinaryMarshaler, height exported.Height) (*ConsensusState, bool) {
	iterateStore := prefix.NewStore(clientStore, []byte(KeyIterateConsensusStatePrefix))
	iterator := iterateStore.Iterator(bigEndianHeightBytes(height), nil)
	defer iterator.Close()
	// ignore the consensus state at current height and get next height
	iterator.Next()
	if !iterator.Valid() {
		return nil, false
	}

	csKey := iterator.Value()
	bz := clientStore.Get(csKey)
	if bz == nil {
		return nil, false
	}

	consensusStateI, err := clienttypes.UnmarshalConsensusState(cdc, bz)
	if err != nil {
		return nil, false
	}

	consensusState, ok := consensusStateI.(*ConsensusState)
	if !ok {
		return nil, false
	}

	return consensusState, true
}

func GetPreviousConsensusState(clientStore sdk.KVStore, cdc codec.BinaryMarshaler, height exported.Height) (*ConsensusState, bool) {
	iterateStore := prefix.NewStore(clientStore, []byte(KeyIterateConsensusStatePrefix))
	iterator := iterateStore.ReverseIterator(nil, bigEndianHeightBytes(height))
	defer iterator.Close()

	if !iterator.Valid() {
		return nil, false
	}

	csKey := iterator.Value()
	bz := clientStore.Get(csKey)
	if bz == nil {
		return nil, false
	}

	consensusStateI, err := clienttypes.UnmarshalConsensusState(cdc, bz)
	if err != nil {
		return nil, false
	}

	consensusState, ok := consensusStateI.(*ConsensusState)
	if !ok {
		return nil, false
	}

	return consensusState, true
}

func bigEndianHeightBytes(height exported.Height) []byte {
	revisionBytes := make([]byte, 8)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(revisionBytes, height.GetRevisionNumber())
	binary.BigEndian.PutUint64(heightBytes, height.GetRevisionHeight())
	return append(revisionBytes, heightBytes...)
}
