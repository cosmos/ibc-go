package types_test

import (
	"errors"

	prefixstore "cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

var invalidPrefix = []byte("invalid/")

// TestGetStore tests the getStore method of the updateProposalWrappedStore.
func (suite *TypesTestSuite) TestGetStore() {
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
			errors.New("key must be prefixed with either subject/ or substitute/"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewUpdateProposalWrappedStore(subjectStore, substituteStore)

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
	testCases := []struct {
		name      string
		prefix    []byte
		expValues [][]byte
	}{
		{
			"success: subject prefix",
			types.SubjectPrefix,
			[][]byte{types.SubjectPrefix, []byte("")},
		},
		{
			"success: substitute prefix",
			types.SubstitutePrefix,
			[][]byte{types.SubstitutePrefix, []byte("")},
		},
		{
			"success: prefix returned unchanged",
			invalidPrefix,
			[][]byte{nil, invalidPrefix},
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

// TestUpdateProposalWrappedStoreGet tests the Get method of the updateProposalWrappedStore.
func (suite *TypesTestSuite) TestUpdateProposalWrappedStoreGet() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		key      []byte
		prefix   []byte
		expStore storetypes.KVStore
		expPanic error
	}{
		{
			"success: subject store Get",
			host.ClientStateKey(),
			types.SubjectPrefix,
			subjectStore,
			nil,
		},
		{
			"success: substitute store Get",
			host.ClientStateKey(),
			types.SubstitutePrefix,
			substituteStore,
			nil,
		},
		{
			"failure: key not prefixed with subject/ or substitute/",
			host.ClientStateKey(),
			invalidPrefix,
			nil,
			errors.New("key must be prefixed with either subject/ or substitute/"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewUpdateProposalWrappedStore(subjectStore, substituteStore)

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

// TestUpdateProposalWrappedStoreSet tests the Set method of the updateProposalWrappedStore.
func (suite *TypesTestSuite) TestUpdateProposalWrappedStoreSet() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		key      []byte
		prefix   []byte
		expStore storetypes.KVStore
		expPanic error
	}{
		{
			"success: subject store Set",
			host.ClientStateKey(),
			types.SubjectPrefix,
			subjectStore,
			nil,
		},
		{
			"failure: cannot Set on substitute store",
			host.ClientStateKey(),
			types.SubstitutePrefix,
			nil,
			errors.New("key must be prefixed with subject/"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewUpdateProposalWrappedStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			if tc.expPanic == nil {
				wrappedStore.Set(prefixedKey, wasmtesting.MockClientStateBz)

				expValue := tc.expStore.Get(tc.key)

				suite.Require().Equal(wasmtesting.MockClientStateBz, expValue)
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), func() { wrappedStore.Set(prefixedKey, wasmtesting.MockClientStateBz) })
			}
		})
	}
}

// TestUpdateProposalWrappedStoreDelete tests the Delete method of the updateProposalWrappedStore.
func (suite *TypesTestSuite) TestUpdateProposalWrappedStoreDelete() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		key      []byte
		prefix   []byte
		expStore storetypes.KVStore
		expPanic error
	}{
		{
			"success: subject store Delete",
			host.ClientStateKey(),
			types.SubjectPrefix,
			subjectStore,
			nil,
		},
		{
			"failure: cannot Delete on substitute store",
			host.ClientStateKey(),
			types.SubstitutePrefix,
			nil,
			errors.New("key must be prefixed with subject/"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewUpdateProposalWrappedStore(subjectStore, substituteStore)

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

// TestUpdateProposalWrappedStoreIterators tests the Iterator/ReverseIterator methods of the updateProposalWrappedStore.
func (suite *TypesTestSuite) TestUpdateProposalWrappedStoreIterators() {
	// calls suite.SetupWasmWithMockVM() and creates two clients with their respective stores
	subjectStore, substituteStore := suite.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name        string
		start       []byte
		end         []byte
		prefixStart []byte
		prefixEnd   []byte
		expPanic    error
	}{
		{
			"success: subject store Iterate",
			[]byte("start"),
			[]byte("end"),
			types.SubjectPrefix,
			types.SubjectPrefix,
			nil,
		},
		{
			"success: substitute store Iterate",
			[]byte("start"),
			[]byte("end"),
			types.SubstitutePrefix,
			types.SubstitutePrefix,
			nil,
		},
		{
			"failure: key not prefixed",
			[]byte("start"),
			[]byte("end"),
			invalidPrefix,
			invalidPrefix,
			errors.New("key must be prefixed with either subject/ or substitute/"),
		},
		{
			"failure: start and end keys not prefixed with same prefix",
			[]byte("start"),
			[]byte("end"),
			types.SubjectPrefix,
			types.SubstitutePrefix,
			errors.New("start and end keys must be prefixed with the same prefix"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			wrappedStore := types.NewUpdateProposalWrappedStore(subjectStore, substituteStore)

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
			"success: clientID retrieved from updateProposalWrappedStore",
			func() {
				clientStore = types.NewUpdateProposalWrappedStore(clientStore, nil)
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
