package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/cachekv"
	storeprefix "cosmossdk.io/store/prefix"
	"cosmossdk.io/store/tracekv"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var (
	_ wasmvmtypes.KVStore = &storeAdapter{}
	_ storetypes.KVStore  = &updateProposalWrappedStore{}

	subjectPrefix    = []byte("subject/")
	substitutePrefix = []byte("substitute/")
)

// updateProposalWrappedStore combines two KVStores into one.
//
// Both stores are used for reads, but only the subjectStore is used for writes. For all operations, the key
// is checked to determine which store to use and must be prefixed with either "subject/" or "substitute/" accordingly.
// If the key is not prefixed with either "subject/" or "substitute/", a panic is thrown.
type updateProposalWrappedStore struct {
	subjectStore    storetypes.KVStore
	substituteStore storetypes.KVStore
}

func newUpdateProposalWrappedStore(subjectStore, substituteStore storetypes.KVStore) updateProposalWrappedStore {
	return updateProposalWrappedStore{
		subjectStore:    subjectStore,
		substituteStore: substituteStore,
	}
}

// Get implements the storetypes.KVStore interface. It allows reads from both the subjectStore and substituteStore.
//
// Get will panic if the key is not prefixed with either "subject/" or "substitute/".
func (ws updateProposalWrappedStore) Get(key []byte) []byte {
	prefix, key := splitPrefix(key)

	return ws.getStore(prefix).Get(key)
}

// Has implements the storetypes.KVStore interface. It allows reads from both the subjectStore and substituteStore.
//
// Note: contracts do not have access to the Has method, it is only implemented here to satisfy the storetypes.KVStore interface.
func (ws updateProposalWrappedStore) Has(key []byte) bool {
	prefix, key := splitPrefix(key)

	return ws.getStore(prefix).Has(key)
}

// Set implements the storetypes.KVStore interface. It allows writes solely to the subjectStore.
//
// Set will panic if the key is not prefixed with "subject/".
func (ws updateProposalWrappedStore) Set(key, value []byte) {
	prefix, key := splitPrefix(key)
	if !bytes.Equal(prefix, subjectPrefix) {
		panic(errors.New("writes only allowed on subject store; key must be prefixed with subject/"))
	}

	ws.subjectStore.Set(key, value)
}

// Delete implements the storetypes.KVStore interface. It allows deletions solely to the subjectStore.
//
// Delete will panic if the key is not prefixed with "subject/".
func (ws updateProposalWrappedStore) Delete(key []byte) {
	prefix, key := splitPrefix(key)
	if !bytes.Equal(prefix, subjectPrefix) {
		panic(fmt.Errorf("writes only allowed on subject store; key must be prefixed with %s", subjectPrefix))
	}

	ws.subjectStore.Delete(key)
}

// Iterator implements the storetypes.KVStore interface. It allows iteration over both the subjectStore and substituteStore.
//
// Iterator will panic if the start or end keys are not prefixed with either "subject/" or "substitute/".
func (ws updateProposalWrappedStore) Iterator(start, end []byte) storetypes.Iterator {
	prefixStart, start := splitPrefix(start)
	prefixEnd, end := splitPrefix(end)

	if !bytes.Equal(prefixStart, prefixEnd) {
		panic(fmt.Errorf("writes only allowed on subject store; key must be prefixed with %s", subjectPrefix))
	}

	return ws.getStore(prefixStart).Iterator(start, end)
}

// ReverseIterator implements the storetypes.KVStore interface. It allows iteration over both the subjectStore and substituteStore.
//
// ReverseIterator will panic if the start or end keys are not prefixed with either "subject/" or "substitute/".
func (ws updateProposalWrappedStore) ReverseIterator(start, end []byte) storetypes.Iterator {
	prefixStart, start := splitPrefix(start)
	prefixEnd, end := splitPrefix(end)

	if !bytes.Equal(prefixStart, prefixEnd) {
		panic(errors.New("start and end keys must be prefixed with the same prefix"))
	}

	return ws.getStore(prefixStart).ReverseIterator(start, end)
}

// GetStoreType implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws updateProposalWrappedStore) GetStoreType() storetypes.StoreType {
	return ws.substituteStore.GetStoreType()
}

// CacheWrap implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws updateProposalWrappedStore) CacheWrap() storetypes.CacheWrap {
	return cachekv.NewStore(ws)
}

// CacheWrapWithTrace implements the storetypes.KVStore interface, it is implemented solely to satisfy the interface.
func (ws updateProposalWrappedStore) CacheWrapWithTrace(w io.Writer, tc storetypes.TraceContext) storetypes.CacheWrap {
	return cachekv.NewStore(tracekv.NewStore(ws, w, tc))
}

// getStore returns the store to be used for the given key. If the key is prefixed with "subject/", the subjectStore
// is returned. If the key is prefixed with "substitute/", the substituteStore is returned.
//
// If the key is not prefixed with either "subject/" or "substitute/", a panic is thrown.
func (ws updateProposalWrappedStore) getStore(prefix []byte) storetypes.KVStore {
	if bytes.Equal(prefix, subjectPrefix) {
		return ws.subjectStore
	} else if bytes.Equal(prefix, substitutePrefix) {
		return ws.substituteStore
	}

	panic(errors.New("key must be prefixed with either subject/ or substitute/"))
}

// splitPrefix splits the key into the prefix and the key itself, if the key is prefixed with either "subject/" or "substitute/".
// If the key is not prefixed with either "subject/" or "substitute/", the prefix is nil.
func splitPrefix(key []byte) ([]byte, []byte) {
	if bytes.HasPrefix(key, subjectPrefix) {
		return subjectPrefix, bytes.TrimPrefix(key, subjectPrefix)
	} else if bytes.HasPrefix(key, substitutePrefix) {
		return substitutePrefix, bytes.TrimPrefix(key, substitutePrefix)
	}

	return nil, key
}

// storeAdapter bridges the SDK store implementation to wasmvm one. It implements the wasmvmtypes.KVStore interface.
type storeAdapter struct {
	parent storetypes.KVStore
}

// newStoreAdapter constructor
func newStoreAdapter(s storetypes.KVStore) *storeAdapter {
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

// getClientID extracts and validates the clientID from the clientStore's prefix.
//
// Due to the 02-client module not passing the clientID to the 08-wasm module,
// this function was devised to infer it from the store's prefix.
// The expected format of the clientStore prefix is "<placeholder>/{clientID}/".
// If the clientStore is of type updateProposalWrappedStore, the subjectStore's prefix is utilized instead.
func getClientID(clientStore storetypes.KVStore) (string, error) {
	upws, isUpdateProposalWrappedStore := clientStore.(updateProposalWrappedStore)
	if isUpdateProposalWrappedStore {
		// if the clientStore is a updateProposalWrappedStore, we retrieve the subjectStore
		// because the contract call will be made on the client with the ID of the subjectStore
		clientStore = upws.subjectStore
	}

	store, ok := clientStore.(storeprefix.Store)
	if !ok {
		return "", errorsmod.Wrap(ErrRetrieveClientID, "clientStore is not a prefix store")
	}

	// using reflect to retrieve the private prefix field
	r := reflect.ValueOf(&store).Elem()

	f := r.FieldByName("prefix")
	if !f.IsValid() {
		return "", errorsmod.Wrap(ErrRetrieveClientID, "prefix field not found")
	}

	prefix := string(f.Bytes())

	split := strings.Split(prefix, "/")
	if len(split) < 3 {
		return "", errorsmod.Wrap(ErrRetrieveClientID, "prefix is not of the expected form")
	}

	// the clientID is the second to last element of the prefix
	// the prefix is expected to be of the form "<placeholder>/{clientID}/"
	clientID := split[len(split)-2]
	isClientID := strings.HasPrefix(clientID, exported.Wasm)
	if !isClientID {
		return "", errorsmod.Wrapf(ErrRetrieveClientID, "prefix does not contain a %s clientID", exported.Wasm)
	}

	return clientID, nil
}
