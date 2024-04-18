package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	"cosmossdk.io/store/cachekv"
	"cosmossdk.io/store/tracekv"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

var (
	_ wasmvmtypes.KVStore = &StoreAdapter{}
	_ storetypes.KVStore  = &MigrateClientWrappedStore{}

	SubjectPrefix    = []byte("subject/")
	SubstitutePrefix = []byte("substitute/")
)

// GetClientState retrieves the client state from the store using the provided KVStore and codec.
// It returns the unmarshaled ClientState and a boolean indicating if the state was found.
func GetClientState(store storetypes.KVStore, cdc codec.BinaryCodec) (*ClientState, bool) {
	bz := store.Get(host.ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}

	clientStateI := clienttypes.MustUnmarshalClientState(cdc, bz)
	var clientState *ClientState
	clientState, ok := clientStateI.(*ClientState)
	if !ok {
		panic(fmt.Errorf("cannot convert %T into %T", clientStateI, clientState))
	}
	return clientState, ok
}

// Checksum is a type alias used for wasm byte code checksums.
type Checksum = wasmvmtypes.Checksum

// CreateChecksum creates a sha256 checksum from the given wasm code, it forwards the
// call to the wasmvm package. The code is checked for the following conditions:
// - code length is zero.
// - code length is less than 4 bytes (magic number length).
// - code does not start with the wasm magic number.
func CreateChecksum(code []byte) (Checksum, error) {
	return wasmvm.CreateChecksum(code)
}

// MigrateClientWrappedStore combines two KVStores into one.
//
// Both stores are used for reads, but only the subjectStore is used for writes. For all operations, the key
// is checked to determine which store to use and must be prefixed with either "subject/" or "substitute/" accordingly.
// If the key is not prefixed with either "subject/" or "substitute/", a default action is taken (e.g. no-op for Set/Delete).
type MigrateClientWrappedStore struct {
	subjectStore    storetypes.KVStore
	substituteStore storetypes.KVStore
}

func NewMigrateClientWrappedStore(subjectStore, substituteStore storetypes.KVStore) MigrateClientWrappedStore {
	if subjectStore == nil {
		panic(errors.New("subjectStore must not be nil"))
	}
	if substituteStore == nil {
		panic(errors.New("substituteStore must not be nil"))
	}

	return MigrateClientWrappedStore{
		subjectStore:    subjectStore,
		substituteStore: substituteStore,
	}
}

// Get implements the storetypes.KVStore interface. It allows reads from both the subjectStore and substituteStore.
//
// Get will return an empty byte slice if the key is not prefixed with either "subject/" or "substitute/".
func (ws MigrateClientWrappedStore) Get(key []byte) []byte {
	prefix, key := splitPrefix(key)

	store, found := ws.getStore(prefix)
	if !found {
		// return a nil byte slice as KVStore.Get() does by default
		return []byte(nil)
	}

	return store.Get(key)
}

// Has implements the storetypes.KVStore interface. It allows reads from both the subjectStore and substituteStore.
//
// Note: contracts do not have access to the Has method, it is only implemented here to satisfy the storetypes.KVStore interface.
func (ws MigrateClientWrappedStore) Has(key []byte) bool {
	prefix, key := splitPrefix(key)

	store, found := ws.getStore(prefix)
	if !found {
		// return false as value when store is not found
		return false
	}

	return store.Has(key)
}

// Set implements the storetypes.KVStore interface. It allows writes solely to the subjectStore.
//
// Set will no-op if the key is not prefixed with "subject/".
func (ws MigrateClientWrappedStore) Set(key, value []byte) {
	prefix, key := splitPrefix(key)
	if !bytes.Equal(prefix, SubjectPrefix) {
		return // no-op
	}

	ws.subjectStore.Set(key, value)
}

// Delete implements the storetypes.KVStore interface. It allows deletions solely to the subjectStore.
//
// Delete will no-op if the key is not prefixed with "subject/".
func (ws MigrateClientWrappedStore) Delete(key []byte) {
	prefix, key := splitPrefix(key)
	if !bytes.Equal(prefix, SubjectPrefix) {
		return // no-op
	}

	ws.subjectStore.Delete(key)
}

// Iterator implements the storetypes.KVStore interface. It allows iteration over both the subjectStore and substituteStore.
//
// Iterator will return a closed iterator if the start or end keys are not prefixed with either "subject/" or "substitute/".
func (ws MigrateClientWrappedStore) Iterator(start, end []byte) storetypes.Iterator {
	prefixStart, start := splitPrefix(start)
	prefixEnd, end := splitPrefix(end)

	if !bytes.Equal(prefixStart, prefixEnd) {
		return ws.closedIterator()
	}

	store, found := ws.getStore(prefixStart)
	if !found {
		return ws.closedIterator()
	}

	return store.Iterator(start, end)
}

// ReverseIterator implements the storetypes.KVStore interface. It allows iteration over both the subjectStore and substituteStore.
//
// ReverseIterator will return a closed iterator if the start or end keys are not prefixed with either "subject/" or "substitute/".
func (ws MigrateClientWrappedStore) ReverseIterator(start, end []byte) storetypes.Iterator {
	prefixStart, start := splitPrefix(start)
	prefixEnd, end := splitPrefix(end)

	if !bytes.Equal(prefixStart, prefixEnd) {
		return ws.closedIterator()
	}

	store, found := ws.getStore(prefixStart)
	if !found {
		return ws.closedIterator()
	}

	return store.ReverseIterator(start, end)
}

// GetStoreType implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws MigrateClientWrappedStore) GetStoreType() storetypes.StoreType {
	return ws.substituteStore.GetStoreType()
}

// CacheWrap implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws MigrateClientWrappedStore) CacheWrap() storetypes.CacheWrap {
	return cachekv.NewStore(ws)
}

// CacheWrapWithTrace implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws MigrateClientWrappedStore) CacheWrapWithTrace(w io.Writer, tc storetypes.TraceContext) storetypes.CacheWrap {
	return cachekv.NewStore(tracekv.NewStore(ws, w, tc))
}

// getStore returns the store to be used for the given key and a boolean flag indicating if that store was found.
// If the key is prefixed with "subject/", the subjectStore is returned. If the key is prefixed with "substitute/",
// the substituteStore is returned.
//
// If the key is not prefixed with either "subject/" or "substitute/", a nil store is returned and the boolean flag is false.
func (ws MigrateClientWrappedStore) getStore(prefix []byte) (storetypes.KVStore, bool) {
	if bytes.Equal(prefix, SubjectPrefix) {
		return ws.subjectStore, true
	} else if bytes.Equal(prefix, SubstitutePrefix) {
		return ws.substituteStore, true
	}

	return nil, false
}

// closedIterator returns an iterator that is always closed, used when Iterator() or ReverseIterator() is called
// with an invalid prefix or start/end key.
func (ws MigrateClientWrappedStore) closedIterator() storetypes.Iterator {
	// Create a dummy iterator that is always closed right away.
	it := ws.subjectStore.Iterator([]byte{0}, []byte{1})
	it.Close()

	return it
}

// splitPrefix splits the key into the prefix and the key itself, if the key is prefixed with either "subject/" or "substitute/".
// If the key is not prefixed with either "subject/" or "substitute/", the prefix is nil.
func splitPrefix(key []byte) ([]byte, []byte) {
	if bytes.HasPrefix(key, SubjectPrefix) {
		return SubjectPrefix, bytes.TrimPrefix(key, SubjectPrefix)
	} else if bytes.HasPrefix(key, SubstitutePrefix) {
		return SubstitutePrefix, bytes.TrimPrefix(key, SubstitutePrefix)
	}

	return nil, key
}

// StoreAdapter bridges the SDK store implementation to wasmvm one. It implements the wasmvmtypes.KVStore interface.
type StoreAdapter struct {
	parent storetypes.KVStore
}

// NewStoreAdapter constructor
func NewStoreAdapter(s storetypes.KVStore) *StoreAdapter {
	if s == nil {
		panic(errors.New("store must not be nil"))
	}
	return &StoreAdapter{parent: s}
}

// Get implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) Get(key []byte) []byte {
	return s.parent.Get(key)
}

// Set implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) Set(key, value []byte) {
	s.parent.Set(key, value)
}

// Delete implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) Delete(key []byte) {
	s.parent.Delete(key)
}

// Iterator implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) Iterator(start, end []byte) wasmvmtypes.Iterator {
	return s.parent.Iterator(start, end)
}

// ReverseIterator implements the wasmvmtypes.KVStore interface.
func (s StoreAdapter) ReverseIterator(start, end []byte) wasmvmtypes.Iterator {
	return s.parent.ReverseIterator(start, end)
}
