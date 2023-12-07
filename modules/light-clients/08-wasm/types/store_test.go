package types_test

import (
	prefixstore "cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
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
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

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
			prefix, key := types.SplitPrefix(tc.prefix)

			suite.Require().Equal(tc.expValues[0], prefix)
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
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

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
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

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
	testCases := []struct {
		name      string
		prefix    []byte
		key       []byte
		expDelete bool
	}{
		{
			"success: subject store Delete",
			types.SubjectPrefix,
			host.ClientStateKey(),
			true,
		},
		{
			"failure: cannot Delete on substitute store",
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
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

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
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

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
					types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)
				})
			} else {
				suite.Require().Panics(func() {
					types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)
				})
			}
		})
	}
}

func (suite *TypesTestSuite) TestGetClientID() {
	clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), defaultWasmClientID)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: clientID retrieved",
			func() {},
			nil,
		},
		{
			"success: clientID retrieved from migrateClientWrappedStore",
			func() {
				// substituteStore is ignored.
				clientStore = types.NewMigrateProposalWrappedStore(clientStore, clientStore)
			},
			nil,
		},
		{
			"failure: clientStore is nil",
			func() {
				clientStore = nil
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: prefix store does not contain prefix",
			func() {
				clientStore = prefixstore.NewStore(nil, nil)
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: prefix does not contain slash separated path",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("not-a-slash-separated-path"))
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: prefix only contains one slash",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("only-one-slash/"))
			},
			types.ErrRetrieveClientID,
		},
		{
			"failure: prefix does not contain a wasm clientID",
			func() {
				clientStore = prefixstore.NewStore(nil, []byte("/not-client-id/"))
			},
			types.ErrRetrieveClientID,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()
			clientID, err := types.GetClientID(clientStore)

			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().Equal(defaultWasmClientID, clientID)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

// GetSubjectAndSubstituteStore returns a subject and substitute store for testing.
func (suite *TypesTestSuite) GetSubjectAndSubstituteStore() (storetypes.KVStore, storetypes.KVStore) {
	suite.SetupWasmWithMockVM()

	endpointA := wasmtesting.NewWasmEndpoint(suite.chainA)
	err := endpointA.CreateClient()
	suite.Require().NoError(err)

	subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpointA.ClientID)

	substituteEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
	err = substituteEndpoint.CreateClient()
	suite.Require().NoError(err)

	substituteClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), substituteEndpoint.ClientID)

	return subjectClientStore, substituteClientStore
}
