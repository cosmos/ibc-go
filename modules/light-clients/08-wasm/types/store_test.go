package types_test

import (
	"errors"
	fmt "fmt"

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
		expPanic error
	}{
		{
			"success: subject store",
			types.SubjectPrefix,
			subjectStore,
			nil,
		},
		{
			"success: substitute store",
			types.SubstitutePrefix,
			substituteStore,
			nil,
		},
		{
			"failure: invalid prefix",
			invalidPrefix,
			nil,
			fmt.Errorf("key must be prefixed with either \"%s\" or \"%s\"", types.SubjectPrefix, types.SubstitutePrefix),
		},
		{
			"failure: invalid prefix contains both subject/ and substitute/",
			append(types.SubjectPrefix, types.SubstitutePrefix...),
			nil,
			fmt.Errorf("key must be prefixed with either \"%s\" or \"%s\"", types.SubjectPrefix, types.SubstitutePrefix),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

			if tc.expPanic == nil {
				suite.Require().Equal(tc.expStore, wrappedStore.GetStore(tc.prefix))
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), func() { wrappedStore.GetStore(tc.prefix) })
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
		expPanic error
	}{
		{
			"success: subject store Get",
			types.SubjectPrefix,
			host.ClientStateKey(),
			subjectStore,
			nil,
		},
		{
			"success: substitute store Get",
			types.SubstitutePrefix,
			host.ClientStateKey(),
			substituteStore,
			nil,
		},
		{
			"failure: key not prefixed with subject/ or substitute/",
			invalidPrefix,
			host.ClientStateKey(),
			nil,
			fmt.Errorf("key must be prefixed with either \"%s\" or \"%s\"", types.SubjectPrefix, types.SubstitutePrefix),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			if tc.expPanic == nil {
				expValue := tc.expStore.Get(tc.key)

				suite.Require().Equal(expValue, wrappedStore.Get(prefixedKey))
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), func() { wrappedStore.Get(prefixedKey) })
			}
		})
	}
}

// TestMigrateClientWrappedStoreSet tests the Set method of the migrateClientWrappedStore.
func (suite *TypesTestSuite) TestMigrateClientWrappedStoreSet() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		prefix   []byte
		key      []byte
		expStore storetypes.KVStore
		expPanic error
	}{
		{
			"success: subject store Set",
			types.SubjectPrefix,
			host.ClientStateKey(),
			subjectStore,
			nil,
		},
		{
			"failure: cannot Set on substitute store",
			types.SubstitutePrefix,
			host.ClientStateKey(),
			nil,
			fmt.Errorf("writes only allowed on subject store; key must be prefixed with \"%s\"", types.SubjectPrefix),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			if tc.expPanic == nil {
				wrappedStore.Set(prefixedKey, wasmtesting.MockClientStateBz)

				expValue := tc.expStore.Get(tc.key)

				suite.Require().Equal(expValue, wasmtesting.MockClientStateBz)
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), func() { wrappedStore.Set(prefixedKey, wasmtesting.MockClientStateBz) })
			}
		})
	}
}

// TestMigrateClientWrappedStoreDelete tests the Delete method of the migrateClientWrappedStore.
func (suite *TypesTestSuite) TestMigrateClientWrappedStoreDelete() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		prefix   []byte
		key      []byte
		expStore storetypes.KVStore
		expPanic error
	}{
		{
			"success: subject store Delete",
			types.SubjectPrefix,
			host.ClientStateKey(),
			subjectStore,
			nil,
		},
		{
			"failure: cannot Delete on substitute store",
			types.SubstitutePrefix,
			host.ClientStateKey(),
			nil,
			fmt.Errorf("writes only allowed on subject store; key must be prefixed with \"%s\"", types.SubjectPrefix),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewMigrateProposalWrappedStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			if tc.expPanic == nil {
				wrappedStore.Delete(prefixedKey)

				suite.Require().False(tc.expStore.Has(tc.key))
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), func() { wrappedStore.Delete(prefixedKey) })
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
		expPanic    error
	}{
		{
			"success: subject store Iterate",
			types.SubjectPrefix,
			types.SubjectPrefix,
			[]byte("start"),
			[]byte("end"),
			nil,
		},
		{
			"success: substitute store Iterate",
			types.SubstitutePrefix,
			types.SubstitutePrefix,
			[]byte("start"),
			[]byte("end"),
			nil,
		},
		{
			"failure: key not prefixed",
			invalidPrefix,
			invalidPrefix,
			[]byte("start"),
			[]byte("end"),
			fmt.Errorf("key must be prefixed with either \"%s\" or \"%s\"", types.SubjectPrefix, types.SubstitutePrefix),
		},
		{
			"failure: start and end keys not prefixed with same prefix",
			types.SubjectPrefix,
			types.SubstitutePrefix,
			[]byte("start"),
			[]byte("end"),
			errors.New("start and end keys must be prefixed with the same prefix"),
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

			if tc.expPanic == nil {
				suite.Require().NotNil(wrappedStore.Iterator(prefixedKeyStart, prefixedKeyEnd))
				suite.Require().NotNil(wrappedStore.ReverseIterator(prefixedKeyStart, prefixedKeyEnd))
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), func() { wrappedStore.Iterator(prefixedKeyStart, prefixedKeyEnd) })
				suite.Require().PanicsWithError(tc.expPanic.Error(), func() { wrappedStore.ReverseIterator(prefixedKeyStart, prefixedKeyEnd) })
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
				clientStore = types.NewMigrateProposalWrappedStore(clientStore, nil)
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
