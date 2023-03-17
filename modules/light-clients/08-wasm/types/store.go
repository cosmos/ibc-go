package types

import (
	"bytes"
	"io"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/cachekv"
	"github.com/cosmos/cosmos-sdk/store/listenkv"
	"github.com/cosmos/cosmos-sdk/store/tracekv"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// WrappedStore combines two KVStores into one while transparently routing the calls based on key prefix
type WrappedStore struct {
	first  sdk.KVStore
	second sdk.KVStore

	firstPrefix  []byte
	secondPrefix []byte
}

func NewWrappedStore(first, second sdk.KVStore, firstPrefix, secondPrefix []byte) WrappedStore {
	return WrappedStore{
		first:        first,
		second:       second,
		firstPrefix:  firstPrefix,
		secondPrefix: secondPrefix,
	}
}

func (ws WrappedStore) Get(key []byte) []byte {
	return ws.getStore(key).Get(ws.trimPrefix(key))
}

func (ws WrappedStore) Has(key []byte) bool {
	return ws.getStore(key).Has(ws.trimPrefix(key))
}

func (ws WrappedStore) Set(key, value []byte) {
	ws.getStore(key).Set(ws.trimPrefix(key), value)
}

func (ws WrappedStore) Delete(key []byte) {
	ws.getStore(key).Delete(ws.trimPrefix(key))
}

func (ws WrappedStore) GetStoreType() storetypes.StoreType {
	return ws.first.GetStoreType()
}

func (ws WrappedStore) Iterator(start, end []byte) sdk.Iterator {
	return ws.getStore(start).Iterator(ws.trimPrefix(start), ws.trimPrefix(end))
}

func (ws WrappedStore) ReverseIterator(start, end []byte) sdk.Iterator {
	return ws.getStore(start).ReverseIterator(ws.trimPrefix(start), ws.trimPrefix(end))
}

func (ws WrappedStore) CacheWrap() storetypes.CacheWrap {
	return cachekv.NewStore(ws)
}

func (ws WrappedStore) CacheWrapWithTrace(w io.Writer, tc storetypes.TraceContext) storetypes.CacheWrap {
	return cachekv.NewStore(tracekv.NewStore(ws, w, tc))
}

func (ws WrappedStore) CacheWrapWithListeners(storeKey storetypes.StoreKey, listeners []storetypes.WriteListener) storetypes.CacheWrap {
	return cachekv.NewStore(listenkv.NewStore(ws, storeKey, listeners))
}

func (ws WrappedStore) trimPrefix(key []byte) []byte {
	if bytes.HasPrefix(key, ws.firstPrefix) {
		key = bytes.TrimPrefix(key, ws.firstPrefix)
	} else {
		key = bytes.TrimPrefix(key, ws.secondPrefix)
	}

	return key
}

func (ws WrappedStore) getStore(key []byte) sdk.KVStore {
	if bytes.HasPrefix(key, ws.firstPrefix) {
		return ws.first
	}

	return ws.second
}

// setClientState stores the client state
func setClientState(clientStore sdk.KVStore, cdc codec.BinaryCodec, clientState *ClientState) {
	key := host.ClientStateKey()
	val := clienttypes.MustMarshalClientState(cdc, clientState)
	clientStore.Set(key, val)
}

// setConsensusState stores the consensus state at the given height.
func setConsensusState(clientStore sdk.KVStore, cdc codec.BinaryCodec, consensusState *ConsensusState, height exported.Height) {
	key := host.ConsensusStateKey(height)
	val := clienttypes.MustMarshalConsensusState(cdc, consensusState)
	clientStore.Set(key, val)
}

// GetConsensusState retrieves the consensus state from the client prefixed
// store. An error is returned if the consensus state does not exist.
func GetConsensusState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height) (*ConsensusState, error) {
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

var _ wasmvmtypes.KVStore = &StoreAdapter{}

// StoreAdapter adapter to bridge SDK store impl to wasmvm
type StoreAdapter struct {
	parent sdk.KVStore
}

// NewStoreAdapter constructor
func NewStoreAdapter(s sdk.KVStore) *StoreAdapter {
	if s == nil {
		panic("store must not be nil")
	}
	return &StoreAdapter{parent: s}
}

func (s StoreAdapter) Get(key []byte) []byte {
	return s.parent.Get(key)
}

func (s StoreAdapter) Set(key, value []byte) {
	s.parent.Set(key, value)
}

func (s StoreAdapter) Delete(key []byte) {
	s.parent.Delete(key)
}

func (s StoreAdapter) Iterator(start, end []byte) wasmvmtypes.Iterator {
	return s.parent.Iterator(start, end)
}

func (s StoreAdapter) ReverseIterator(start, end []byte) wasmvmtypes.Iterator {
	return s.parent.ReverseIterator(start, end)
}
