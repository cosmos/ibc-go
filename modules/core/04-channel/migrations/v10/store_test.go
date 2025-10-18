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
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v10"
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

func TestMigrationsV10TestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsV10TestSuite))
}

func (s *MigrationsV10TestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

// set up channels that are still in upgrade state, and assert that the upgrade fails.
// migrate the store, and assert that the channels have been upgraded and state removed as expected
func (s *MigrationsV10TestSuite) TestMigrateStoreWithUpgradingChannels() {
	ctx := s.chainA.GetContext()
	cdc := s.chainA.App.AppCodec()
	channelKeeper := s.chainA.GetSimApp().IBCKeeper.ChannelKeeper
	storeService := runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey))

	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()
	path = ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()

	preMigrationChannels := s.getPreMigrationTypeChannels(ctx, cdc, storeService)
	s.Require().Len(preMigrationChannels, 2)

	// Set up some channels with old state
	flushingChannel := preMigrationChannels[0]
	flushingChannel.State = v10.FLUSHING
	s.setPreMigrationChannel(ctx, cdc, storeService, flushingChannel)

	flushCompleteChannel := preMigrationChannels[1]
	flushCompleteChannel.State = v10.FLUSHCOMPLETE
	s.setPreMigrationChannel(ctx, cdc, storeService, flushCompleteChannel)

	err := v10.MigrateStore(ctx, storeService, cdc, channelKeeper)
	s.Require().Errorf(err, "channel in state FLUSHING or FLUSHCOMPLETE found, to proceed with migration, please ensure no channels are currently upgrading")
}

// set up channels, upgrades, params, and prune sequences in the store,
// migrate the store, and assert that the channels have been upgraded and state removed as expected
func (s *MigrationsV10TestSuite) TestMigrateStore() {
	ctx := s.chainA.GetContext()
	cdc := s.chainA.App.AppCodec()
	channelKeeper := s.chainA.GetSimApp().IBCKeeper.ChannelKeeper
	storeService := runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey))
	store := storeService.OpenKVStore(ctx)
	numberOfChannels := 100

	for range numberOfChannels {
		path := ibctesting.NewPath(s.chainA, s.chainB)
		path.Setup()
	}

	preMigrationChannels := s.getPreMigrationTypeChannels(ctx, cdc, storeService)
	s.Require().Len(preMigrationChannels, numberOfChannels)

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
	s.Require().NoError(err)
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
	s.Require().NoError(err)

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
	s.Require().NoError(err)

	errorReceipt := v10.ErrorReceipt{
		Sequence: 3,
		Message:  "🤷",
	}
	err = store.Set(v10.ChannelUpgradeErrorKey(testChannel1.PortId, testChannel1.ChannelId), cdc.MustMarshal(&errorReceipt))
	s.Require().NoError(err)

	// Set some params
	err = store.Set([]byte(v10.ParamsKey), cdc.MustMarshal(&v10.Params{UpgradeTimeout: v10.Timeout{
		Timestamp: 1000,
	}}))
	s.Require().NoError(err)

	// Set some prune sequences
	err = store.Set(v10.PruningSequenceStartKey(testChannel1.PortId, testChannel1.ChannelId), sdk.Uint64ToBigEndian(0))
	s.Require().NoError(err)
	err = store.Set(v10.PruningSequenceStartKey(testChannel2.PortId, testChannel2.ChannelId), sdk.Uint64ToBigEndian(42))
	s.Require().NoError(err)

	err = v10.MigrateStore(ctx, storeService, cdc, channelKeeper)
	s.Require().NoError(err)

	s.assertChannelsUpgraded(ctx, s.chainA.App.AppCodec(), storeService, channelKeeper, preMigrationChannels)
	s.assertNoUpgrades(ctx, storeService)
	s.assertNoParms(ctx, storeService)
	s.assertNoPruneSequences(ctx, storeService)
}

func (s *MigrationsV10TestSuite) setPreMigrationChannel(ctx sdk.Context, cdc codec.Codec, storeService corestore.KVStoreService, channel v10.IdentifiedChannel) {
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
	s.Require().NoError(err)
}

func (s *MigrationsV10TestSuite) getPreMigrationTypeChannels(ctx sdk.Context, cdc codec.Codec, storeService corestore.KVStoreService) []v10.IdentifiedChannel {
	var channels []v10.IdentifiedChannel

	iterator := storetypes.KVStorePrefixIterator(runtime.KVStoreAdapter(storeService.OpenKVStore(ctx)), []byte(host.KeyChannelEndPrefix))
	for ; iterator.Valid(); iterator.Next() {
		var channel v10.Channel
		err := cdc.Unmarshal(iterator.Value(), &channel)
		s.Require().NoError(err)

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
		s.Require().NoError(err)
		channels = append(channels, identifiedChannel)
	}
	iterator.Close()

	return channels
}

func (s *MigrationsV10TestSuite) assertChannelsUpgraded(ctx sdk.Context, cdc codec.Codec, storeService corestore.KVStoreService, channelKeeper *keeper.Keeper, preMigrationChannels []v10.IdentifiedChannel) {
	// First check that all channels have gotten the old state pruned
	newChannelsWithPreMigrationType := s.getPreMigrationTypeChannels(ctx, cdc, storeService)
	for _, channel := range newChannelsWithPreMigrationType {
		s.Require().NotEqual(v10.FLUSHING, channel.State)
		s.Require().NotEqual(v10.FLUSHCOMPLETE, channel.State)
		s.Require().Equal(uint64(0), channel.UpgradeSequence)
	}

	// Then check that we can still receive all the channels
	newChannelsWithPostMigrationType := channelKeeper.GetAllChannels(ctx)
	for _, channel := range newChannelsWithPostMigrationType {
		s.Require().NoError(channel.ValidateBasic())
	}

	s.Require().Len(newChannelsWithPostMigrationType, len(newChannelsWithPreMigrationType))
	s.Require().Len(preMigrationChannels, len(newChannelsWithPostMigrationType))
}

func (s *MigrationsV10TestSuite) assertNoUpgrades(ctx sdk.Context, storeService corestore.KVStoreService) {
	store := storeService.OpenKVStore(ctx)
	s.Require().False(store.Has([]byte(v10.KeyChannelUpgradePrefix)))
}

func (s *MigrationsV10TestSuite) assertNoParms(ctx sdk.Context, storeService corestore.KVStoreService) {
	store := storeService.OpenKVStore(ctx)
	s.Require().False(store.Has([]byte(v10.ParamsKey)))
}

func (s *MigrationsV10TestSuite) assertNoPruneSequences(ctx sdk.Context, storeService corestore.KVStoreService) {
	store := storeService.OpenKVStore(ctx)
	s.Require().False(store.Has([]byte(v10.KeyPruningSequenceStart)))
}
