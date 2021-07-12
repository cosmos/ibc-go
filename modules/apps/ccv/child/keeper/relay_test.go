package keeper_test

import (
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

	// setup CCV channel
	suite.SetupCCVChannel()

	packet := channeltypes.NewPacket(pd.GetBytes(), 1, parenttypes.PortID, suite.path.EndpointB.ChannelID, childtypes.PortID, suite.path.EndpointA.ChannelID,
		clienttypes.NewHeight(1, 0), 0)

	testCases := []struct {
		name           string
		malleatePacket func()
		expErrorAck    bool
	}{
		{
			"success on first packet",
			func() {},
			false,
		},
		{
			"success on subsequent packet",
			func() {
				packet.Sequence = 2
			},
			false,
		},
		{
			"invalid packet: different destination channel than parent channel",
			func() {
				packet.Sequence = 1
				// change destination channel to different channelID than parent channel
				packet.DestinationChannel = "invalidChannel"
			},
			true,
		},
	}

	for _, tc := range testCases {
		// malleate packet for each case
		tc.malleatePacket()

		ack, err := suite.childChain.GetSimApp().ChildKeeper.OnRecvPacket(suite.ctx, packet, pd)

		if tc.expErrorAck {
			suite.Require().NotNil(ack, "invalid test case: %s did not return ack", tc.name)
			suite.Require().False(ack.Success(), "invalid test case: %s did not return an Error Acknowledgment")
			suite.Require().Nil(err, "returned error unexpectedly. should be nil to commit RecvPacket callback changes")
		} else {
			suite.Require().Equal(ccv.Validating, suite.childChain.GetSimApp().ChildKeeper.GetChannelStatus(suite.ctx, suite.path.EndpointA.ChannelID),
				"channel status is not valdidating after receive packet for valid test case: %s", tc.name)
			parentChannel, ok := suite.childChain.GetSimApp().ChildKeeper.GetParentChannel(suite.ctx)
			suite.Require().True(ok)
			suite.Require().Equal(packet.DestinationChannel, parentChannel,
				"parent channel is not destination channel on successful receive for valid test case: %s", tc.name)
			actualPd, ok := suite.childChain.GetSimApp().ChildKeeper.GetPendingChanges(suite.ctx)
			suite.Require().True(ok)
			suite.Require().Equal(&pd, actualPd, "pending changes not equal to packet data after successful packet receive. case: %s", tc.name)
			expectedTime := uint64(suite.ctx.BlockTime().Add(childtypes.UnbondingTime).UnixNano())
			unbondingTime := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, packet.Sequence)
			suite.Require().Equal(expectedTime, unbondingTime, "unbonding time has unexpected value for case: %s", tc.name)
			unbondingPacket, err := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingPacket(suite.ctx, packet.Sequence)
			suite.Require().NoError(err)
			suite.Require().Equal(&packet, unbondingPacket, "packet is not added to unbonding queue after successful receive. case: %s", tc.name)
		}
	}
}
