package types

import (
	"bytes"
	"errors"
	"io"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	"github.com/cosmos/cosmos-sdk/store/cachekv"
	"github.com/cosmos/cosmos-sdk/store/listenkv"
	"github.com/cosmos/cosmos-sdk/store/tracekv"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// updateProposalWrappedStore combines two KVStores into one while transparently routing the calls based on key prefix
type updateProposalWrappedStore struct {
	subjectStore    sdk.KVStore
	substituteStore sdk.KVStore

	subjectPrefix    []byte
	substitutePrefix []byte
}

func newUpdateProposalWrappedStore(subjectStore, substituteStore sdk.KVStore, subjectPrefix, substitutePrefix []byte) updateProposalWrappedStore {
	return updateProposalWrappedStore{
		subjectStore:     subjectStore,
		substituteStore:  substituteStore,
		subjectPrefix:    subjectPrefix,
		substitutePrefix: substitutePrefix,
	}
}

func (ws updateProposalWrappedStore) Get(key []byte) []byte {
	return ws.getStore(key).Get(ws.trimPrefix(key))
}

func (ws updateProposalWrappedStore) Has(key []byte) bool {
	return ws.getStore(key).Has(ws.trimPrefix(key))
}

func (ws updateProposalWrappedStore) Set(key, value []byte) {
	ws.getStore(key).Set(ws.trimPrefix(key), value)
}

func (ws updateProposalWrappedStore) Delete(key []byte) {
	ws.getStore(key).Delete(ws.trimPrefix(key))
}

func (ws updateProposalWrappedStore) GetStoreType() storetypes.StoreType {
	return ws.subjectStore.GetStoreType()
}

func (ws updateProposalWrappedStore) Iterator(start, end []byte) sdk.Iterator {
	return ws.getStore(start).Iterator(ws.trimPrefix(start), ws.trimPrefix(end))
}

func (ws updateProposalWrappedStore) ReverseIterator(start, end []byte) sdk.Iterator {
	return ws.getStore(start).ReverseIterator(ws.trimPrefix(start), ws.trimPrefix(end))
}

func (ws updateProposalWrappedStore) CacheWrap() storetypes.CacheWrap {
	return cachekv.NewStore(ws)
}

func (ws updateProposalWrappedStore) CacheWrapWithTrace(w io.Writer, tc storetypes.TraceContext) storetypes.CacheWrap {
	return cachekv.NewStore(tracekv.NewStore(ws, w, tc))
}

func (ws updateProposalWrappedStore) CacheWrapWithListeners(storeKey storetypes.StoreKey, listeners []storetypes.WriteListener) storetypes.CacheWrap {
	return cachekv.NewStore(listenkv.NewStore(ws, storeKey, listeners))
}

func (ws updateProposalWrappedStore) trimPrefix(key []byte) []byte {
	if bytes.HasPrefix(key, ws.subjectPrefix) {
		key = bytes.TrimPrefix(key, ws.subjectPrefix)
	} else {
		key = bytes.TrimPrefix(key, ws.substitutePrefix)
	}

	return key
}

func (ws updateProposalWrappedStore) getStore(key []byte) sdk.KVStore {
	if bytes.HasPrefix(key, ws.subjectPrefix) {
		return ws.subjectStore
	}

	return ws.substituteStore
}

var _ wasmvmtypes.KVStore = &storeAdapter{}

// storeAdapter adapter to bridge SDK store impl to wasmvm
type storeAdapter struct {
	parent sdk.KVStore
}

// newStoreAdapter constructor
func newStoreAdapter(s sdk.KVStore) *storeAdapter {
	if s == nil {
		panic(errors.New("store must not be nil"))
	}
	return &storeAdapter{parent: s}
}

func (s storeAdapter) Get(key []byte) []byte {
	return s.parent.Get(key)
}

func (s storeAdapter) Set(key, value []byte) {
	s.parent.Set(key, value)
}

func (s storeAdapter) Delete(key []byte) {
	s.parent.Delete(key)
}

func (s storeAdapter) Iterator(start, end []byte) wasmvmtypes.Iterator {
	return s.parent.Iterator(start, end)
}

func (s storeAdapter) ReverseIterator(start, end []byte) wasmvmtypes.Iterator {
	return s.parent.ReverseIterator(start, end)
}
