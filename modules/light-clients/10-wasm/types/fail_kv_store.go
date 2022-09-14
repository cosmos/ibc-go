package types

import (
	"io"

	store "github.com/cosmos/cosmos-sdk/store/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/types"
)

var _ types.KVStore = (*FailKVStore)(nil)

type FailKVStore struct {
}

func (f *FailKVStore) GetStoreType() store.StoreType {
	panic("not available for this method of IBC contract")
}

func (f *FailKVStore) CacheWrap() store.CacheWrap {
	panic("not available for this method of IBC contract")
}

func (f *FailKVStore) CacheWrapWithTrace(w io.Writer, tc store.TraceContext) store.CacheWrap {
	panic("not available for this method of IBC contract")
}

func (f *FailKVStore) CacheWrapWithListeners(storeKey storetypes.StoreKey, listeners []store.WriteListener) storetypes.CacheWrap {
	panic("not available for this method of IBC contract")
}

func (f *FailKVStore) Get(key []byte) []byte {
	panic("not available for this method of IBC contract")
}

func (f *FailKVStore) Has(key []byte) bool {
	panic("not available for this method of IBC contract")
}

func (f FailKVStore) Set(key, value []byte) {
	panic("not available for this method of IBC contract")
}

func (f FailKVStore) Delete(key []byte) {
	panic("not available for this method of IBC contract")
}

func (f FailKVStore) Iterator(start, end []byte) store.Iterator {
	panic("not available for this method of IBC contract")
}

func (f FailKVStore) ReverseIterator(start, end []byte) store.Iterator {
	panic("not available for this method of IBC contract")
}
