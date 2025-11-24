package attestations

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	keyProcessedTime   = []byte("/processedTime")
	keyProcessedHeight = []byte("/processedHeight")
)

// getClientState retrieves the client state from the store using the provided KVStore and codec.
// It returns the unmarshaled ClientState and a boolean indicating if the state was found.
func getClientState(store storetypes.KVStore, cdc codec.BinaryCodec) (*ClientState, bool) {
	bz := store.Get(host.ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}

	clientStateI := clienttypes.MustUnmarshalClientState(cdc, bz)
	var clientState *ClientState
	clientState, ok := clientStateI.(*ClientState)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to %T", clientStateI, clientState))
	}

	return clientState, true
}

// setConsensusState stores the consensus state at the given height.
func setConsensusState(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, consensusState *ConsensusState, height exported.Height) {
	key := host.ConsensusStateKey(height)
	val := clienttypes.MustMarshalConsensusState(cdc, consensusState)
	clientStore.Set(key, val)

	setConsensusMetadata(ctx, clientStore, height)
}

// getConsensusState retrieves the consensus state from the client prefixed store.
// If the ConsensusState does not exist in state for the provided height a nil value and false boolean flag is returned
func getConsensusState(store storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height) (*ConsensusState, bool) {
	bz := store.Get(host.ConsensusStateKey(height))
	if len(bz) == 0 {
		return nil, false
	}

	consensusStateI := clienttypes.MustUnmarshalConsensusState(cdc, bz)
	var consensusState *ConsensusState
	consensusState, ok := consensusStateI.(*ConsensusState)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to %T", consensusStateI, consensusState))
	}

	return consensusState, true
}

func processedTimeKey(height exported.Height) []byte {
	return append(host.ConsensusStateKey(height), keyProcessedTime...)
}

func processedHeightKey(height exported.Height) []byte {
	return append(host.ConsensusStateKey(height), keyProcessedHeight...)
}

func setProcessedTime(clientStore storetypes.KVStore, height exported.Height, timeNs uint64) {
	clientStore.Set(processedTimeKey(height), sdk.Uint64ToBigEndian(timeNs))
}

func getProcessedTime(clientStore storetypes.KVStore, height exported.Height) (uint64, bool) {
	bz := clientStore.Get(processedTimeKey(height))
	if len(bz) == 0 {
		return 0, false
	}

	return sdk.BigEndianToUint64(bz), true
}

func setProcessedHeight(clientStore storetypes.KVStore, consHeight, processedHeight exported.Height) {
	clientStore.Set(processedHeightKey(consHeight), []byte(processedHeight.String()))
}

func getProcessedHeight(clientStore storetypes.KVStore, height exported.Height) (exported.Height, bool) {
	bz := clientStore.Get(processedHeightKey(height))
	if len(bz) == 0 {
		return nil, false
	}

	processedHeight, err := clienttypes.ParseHeight(string(bz))
	if err != nil {
		return nil, false
	}

	return processedHeight, true
}

func setConsensusMetadata(ctx sdk.Context, clientStore storetypes.KVStore, height exported.Height) {
	setProcessedTime(clientStore, height, uint64(ctx.BlockTime().UnixNano()))
	setProcessedHeight(clientStore, height, clienttypes.GetSelfHeight(ctx))
}
