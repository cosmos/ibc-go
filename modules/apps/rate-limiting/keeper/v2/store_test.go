package v2_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	v2 "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/migrations/v2"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type MigrationsV2TestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	cdc         codec.BinaryCodec
}

func (suite *MigrationsV2TestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.cdc = suite.chainA.App.AppCodec()
}

func TestMigrationsV2TestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsV2TestSuite))
}

func (suite *MigrationsV2TestSuite) TestMigrateStore() {
	ctx := suite.chainA.GetContext()
	storeService := runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(types.StoreKey))
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

	// Create test whitelist entries using the legacy prefix
	whitelistPairs := []types.WhitelistedAddressPair{
		{
			Sender:   "cosmos1abc123",
			Receiver: "cosmos1def456",
		},
		{
			Sender:   "cosmos1ghi789",
			Receiver: "cosmos1jkl012",
		},
	}

	// Store entries with the legacy prefix
	for _, pair := range whitelistPairs {
		key := append(types.LegacyAddressWhitelistKeyPrefix, types.AddressWhitelistKey(pair.Sender, pair.Receiver)...)
		value := suite.cdc.MustMarshal(&pair)
		store.Set(key, value)
	}

	// Verify entries exist with legacy prefix
	for _, pair := range whitelistPairs {
		key := append(types.LegacyAddressWhitelistKeyPrefix, types.AddressWhitelistKey(pair.Sender, pair.Receiver)...)
		suite.Require().True(store.Has(key))
	}

	// Run migration
	err := v2.MigrateStore(ctx, storeService, suite.cdc)
	suite.Require().NoError(err)

	// Verify entries no longer exist with legacy prefix
	for _, pair := range whitelistPairs {
		key := append(types.LegacyAddressWhitelistKeyPrefix, types.AddressWhitelistKey(pair.Sender, pair.Receiver)...)
		suite.Require().False(store.Has(key))
	}

	// Verify entries exist with new prefix
	for _, pair := range whitelistPairs {
		key := append(types.AddressWhitelistKeyPrefix, types.AddressWhitelistKey(pair.Sender, pair.Receiver)...)
		suite.Require().True(store.Has(key))

		// Verify the value is preserved correctly
		value := store.Get(key)
		var retrievedPair types.WhitelistedAddressPair
		suite.cdc.MustUnmarshal(value, &retrievedPair)
		suite.Require().Equal(pair.Sender, retrievedPair.Sender)
		suite.Require().Equal(pair.Receiver, retrievedPair.Receiver)
	}
}

func (suite *MigrationsV2TestSuite) TestMigrateStoreEmptyStore() {
	ctx := suite.chainA.GetContext()
	storeService := runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(types.StoreKey))

	// Run migration on empty store
	err := v2.MigrateStore(ctx, storeService, suite.cdc)
	suite.Require().NoError(err)

	// Verify no entries exist
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, types.LegacyAddressWhitelistKeyPrefix)
	defer iterator.Close()
	suite.Require().False(iterator.Valid())

	iterator = storetypes.KVStorePrefixIterator(store, types.AddressWhitelistKeyPrefix)
	defer iterator.Close()
	suite.Require().False(iterator.Valid())
}
