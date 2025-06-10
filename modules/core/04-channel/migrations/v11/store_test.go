package v11_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	v11 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v11"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/mock"
)

type MigrationsV11TestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *MigrationsV11TestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestMigrationsV11TestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsV11TestSuite))
}

func (suite *MigrationsV11TestSuite) TestMigrateStore() {
	ctx := suite.chainA.GetContext()
	cdc := suite.chainA.App.AppCodec()
	ibcKeeper := suite.chainA.App.GetIBCKeeper()
	storeService := runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey))
	store := storeService.OpenKVStore(ctx)
	numberOfChannels := 100

	for i := range numberOfChannels {
		path := ibctesting.NewPath(suite.chainA, suite.chainB)
		// needed to add this line to have channel ids increment correctly
		// without this line, the channel ids skip a number in the sequence
		path = path.DisableUniqueChannelIDs()
		if i%2 == 0 {
			path.SetChannelOrdered()
		}
		path.Setup()

		// Move sequence back to its old v1 format key
		// to mock channels that were created before the new changes
		seq, ok := ibcKeeper.ChannelKeeper.GetNextSequenceSend(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		suite.Require().True(ok)
		store.Delete(hostv2.NextSequenceSendKey(path.EndpointA.ChannelID))
		store.Set(v11.NextSequenceSendKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), sdk.Uint64ToBigEndian(seq))

		if i%5 == 0 {
			channel, ok := ibcKeeper.ChannelKeeper.GetChannel(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			suite.Require().True(ok)
			channel.State = types.CLOSED
			ibcKeeper.ChannelKeeper.SetChannel(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
		}
	}

	err := v11.MigrateStore(ctx, storeService, cdc, ibcKeeper)
	suite.Require().NoError(err)

	for i := range numberOfChannels {
		channelId := types.FormatChannelIdentifier(uint64(i))
		channel, ok := ibcKeeper.ChannelKeeper.GetChannel(ctx, mock.PortID, channelId)
		suite.Require().True(ok, i)

		if channel.Ordering == types.UNORDERED && channel.State == types.OPEN {
			// ensure counterparty set
			expCounterparty, ok := ibcKeeper.ChannelKeeper.GetV2Counterparty(ctx, mock.PortID, channelId)
			suite.Require().True(ok)
			counterparty, ok := ibcKeeper.ClientV2Keeper.GetClientCounterparty(ctx, channelId)
			suite.Require().True(ok)
			suite.Require().Equal(expCounterparty, counterparty, "counterparty not set correctly")

			// ensure base client mapping set
			baseClientId, ok := ibcKeeper.ChannelKeeperV2.GetClientForAlias(ctx, channelId)
			suite.Require().True(ok)
			suite.Require().NotEqual(channelId, baseClientId)
			connection, ok := ibcKeeper.ConnectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
			suite.Require().True(ok)
			suite.Require().Equal(connection.ClientId, baseClientId, "base client mapping not set correctly")
		} else {
			// ensure counterparty not set for closed channels
			_, ok := ibcKeeper.ClientV2Keeper.GetClientCounterparty(ctx, channelId)
			suite.Require().False(ok, "counterparty should not be set for closed channels")

			// ensure base client mapping not set for closed channels
			baseClientId, ok := ibcKeeper.ChannelKeeperV2.GetClientForAlias(ctx, channelId)
			suite.Require().False(ok)
			suite.Require().Equal("", baseClientId, "base client mapping should not be set for closed channels")
		}

		// ensure that sequence migrated correctly
		bz, _ := store.Get(v11.NextSequenceSendKey(mock.PortID, channelId))
		suite.Require().Nil(bz)
		seq, ok := ibcKeeper.ChannelKeeper.GetNextSequenceSend(ctx, mock.PortID, channelId)
		suite.Require().True(ok)
		suite.Require().Equal(uint64(1), seq)

	}
}
