package v11_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v11"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channelv2types "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
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

func (s *MigrationsV11TestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestMigrationsV11TestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsV11TestSuite))
}

func (s *MigrationsV11TestSuite) TestMigrateStore() {
	ctx := s.chainA.GetContext()
	cdc := s.chainA.App.AppCodec()
	ibcKeeper := s.chainA.App.GetIBCKeeper()
	storeService := runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey))
	store := storeService.OpenKVStore(ctx)
	numberOfChannels := 100

	for i := range numberOfChannels {
		path := ibctesting.NewPath(s.chainA, s.chainB)
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
		s.Require().True(ok)
		err := store.Delete(hostv2.NextSequenceSendKey(path.EndpointA.ChannelID))
		s.Require().NoError(err)
		err = store.Set(v11.NextSequenceSendV1Key(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), sdk.Uint64ToBigEndian(seq))
		s.Require().NoError(err)

		// Remove counterparty to mock pre migration channels
		clientStore := ibcKeeper.ClientKeeper.ClientStore(ctx, path.EndpointA.ChannelID)
		clientStore.Delete(clientv2types.CounterpartyKey())

		// Remove alias to mock pre migration channels
		err = store.Delete(channelv2types.AliasKey(path.EndpointA.ChannelID))
		s.Require().NoError(err)

		if i%5 == 0 {
			channel, ok := ibcKeeper.ChannelKeeper.GetChannel(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			s.Require().True(ok)
			if i%2 == 0 {
				channel.State = types.INIT
			} else {
				channel.State = types.CLOSED
			}
			ibcKeeper.ChannelKeeper.SetChannel(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
		}
	}

	err := v11.MigrateStore(ctx, storeService, cdc, ibcKeeper)
	s.Require().NoError(err)

	for i := range numberOfChannels {
		channelID := types.FormatChannelIdentifier(uint64(i))
		channel, ok := ibcKeeper.ChannelKeeper.GetChannel(ctx, mock.PortID, channelID)
		s.Require().True(ok, i)

		if channel.Ordering == types.UNORDERED && channel.State == types.OPEN {
			// ensure counterparty set
			expCounterparty, ok := ibcKeeper.ChannelKeeper.GetV2Counterparty(ctx, mock.PortID, channelID)
			s.Require().True(ok)
			counterparty, ok := ibcKeeper.ClientV2Keeper.GetClientCounterparty(ctx, channelID)
			s.Require().True(ok)
			s.Require().Equal(expCounterparty, counterparty, "counterparty not set correctly")

			// ensure base client mapping set
			baseClientID, ok := ibcKeeper.ChannelKeeperV2.GetClientForAlias(ctx, channelID)
			s.Require().True(ok)
			s.Require().NotEqual(channelID, baseClientID)
			connection, ok := ibcKeeper.ConnectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
			s.Require().True(ok)
			s.Require().Equal(connection.ClientId, baseClientID, "base client mapping not set correctly")
		} else {
			// ensure counterparty not set for closed channels
			_, ok := ibcKeeper.ClientV2Keeper.GetClientCounterparty(ctx, channelID)
			s.Require().False(ok, "counterparty should not be set for closed channels")

			// ensure base client mapping not set for closed channels
			baseClientID, ok := ibcKeeper.ChannelKeeperV2.GetClientForAlias(ctx, channelID)
			s.Require().False(ok)
			s.Require().Empty(baseClientID, "base client mapping should not be set for closed channels")
		}

		// ensure that sequence migrated correctly
		bz, _ := store.Get(v11.NextSequenceSendV1Key(mock.PortID, channelID))
		s.Require().Nil(bz)
		seq, ok := ibcKeeper.ChannelKeeper.GetNextSequenceSend(ctx, mock.PortID, channelID)
		s.Require().True(ok)
		s.Require().Equal(uint64(1), seq)
	}
}
