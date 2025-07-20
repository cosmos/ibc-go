package types_test

import (
	"encoding/json"
	"errors"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	internaltypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/internal/types"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

var invalidPrefix = []byte("invalid/")

type TypesTestSuite struct {
	testifysuite.Suite
	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
}

func TestWasmTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TypesTestSuite))
}

func (s *TypesTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCustomAppCoordinator(s.T(), 1, setupTestingApp)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
}

// GetSimApp returns the duplicated SimApp from within the 08-wasm directory.
// This must be used instead of chain.GetSimApp() for tests within this directory.
func GetSimApp(chain *ibctesting.TestChain) *simapp.SimApp {
	app, ok := chain.App.(*simapp.SimApp)
	if !ok {
		panic(errors.New("chain is not a simapp.SimApp"))
	}
	return app
}

// setupTestingApp provides the duplicated simapp which is specific to the 08-wasm module on chain creation.
func setupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := simapp.NewUnitTestSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, nil)
	return app, app.DefaultGenesis()
}

// TestClientRecoveryStoreGetStore tests the GetStore method of the ClientRecoveryStore.
func (s *TypesTestSuite) TestClientRecoveryStoreGetStore() {
	subjectStore, substituteStore := s.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		prefix   []byte
		expStore storetypes.KVStore
	}{
		{
			"success: subject store",
			internaltypes.SubjectPrefix,
			subjectStore,
		},
		{
			"success: substitute store",
			internaltypes.SubstitutePrefix,
			substituteStore,
		},
		{
			"failure: invalid prefix",
			invalidPrefix,
			nil,
		},
		{
			"failure: invalid prefix contains both subject/ and substitute/",
			append(internaltypes.SubjectPrefix, internaltypes.SubstitutePrefix...),
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			wrappedStore := internaltypes.NewClientRecoveryStore(subjectStore, substituteStore)

			store, found := wrappedStore.GetStore(tc.prefix)

			storeFound := tc.expStore != nil
			if storeFound {
				s.Require().Equal(tc.expStore, store)
				s.Require().True(found)
			} else {
				s.Require().Nil(store)
				s.Require().False(found)
			}
		})
	}
}

// TestSplitPrefix tests the SplitPrefix function.
func (s *TypesTestSuite) TestSplitPrefix() {
	clientStateKey := host.ClientStateKey()

	testCases := []struct {
		name      string
		prefix    []byte
		expValues [][]byte
	}{
		{
			"success: subject prefix",
			append(internaltypes.SubjectPrefix, clientStateKey...),
			[][]byte{internaltypes.SubjectPrefix, clientStateKey},
		},
		{
			"success: substitute prefix",
			append(internaltypes.SubstitutePrefix, clientStateKey...),
			[][]byte{internaltypes.SubstitutePrefix, clientStateKey},
		},
		{
			"success: nil prefix returned",
			invalidPrefix,
			[][]byte{nil, invalidPrefix},
		},
		{
			"success: invalid prefix contains both subject/ and substitute/",
			append(internaltypes.SubjectPrefix, internaltypes.SubstitutePrefix...),
			[][]byte{internaltypes.SubjectPrefix, internaltypes.SubstitutePrefix},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			keyPrefix, key := internaltypes.SplitPrefix(tc.prefix)

			s.Require().Equal(tc.expValues[0], keyPrefix)
			s.Require().Equal(tc.expValues[1], key)
		})
	}
}

// TestClientRecoveryStoreGet tests the Get method of the ClientRecoveryStore.
func (s *TypesTestSuite) TestClientRecoveryStoreGet() {
	subjectStore, substituteStore := s.GetSubjectAndSubstituteStore()

	testCases := []struct {
		name     string
		prefix   []byte
		key      []byte
		expStore storetypes.KVStore
	}{
		{
			"success: subject store Get",
			internaltypes.SubjectPrefix,
			host.ClientStateKey(),
			subjectStore,
		},
		{
			"success: substitute store Get",
			internaltypes.SubstitutePrefix,
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
		s.Run(tc.name, func() {
			wrappedStore := internaltypes.NewClientRecoveryStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			storeFound := tc.expStore != nil
			if storeFound {
				expValue := tc.expStore.Get(tc.key)

				s.Require().Equal(expValue, wrappedStore.Get(prefixedKey))
			} else {
				// expected value when types is not found is an empty byte slice
				s.Require().Equal([]byte(nil), wrappedStore.Get(prefixedKey))
			}
		})
	}
}

// TestClientRecoveryStoreSet tests the Set method of the ClientRecoveryStore.
func (s *TypesTestSuite) TestClientRecoveryStoreSet() {
	testCases := []struct {
		name   string
		prefix []byte
		key    []byte
		expSet bool
	}{
		{
			"success: subject store Set",
			internaltypes.SubjectPrefix,
			host.ClientStateKey(),
			true,
		},
		{
			"failure: cannot Set on substitute store",
			internaltypes.SubstitutePrefix,
			host.ClientStateKey(),
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			subjectStore, substituteStore := s.GetSubjectAndSubstituteStore()
			wrappedStore := internaltypes.NewClientRecoveryStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			wrappedStore.Set(prefixedKey, wasmtesting.MockClientStateBz)

			if tc.expSet {
				store, found := wrappedStore.GetStore(tc.prefix)
				s.Require().True(found)
				s.Require().Equal(subjectStore, store)

				value := store.Get(tc.key)

				s.Require().Equal(wasmtesting.MockClientStateBz, value)
			} else {
				// Assert that no writes happened to subject or substitute types
				s.Require().NotEqual(wasmtesting.MockClientStateBz, subjectStore.Get(tc.key))
				s.Require().NotEqual(wasmtesting.MockClientStateBz, substituteStore.Get(tc.key))
			}
		})
	}
}

// TestClientRecoveryStoreDelete tests the Delete method of the ClientRecoveryStore.
func (s *TypesTestSuite) TestClientRecoveryStoreDelete() {
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
			internaltypes.SubjectPrefix,
			mockStoreKey,
			true,
		},
		{
			"failure: cannot Delete on substitute store",
			internaltypes.SubstitutePrefix,
			mockStoreKey,
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			subjectStore, substituteStore := s.GetSubjectAndSubstituteStore()

			// Set values under the mock key:
			subjectStore.Set(mockStoreKey, mockStoreValue)
			substituteStore.Set(mockStoreKey, mockStoreValue)

			wrappedStore := internaltypes.NewClientRecoveryStore(subjectStore, substituteStore)

			prefixedKey := tc.prefix
			prefixedKey = append(prefixedKey, tc.key...)

			wrappedStore.Delete(prefixedKey)

			if tc.expDelete {
				store, found := wrappedStore.GetStore(tc.prefix)
				s.Require().True(found)
				s.Require().Equal(subjectStore, store)

				s.Require().False(store.Has(tc.key))
			} else {
				// Assert that no deletions happened to subject or substitute types
				s.Require().True(subjectStore.Has(tc.key))
				s.Require().True(substituteStore.Has(tc.key))
			}
		})
	}
}

// TestClientRecoveryStoreIterators tests the Iterator/ReverseIterator methods of the ClientRecoveryStore.
func (s *TypesTestSuite) TestClientRecoveryStoreIterators() {
	subjectStore, substituteStore := s.GetSubjectAndSubstituteStore()

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
			internaltypes.SubjectPrefix,
			internaltypes.SubjectPrefix,
			[]byte("start"),
			[]byte("end"),
			true,
		},
		{
			"success: substitute store Iterate",
			internaltypes.SubstitutePrefix,
			internaltypes.SubstitutePrefix,
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
			internaltypes.SubjectPrefix,
			internaltypes.SubstitutePrefix,
			[]byte("start"),
			[]byte("end"),
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			wrappedStore := internaltypes.NewClientRecoveryStore(subjectStore, substituteStore)

			prefixedKeyStart := tc.prefixStart
			prefixedKeyStart = append(prefixedKeyStart, tc.start...)
			prefixedKeyEnd := tc.prefixEnd
			prefixedKeyEnd = append(prefixedKeyEnd, tc.end...)

			if tc.expValid {
				s.Require().NotNil(wrappedStore.Iterator(prefixedKeyStart, prefixedKeyEnd))
				s.Require().NotNil(wrappedStore.ReverseIterator(prefixedKeyStart, prefixedKeyEnd))
			} else {
				// Iterator returned should be Closed, calling `Valid` should return false
				s.Require().False(wrappedStore.Iterator(prefixedKeyStart, prefixedKeyEnd).Valid())
				s.Require().False(wrappedStore.ReverseIterator(prefixedKeyStart, prefixedKeyEnd).Valid())
			}
		})
	}
}

func (s *TypesTestSuite) TestNewClientRecoveryStore() {
	subjectStore, substituteStore := s.GetSubjectAndSubstituteStore()

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
		s.Run(tc.name, func() {
			tc.malleate()

			if !tc.expPanic {
				s.Require().NotPanics(func() {
					internaltypes.NewClientRecoveryStore(subjectStore, substituteStore)
				})
			} else {
				s.Require().Panics(func() {
					internaltypes.NewClientRecoveryStore(subjectStore, substituteStore)
				})
			}
		})
	}
}

// GetSubjectAndSubstituteStore returns two KVStores for testing the migrate client wrapping types.
func (s *TypesTestSuite) GetSubjectAndSubstituteStore() (storetypes.KVStore, storetypes.KVStore) {
	s.SetupTest()

	ctx := s.chainA.GetContext()
	storeKey := GetSimApp(s.chainA).GetKey(ibcexported.StoreKey)

	subjectClientStore := prefix.NewStore(ctx.KVStore(storeKey), []byte(clienttypes.FormatClientIdentifier(types.Wasm, 0)))
	substituteClientStore := prefix.NewStore(ctx.KVStore(storeKey), []byte(clienttypes.FormatClientIdentifier(types.Wasm, 1)))

	return subjectClientStore, substituteClientStore
}
