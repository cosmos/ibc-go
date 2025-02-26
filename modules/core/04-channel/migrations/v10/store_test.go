package v10_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	v10 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v10"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type MigrationsV10TestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *MigrationsV10TestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestMigrationsV10TestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsV10TestSuite))
}

// set up channels that are still in upgrade state, and assert that the upgrade fails.
// migrate the store, and assert that the channels have been upgraded and state removed as expected
func (suite *MigrationsV10TestSuite) TestMigrateStoreWithUpgradingChannels() {
	ctx := suite.chainA.GetContext()
	cdc := suite.chainA.App.AppCodec()
	channelKeeper := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper
	storeService := runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey))

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.Setup()
	path = ibctesting.NewPath(suite.chainA, suite.chainB)
	path.Setup()

	preMigrationChannels := suite.getPreMigrationTypeChannels(ctx, cdc, storeService)
	suite.Require().Len(preMigrationChannels, 2)

	// Set up some channels with old state
	flushingChannel := preMigrationChannels[0]
	flushingChannel.State = v10.FLUSHING
	suite.setPreMigrationChannel(ctx, cdc, storeService, flushingChannel)

	flushCompleteChannel := preMigrationChannels[1]
	flushCompleteChannel.State = v10.FLUSHCOMPLETE
	suite.setPreMigrationChannel(ctx, cdc, storeService, flushCompleteChannel)

	err := v10.MigrateStore(ctx, storeService, cdc, channelKeeper)
	suite.Require().Errorf(err, "channel in state FLUSHING or FLUSHCOMPLETE found, to proceed with migration, please ensure no channels are currently upgrading")
}

// set up channels, upgrades, params, and prune sequences in the store,
// migrate the store, and assert that the channels have been upgraded and state removed as expected
func (suite *MigrationsV10TestSuite) TestMigrateStore() {
	ctx := suite.chainA.GetContext()
	cdc := suite.chainA.App.AppCodec()
	channelKeeper := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper
	storeService := runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey))
	store := storeService.OpenKVStore(ctx)
	numberOfChannels := 100

	for i := 0; i < numberOfChannels; i++ {
		path := ibctesting.NewPath(suite.chainA, suite.chainB)
		path.Setup()
	}

	preMigrationChannels := suite.getPreMigrationTypeChannels(ctx, cdc, storeService)
	suite.Require().Len(preMigrationChannels, numberOfChannels)

	// Set up some channels with old state
	testChannel1 := preMigrationChannels[0]
	testChannel2 := preMigrationChannels[1]

	// Set some upgrades
	upgrade := v10.Upgrade{
		Fields: v10.UpgradeFields{
			Ordering:       v10.ORDERED,
			ConnectionHops: []string{"connection-0"},
			Version:        testChannel1.Version,
		},
		Timeout:          v10.Timeout{},
		NextSequenceSend: 2,
	}
	err := store.Set(v10.ChannelUpgradeKey(testChannel1.PortId, testChannel1.ChannelId), cdc.MustMarshal(&upgrade))
	suite.Require().NoError(err)
	upgrade = v10.Upgrade{
		Fields: v10.UpgradeFields{
			Ordering:       v10.ORDERED,
			ConnectionHops: []string{"connection-0"},
			Version:        testChannel2.Version,
		},
		Timeout:          v10.Timeout{},
		NextSequenceSend: 20,
	}
	err = store.Set(v10.ChannelUpgradeKey(testChannel2.PortId, testChannel2.ChannelId), cdc.MustMarshal(&upgrade))
	suite.Require().NoError(err)

	counterpartyUpgrade := v10.Upgrade{
		Fields: v10.UpgradeFields{
			Ordering:       v10.ORDERED,
			ConnectionHops: []string{"connection-0"},
			Version:        testChannel2.Version,
		},
		Timeout:          v10.Timeout{},
		NextSequenceSend: 20,
	}
	err = store.Set(v10.ChannelCounterpartyUpgradeKey(testChannel2.PortId, testChannel2.ChannelId), cdc.MustMarshal(&counterpartyUpgrade))
	suite.Require().NoError(err)

	errorReceipt := v10.ErrorReceipt{
		Sequence: 3,
		Message:  "🤷",
	}
	err = store.Set(v10.ChannelUpgradeErrorKey(testChannel1.PortId, testChannel1.ChannelId), cdc.MustMarshal(&errorReceipt))
	suite.Require().NoError(err)

	// Set some params
	err = store.Set([]byte(v10.ParamsKey), cdc.MustMarshal(&v10.Params{UpgradeTimeout: v10.Timeout{
		Timestamp: 1000,
	}}))
	suite.Require().NoError(err)

	// Set some prune sequences
	err = store.Set(v10.PruningSequenceStartKey(testChannel1.PortId, testChannel1.ChannelId), sdk.Uint64ToBigEndian(0))
	suite.Require().NoError(err)
	err = store.Set(v10.PruningSequenceStartKey(testChannel2.PortId, testChannel2.ChannelId), sdk.Uint64ToBigEndian(42))
	suite.Require().NoError(err)

	err = v10.MigrateStore(ctx, storeService, cdc, channelKeeper)
	suite.Require().NoError(err)

	suite.assertChannelsUpgraded(ctx, suite.chainA.App.AppCodec(), storeService, channelKeeper, preMigrationChannels)
	suite.assertNoUpgrades(ctx, storeService)
	suite.assertNoParms(ctx, storeService)
	suite.assertNoPruneSequences(ctx, storeService)
}

func (suite *MigrationsV10TestSuite) setPreMigrationChannel(ctx sdk.Context, cdc codec.Codec, storeService corestore.KVStoreService, channel v10.IdentifiedChannel) {
	store := storeService.OpenKVStore(ctx)
	channelKey := host.ChannelKey(channel.PortId, channel.ChannelId)
	err := store.Set(channelKey, cdc.MustMarshal(&v10.Channel{
		State:           channel.State,
		Ordering:        channel.Ordering,
		Counterparty:    channel.Counterparty,
		ConnectionHops:  channel.ConnectionHops,
		Version:         channel.Version,
		UpgradeSequence: channel.UpgradeSequence,
	}))
	suite.Require().NoError(err)
}

func (suite *MigrationsV10TestSuite) getPreMigrationTypeChannels(ctx sdk.Context, cdc codec.Codec, storeService corestore.KVStoreService) []v10.IdentifiedChannel {
	var channels []v10.IdentifiedChannel

	iterator := storetypes.KVStorePrefixIterator(runtime.KVStoreAdapter(storeService.OpenKVStore(ctx)), []byte(host.KeyChannelEndPrefix))
	for ; iterator.Valid(); iterator.Next() {
		var channel v10.Channel
		err := cdc.Unmarshal(iterator.Value(), &channel)
		suite.Require().NoError(err)

		portID, channelID, err := host.ParseChannelPath(string(iterator.Key()))
		identifiedChannel := v10.IdentifiedChannel{
			State:           channel.State,
			Ordering:        channel.Ordering,
			Counterparty:    channel.Counterparty,
			ConnectionHops:  channel.ConnectionHops,
			Version:         channel.Version,
			PortId:          portID,
			ChannelId:       channelID,
			UpgradeSequence: channel.UpgradeSequence,
		}
		suite.Require().NoError(err)
		channels = append(channels, identifiedChannel)

	}
	iterator.Close()

	return channels
}

func (suite *MigrationsV10TestSuite) assertChannelsUpgraded(ctx sdk.Context, cdc codec.Codec, storeService corestore.KVStoreService, channelKeeper *keeper.Keeper, preMigrationChannels []v10.IdentifiedChannel) {
	// First check that all channels have gotten the old state pruned
	newChannelsWithPreMigrationType := suite.getPreMigrationTypeChannels(ctx, cdc, storeService)
	for _, channel := range newChannelsWithPreMigrationType {
		suite.Require().NotEqual(v10.FLUSHING, channel.State)
		suite.Require().NotEqual(v10.FLUSHCOMPLETE, channel.State)
		suite.Require().Equal(uint64(0), channel.UpgradeSequence)
	}

	// Then check that we can still receive all the channels
	newChannelsWithPostMigrationType := channelKeeper.GetAllChannels(ctx)
	for _, channel := range newChannelsWithPostMigrationType {
		suite.Require().NoError(channel.ValidateBasic())
	}

	suite.Require().Equal(len(newChannelsWithPreMigrationType), len(newChannelsWithPostMigrationType))
	suite.Require().Equal(len(newChannelsWithPostMigrationType), len(preMigrationChannels))
}

func (suite *MigrationsV10TestSuite) assertNoUpgrades(ctx sdk.Context, storeService corestore.KVStoreService) {
	store := storeService.OpenKVStore(ctx)
	suite.Require().False(store.Has([]byte(v10.KeyChannelUpgradePrefix)))
}

func (suite *MigrationsV10TestSuite) assertNoParms(ctx sdk.Context, storeService corestore.KVStoreService) {
	store := storeService.OpenKVStore(ctx)
	suite.Require().False(store.Has([]byte(v10.ParamsKey)))
}

func (suite *MigrationsV10TestSuite) assertNoPruneSequences(ctx sdk.Context, storeService corestore.KVStoreService) {
	store := storeService.OpenKVStore(ctx)
	suite.Require().False(store.Has([]byte(v10.KeyPruningSequenceStart)))
}
