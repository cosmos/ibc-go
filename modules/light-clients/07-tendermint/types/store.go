package types

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	proto "github.com/gogo/protobuf/proto"
)

// KeyProcessedTime is appended to consensus state key to store the processed time
// KeyProcessedHeight is appended to consensus state key to store the processed block height
var (
	KeyProcessedTime   = []byte("/processedTime")
	KeyProcessedHeight = []byte("/processedHeight")
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

// IterateProcessedHeight iterates through the prefix store and applies the callback.
// If the cb returns true, then iterator will close and stop.
func IterateProcessedHeight(store sdk.KVStore, cb func(key, val []byte) bool) {
	iterator := sdk.KVStorePrefixIterator(store, []byte(host.KeyConsensusStatePrefix))

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		// processed height key in prefix store has format: "consensusState/<height>/processedHeight"
		if len(keySplit) != 3 || keySplit[2] != "processedHeight" {
			// ignore all consensus state keys
			continue
		}

		if cb(iterator.Key(), iterator.Value()) {
			break
		}
	}
}

// ProcessedTime and ProcessedHeight Store code

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

// ProcessedHeightKey returns the key under which the processed height will be stored in the client store.
func ProcessedHeightKey(height exported.Height) []byte {
	return append(host.ConsensusStateKey(height), KeyProcessedHeight...)
}

// SetProcessedHeight stores the block height at which a header was processed and the corresponding consensus state was created.
// This is useful when validating whether a packet has reached the specified delay period in the tendermint client's
// verification functions
func SetProcessedHeight(clientStore sdk.KVStore, consHeight, processedHeight exported.Height) {
	key := ProcessedHeightKey(consHeight)
	valHeight, ok := processedHeight.(clienttypes.Height)
	if !ok {
		panic("could not set processed height")
	}
	val, err := proto.Marshal(&valHeight)
	if err != nil {
		panic("could not marshal processed height")
	}

	clientStore.Set(key, val)
}

// GetProcessedHeight gets the block height at which this chain received and processed a tendermint header.
// This is used to validate that a received packet has passed the delay period.
func GetProcessedHeight(clientStore sdk.KVStore, height exported.Height) (exported.Height, bool) {
	key := ProcessedHeightKey(height)
	bz := clientStore.Get(key)
	if bz == nil {
		return clienttypes.Height{}, false
	}
	var processedHeight clienttypes.Height
	proto.Unmarshal(bz, &processedHeight)
	return processedHeight, true
}
