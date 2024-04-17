package types_test

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var invalidPrefix = []byte("invalid/")

// TestMigrateClientWrappedStoreGetStore tests the getStore method of the migrateClientWrappedStore.
func (suite *TypesTestSuite) TestMigrateClientWrappedStoreGetStore() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		prefix   []byte
		expStore storetypes.KVStore
	}{
		{
			"success: subject store",
			types.SubjectPrefix,
			subjectStore,
		},
		{
			"success: substitute store",
			types.SubstitutePrefix,
			substituteStore,
		},
		{
			"failure: invalid prefix",
			invalidPrefix,
			nil,
		},
		{
			"failure: invalid prefix contains both subject/ and substitute/",
			append(types.SubjectPrefix, types.SubstitutePrefix...),
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewMigrateClientWrappedStore(subjectStore, substituteStore)

			store, found := wrappedStore.GetStore(tc.prefix)

			storeFound := tc.expStore != nil
			if storeFound {
				suite.Require().Equal(tc.expStore, store)
				suite.Require().True(found)
			} else {
				suite.Require().Nil(store)
				suite.Require().False(found)
			}
		})
	}
}

// TestSplitPrefix tests the splitPrefix function.
func (suite *TypesTestSuite) TestSplitPrefix() {
	clientStateKey := host.ClientStateKey()

	testCases := []struct {
		name      string
		prefix    []byte
		expValues [][]byte
	}{
		{
			"success: subject prefix",
			append(types.SubjectPrefix, clientStateKey...),
			[][]byte{types.SubjectPrefix, clientStateKey},
		},
		{
			"success: substitute prefix",
			append(types.SubstitutePrefix, clientStateKey...),
			[][]byte{types.SubstitutePrefix, clientStateKey},
		},
		{
			"success: nil prefix returned",
			invalidPrefix,
			[][]byte{nil, invalidPrefix},
		},
		{
			"success: invalid prefix contains both subject/ and substitute/",
			append(types.SubjectPrefix, types.SubstitutePrefix...),
			[][]byte{types.SubjectPrefix, types.SubstitutePrefix},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			keyPrefix, key := types.SplitPrefix(tc.prefix)

			suite.Require().Equal(tc.expValues[0], keyPrefix)
			suite.Require().Equal(tc.expValues[1], key)
		})
	}
}

// TestMigrateClientWrappedStoreGet tests the Get method of the migrateClientWrappedStore.
func (suite *TypesTestSuite) TestMigrateClientWrappedStoreGet() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		prefix   []byte
		key      []byte
		expStore storetypes.KVStore
	}{
		{
			"success: subject store Get",
			types.SubjectPrefix,
			host.ClientStateKey(),
			subjectStore,
		},
		{
			"success: substitute store Get",
			types.SubstitutePrefix,
			host.ClientStateKey(),
			substituteStore,
		},
		{
			"failure: key not prefixed with subject/ or substitute/",
			invalidPrefix,
			host.ClientStateKey(),
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewMigrateClientWrappedStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			storeFound := tc.expStore != nil
			if storeFound {
				expValue := tc.expStore.Get(tc.key)

				suite.Require().Equal(expValue, wrappedStore.Get(prefixedKey))
			} else {
				// expected value when store is not found is an empty byte slice
				suite.Require().Equal([]byte(nil), wrappedStore.Get(prefixedKey))
			}
		})
	}
}

// TestMigrateClientWrappedStoreSet tests the Set method of the migrateClientWrappedStore.
func (suite *TypesTestSuite) TestMigrateClientWrappedStoreSet() {
	testCases := []struct {
		name   string
		prefix []byte
		key    []byte
		expSet bool
	}{
		{
			"success: subject store Set",
			types.SubjectPrefix,
			host.ClientStateKey(),
			true,
		},
		{
			"failure: cannot Set on substitute store",
			types.SubstitutePrefix,
			host.ClientStateKey(),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
			subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()
			wrappedStore := types.NewMigrateClientWrappedStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			wrappedStore.Set(prefixedKey, wasmtesting.MockClientStateBz)

			if tc.expSet {
				store, found := wrappedStore.GetStore(tc.prefix)
				suite.Require().True(found)
				suite.Require().Equal(subjectStore, store)

				value := store.Get(tc.key)

				suite.Require().Equal(wasmtesting.MockClientStateBz, value)
			} else {
				// Assert that no writes happened to subject or substitute store
				suite.Require().NotEqual(wasmtesting.MockClientStateBz, subjectStore.Get(tc.key))
				suite.Require().NotEqual(wasmtesting.MockClientStateBz, substituteStore.Get(tc.key))
			}
		})
	}
}

// TestMigrateClientWrappedStoreDelete tests the Delete method of the migrateClientWrappedStore.
func (suite *TypesTestSuite) TestMigrateClientWrappedStoreDelete() {
	var (
		mockStoreKey   = []byte("mock-key")
		mockStoreValue = []byte("mock-value")
	)

	testCases := []struct {
		name      string
		prefix    []byte
		key       []byte
		expDelete bool
	}{
		{
			"success: subject store Delete",
			types.SubjectPrefix,
			mockStoreKey,
			true,
		},
		{
			"failure: cannot Delete on substitute store",
			types.SubstitutePrefix,
			mockStoreKey,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
			subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

			// Set values under the mock key:
			subjectStore.Set(mockStoreKey, mockStoreValue)
			substituteStore.Set(mockStoreKey, mockStoreValue)

			wrappedStore := types.NewMigrateClientWrappedStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			wrappedStore.Delete(prefixedKey)

			if tc.expDelete {
				store, found := wrappedStore.GetStore(tc.prefix)
				suite.Require().True(found)
				suite.Require().Equal(subjectStore, store)

				suite.Require().False(store.Has(tc.key))
			} else {
				// Assert that no deletions happened to subject or substitute store
				suite.Require().True(subjectStore.Has(tc.key))
				suite.Require().True(substituteStore.Has(tc.key))
			}
		})
	}
}

// TestMigrateClientWrappedStoreIterators tests the Iterator/ReverseIterator methods of the migrateClientWrappedStore.
func (suite *TypesTestSuite) TestMigrateClientWrappedStoreIterators() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name        string
		prefixStart []byte
		prefixEnd   []byte
		start       []byte
		end         []byte
		expValid    bool
	}{
		{
			"success: subject store Iterate",
			types.SubjectPrefix,
			types.SubjectPrefix,
			[]byte("start"),
			[]byte("end"),
			true,
		},
		{
			"success: substitute store Iterate",
			types.SubstitutePrefix,
			types.SubstitutePrefix,
			[]byte("start"),
			[]byte("end"),
			true,
		},
		{
			"failure: key not prefixed",
			invalidPrefix,
			invalidPrefix,
			[]byte("start"),
			[]byte("end"),
			false,
		},
		{
			"failure: start and end keys not prefixed with same prefix",
			types.SubjectPrefix,
			types.SubstitutePrefix,
			[]byte("start"),
			[]byte("end"),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewMigrateClientWrappedStore(subjectStore, substituteStore)

			prefixedKeyStart := tc.prefixStart
			prefixedKeyStart = append(prefixedKeyStart, tc.start...)
			prefixedKeyEnd := tc.prefixEnd
			prefixedKeyEnd = append(prefixedKeyEnd, tc.end...)

			if tc.expValid {
				suite.Require().NotNil(wrappedStore.Iterator(prefixedKeyStart, prefixedKeyEnd))
				suite.Require().NotNil(wrappedStore.ReverseIterator(prefixedKeyStart, prefixedKeyEnd))
			} else {
				// Iterator returned should be Closed, calling `Valid` should return false
				suite.Require().False(wrappedStore.Iterator(prefixedKeyStart, prefixedKeyEnd).Valid())
				suite.Require().False(wrappedStore.ReverseIterator(prefixedKeyStart, prefixedKeyEnd).Valid())
			}
		})
	}
}

func (suite *TypesTestSuite) TestNewMigrateClientWrappedStore() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		malleate func()
		expPanic bool
	}{
		{
			"success",
			func() {},
			false,
		},
		{
			"failure: subject store is nil",
			func() {
				subjectStore = nil
			},
			true,
		},
		{
			"failure: substitute store is nil",
			func() {
				substituteStore = nil
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			tc.malleate()

			expPass := !tc.expPanic
			if expPass {
				suite.Require().NotPanics(func() {
					types.NewMigrateClientWrappedStore(subjectStore, substituteStore)
				})
			} else {
				suite.Require().Panics(func() {
					types.NewMigrateClientWrappedStore(subjectStore, substituteStore)
				})
			}
		})
	}
}

// GetSubjectAndSubstituteStore returns two KVStores for testing the migrate client wrapping store.
func (suite *TypesTestSuite) GetSubjectAndSubstituteStore() (storetypes.KVStore, storetypes.KVStore) {
	suite.SetupTest()

	ctx := suite.chainA.GetContext()
	storeKey := GetSimApp(suite.chainA).GetKey(ibcexported.StoreKey)

	subjectClientStore := prefix.NewStore(ctx.KVStore(storeKey), []byte(clienttypes.FormatClientIdentifier(types.Wasm, 0)))
	substituteClientStore := prefix.NewStore(ctx.KVStore(storeKey), []byte(clienttypes.FormatClientIdentifier(types.Wasm, 1)))

	return subjectClientStore, substituteClientStore
}
