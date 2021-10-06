package keeper_test

import (
	"sort"
	"time"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	childtypes "github.com/cosmos/ibc-go/modules/apps/ccv/child/types"
	parenttypes "github.com/cosmos/ibc-go/modules/apps/ccv/parent/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	// setup CCV channel
	suite.SetupCCVChannel()

	pk1, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	suite.Require().NoError(err)
	pk2, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	suite.Require().NoError(err)
	pk3, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	suite.Require().NoError(err)

	changes1 := []abci.ValidatorUpdate{
		{
			PubKey: pk1,
			Power:  30,
		},
		{
			PubKey: pk2,
			Power:  20,
		},
	}

	changes2 := []abci.ValidatorUpdate{
		{
			PubKey: pk2,
			Power:  40,
		},
		{
			PubKey: pk3,
			Power:  10,
		},
	}

	pd := types.NewValidatorSetChangePacketData(
		changes1,
	)

	pd2 := types.NewValidatorSetChangePacketData(
		changes2,
	)

	testCases := []struct {
		name                   string
		packet                 channeltypes.Packet
		newChanges             types.ValidatorSetChangePacketData
		expectedPendingChanges types.ValidatorSetChangePacketData
		expErrorAck            bool
	}{
		{
			"success on first packet",
			channeltypes.NewPacket(pd.GetBytes(), 1, parenttypes.PortID, suite.path.EndpointB.ChannelID, childtypes.PortID, suite.path.EndpointA.ChannelID,
				clienttypes.NewHeight(1, 0), 0),
			types.ValidatorSetChangePacketData{ValidatorUpdates: changes1},
			types.ValidatorSetChangePacketData{ValidatorUpdates: changes1},
			false,
		},
		{
			"success on subsequent packet",
			channeltypes.NewPacket(pd.GetBytes(), 2, parenttypes.PortID, suite.path.EndpointB.ChannelID, childtypes.PortID, suite.path.EndpointA.ChannelID,
				clienttypes.NewHeight(1, 0), 0),
			types.ValidatorSetChangePacketData{ValidatorUpdates: changes1},
			types.ValidatorSetChangePacketData{ValidatorUpdates: changes1},
			false,
		},
		{
			"success on packet with more changes",
			channeltypes.NewPacket(pd2.GetBytes(), 3, parenttypes.PortID, suite.path.EndpointB.ChannelID, childtypes.PortID, suite.path.EndpointA.ChannelID,
				clienttypes.NewHeight(1, 0), 0),
			types.ValidatorSetChangePacketData{ValidatorUpdates: changes2},
			types.ValidatorSetChangePacketData{ValidatorUpdates: []abci.ValidatorUpdate{
				{
					PubKey: pk1,
					Power:  30,
				},
				{
					PubKey: pk2,
					Power:  40,
				},
				{
					PubKey: pk3,
					Power:  10,
				},
			}},
			false,
		},
		{
			"invalid packet: different destination channel than parent channel",
			channeltypes.NewPacket(pd.GetBytes(), 1, parenttypes.PortID, suite.path.EndpointB.ChannelID, childtypes.PortID, "InvalidChannel",
				clienttypes.NewHeight(1, 0), 0),
			types.ValidatorSetChangePacketData{ValidatorUpdates: []abci.ValidatorUpdate{}},
			types.ValidatorSetChangePacketData{ValidatorUpdates: []abci.ValidatorUpdate{}},
			true,
		},
	}

	for _, tc := range testCases {
		ack := suite.childChain.GetSimApp().ChildKeeper.OnRecvPacket(suite.ctx, tc.packet, tc.newChanges)

		if tc.expErrorAck {
			suite.Require().NotNil(ack, "invalid test case: %s did not return ack", tc.name)
			suite.Require().False(ack.Success(), "invalid test case: %s did not return an Error Acknowledgment")
		} else {
			suite.Require().Nil(ack, "successful packet must send ack asynchronously. case: %s", tc.name)
			suite.Require().Equal(ccv.VALIDATING, suite.childChain.GetSimApp().ChildKeeper.GetChannelStatus(suite.ctx, suite.path.EndpointA.ChannelID),
				"channel status is not valdidating after receive packet for valid test case: %s", tc.name)
			parentChannel, ok := suite.childChain.GetSimApp().ChildKeeper.GetParentChannel(suite.ctx)
			suite.Require().True(ok)
			suite.Require().Equal(tc.packet.DestinationChannel, parentChannel,
				"parent channel is not destination channel on successful receive for valid test case: %s", tc.name)

			// Check that pending changes are accumulated and stored correctly
			actualPendingChanges, ok := suite.childChain.GetSimApp().ChildKeeper.GetPendingChanges(suite.ctx)
			suite.Require().True(ok)
			// Sort to avoid dumb inequalities
			sort.SliceStable(actualPendingChanges.ValidatorUpdates, func(i, j int) bool {
				return actualPendingChanges.ValidatorUpdates[i].PubKey.Compare(actualPendingChanges.ValidatorUpdates[j].PubKey) == -1
			})
			sort.SliceStable(tc.expectedPendingChanges.ValidatorUpdates, func(i, j int) bool {
				return tc.expectedPendingChanges.ValidatorUpdates[i].PubKey.Compare(tc.expectedPendingChanges.ValidatorUpdates[j].PubKey) == -1
			})
			suite.Require().Equal(tc.expectedPendingChanges, *actualPendingChanges, "pending changes not equal to expected changes after successful packet receive. case: %s", tc.name)

			expectedTime := uint64(suite.ctx.BlockTime().Add(childtypes.UnbondingTime).UnixNano())
			unbondingTime := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, tc.packet.Sequence)
			suite.Require().Equal(expectedTime, unbondingTime, "unbonding time has unexpected value for case: %s", tc.name)
			unbondingPacket, err := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingPacket(suite.ctx, tc.packet.Sequence)
			suite.Require().NoError(err)
			suite.Require().Equal(&tc.packet, unbondingPacket, "packet is not added to unbonding queue after successful receive. case: %s", tc.name)
		}
	}
}

func (suite *KeeperTestSuite) TestUnbondMaturePackets() {
	// setup CCV channel
	suite.SetupCCVChannel()

	// send 3 packets to child chain at different times
	pk1, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	suite.Require().NoError(err)
	pk2, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	suite.Require().NoError(err)

	pd := types.NewValidatorSetChangePacketData(
		[]abci.ValidatorUpdate{
			{
				PubKey: pk1,
				Power:  30,
			},
			{
				PubKey: pk2,
				Power:  20,
			},
		},
	)

	origTime := suite.ctx.BlockTime()

	// send first packet
	packet := channeltypes.NewPacket(pd.GetBytes(), 1, parenttypes.PortID, suite.path.EndpointB.ChannelID, childtypes.PortID, suite.path.EndpointA.ChannelID,
		clienttypes.NewHeight(1, 0), 0)
	ack := suite.childChain.GetSimApp().ChildKeeper.OnRecvPacket(suite.ctx, packet, pd)
	suite.Require().Nil(ack)

	// update time and send second packet
	suite.ctx = suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(time.Hour))
	pd.ValidatorUpdates[0].Power = 15
	packet.Data = pd.GetBytes()
	packet.Sequence = 2
	ack = suite.childChain.GetSimApp().ChildKeeper.OnRecvPacket(suite.ctx, packet, pd)
	suite.Require().Nil(ack)

	// update time and send third packet
	suite.ctx = suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(24 * time.Hour))
	pd.ValidatorUpdates[1].Power = 40
	packet.Data = pd.GetBytes()
	packet.Sequence = 3
	ack = suite.childChain.GetSimApp().ChildKeeper.OnRecvPacket(suite.ctx, packet, pd)
	suite.Require().Nil(ack)

	// move ctx time forward such that first two packets are unbonded but third is not.
	suite.ctx = suite.ctx.WithBlockTime(origTime.Add(childtypes.UnbondingTime).Add(3 * time.Hour))

	suite.childChain.GetSimApp().ChildKeeper.UnbondMaturePackets(suite.ctx)

	// ensure first two packets are unbonded and acknowledgement is written
	// unbonded time is deleted
	time1 := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, 1)
	time2 := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, 2)
	suite.Require().Equal(uint64(0), time1, "unbonding time not deleted for mature packet 1")
	suite.Require().Equal(uint64(0), time2, "unbonding time not deleted for mature packet 2")

	// unbonded packets are deleted
	_, err = suite.childChain.GetSimApp().ChildKeeper.GetUnbondingPacket(suite.ctx, 1)
	suite.Require().Error(err, "retrieved unbonding packet for matured packet 1")
	_, err = suite.childChain.GetSimApp().ChildKeeper.GetUnbondingPacket(suite.ctx, 2)
	suite.Require().Error(err, "retrieved unbonding packet for matured packet 1")

	expectedWriteAckBytes := channeltypes.CommitAcknowledgement(channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement())

	// successful acknowledgements are written
	ackBytes1, ok := suite.childChain.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.ctx, childtypes.PortID, suite.path.EndpointA.ChannelID, 1)
	suite.Require().True(ok)
	suite.Require().Equal(expectedWriteAckBytes, ackBytes1, "did not write successful ack for matue packet 1")
	ackBytes2, ok := suite.childChain.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.ctx, childtypes.PortID, suite.path.EndpointA.ChannelID, 2)
	suite.Require().True(ok)
	suite.Require().Equal(expectedWriteAckBytes, ackBytes2, "did not write successful ack for matue packet 1")

	// ensure that third packet did not get ack written and is still in store
	time3 := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, 3)
	suite.Require().True(time3 > uint64(suite.ctx.BlockTime().UnixNano()), "Unbonding time for packet 3 is not after current time")
	packet3, err := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingPacket(suite.ctx, 3)
	suite.Require().NoError(err, "retrieving unbonding packet 3 returned error")
	suite.Require().Equal(&packet, packet3, "unbonding packet 3 has unexpected value")

	// ensure acknowledgement has not been written for unbonding packet
	ackBytes3, ok := suite.childChain.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.ctx, childtypes.PortID, suite.path.EndpointA.ChannelID, 3)
	suite.Require().False(ok)
	suite.Require().Nil(ackBytes3, "acknowledgement written for unbonding packet 3")

}
