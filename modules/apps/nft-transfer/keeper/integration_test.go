package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/x/nft"
	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/stretchr/testify/suite"
)

func TestKeeperTestSuite1(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestSendAndReceive() {
	path := NewTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)
	suite.coordinator.CreateChannels(path)

	classID := "cryptoCat"
	classURI := "cat_uri"
	nftID := "kitty"
	nftURI := "kittt_uri"

	//============================== setup start===============================
	nftKeeper := path.EndpointA.Chain.GetSimApp().NFTKeeper
	err := nftKeeper.SaveClass(path.EndpointA.Chain.GetContext(), nft.Class{
		Id:  classID,
		Uri: classURI,
	})
	suite.Require().NoError(err, "SaveClass error")

	err = nftKeeper.Mint(path.EndpointA.Chain.GetContext(), nft.NFT{
		ClassId: classID,
		Id:      nftID,
		Uri:     nftURI,
	}, path.EndpointA.Chain.SenderAccount.GetAddress())
	//============================== setup end===============================

	// transfer from chainA to chainB
	{
		msgTransfer := &types.MsgTransfer{
			SourcePort:    path.EndpointA.ChannelConfig.PortID,
			SourceChannel: path.EndpointA.ChannelID,
			ClassId:       classID,
			TokenIds:      []string{nftID},
			Sender:        path.EndpointA.Chain.SenderAccount.GetAddress().String(),
			Receiver:      path.EndpointB.Chain.SenderAccount.GetAddress().String(),
			TimeoutHeight: clienttypes.Height{
				RevisionNumber: 0,
				RevisionHeight: 100,
			},
			TimeoutTimestamp: 0,
		}

		_, err = suite.chainA.SendMsgs(msgTransfer)
		suite.Require().NoError(err)

		suite.Require().Equal(
			types.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID),
			path.EndpointA.Chain.GetSimApp().NFTKeeper.GetOwner(path.EndpointA.Chain.GetContext(), classID, nftID),
			"escrow nft failed",
		)
	}

	// receive nft on chainB from chainA
	{
		nonFungibleTokenPacket := types.NewNonFungibleTokenPacketData(
			classID,
			classURI,
			[]string{nftID},
			[]string{nftURI},
			suite.chainA.SenderAccount.GetAddress().String(),
			suite.chainB.SenderAccount.GetAddress().String(),
		)

		packet := channeltypes.NewPacket(
			nonFungibleTokenPacket.GetBytes(),
			1,
			path.EndpointA.ChannelConfig.PortID,
			path.EndpointA.ChannelID,
			path.EndpointB.ChannelConfig.PortID,
			path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100),
			0,
		)

		// get proof of packet commitment from chainA
		err = path.EndpointB.UpdateClient()
		suite.Require().NoError(err)

		packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
		proof, proofHeight := path.EndpointA.QueryProof(packetKey)

		recvMsg := channeltypes.NewMsgRecvPacket(
			packet, proof, proofHeight, suite.chainB.SenderAccount.GetAddress().String())
		_, err = suite.chainB.SendMsgs(recvMsg)
		suite.Require().NoError(err) // message committed

		prefixedClassID := types.GetClassPrefix(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID) + classID
		trace := types.ParseClassTrace(prefixedClassID)

		class, found := path.EndpointB.Chain.GetSimApp().
			NFTKeeper.GetClass(path.EndpointB.Chain.GetContext(), trace.IBCClassID())
		suite.Require().True(found, "not found class")
		suite.Require().Equal(nft.Class{Id: trace.IBCClassID(), Uri: classURI}, class, "class not equal")

		token, found := path.EndpointB.Chain.GetSimApp().
			NFTKeeper.GetNFT(path.EndpointB.Chain.GetContext(), trace.IBCClassID(), nftID)
		suite.Require().True(found, "not found class")
		suite.Require().Equal(
			nft.NFT{ClassId: trace.IBCClassID(), Id: nftID, Uri: nftURI},
			token,
			"nft not equal",
		)

	}

}
