package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/x/nft"
	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
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

	var targetClassID string
	var packet channeltypes.Packet

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

	suite.Run("transfer from chainA to chainB", func() {
		{
			packet = suite.transferNFT(
				path.EndpointA,
				path.EndpointB,
				classID,
				nftID,
				path.EndpointA.Chain.SenderAccount.GetAddress().String(),
				path.EndpointB.Chain.SenderAccount.GetAddress().String(),
				true,
			)
		}
	})

	suite.Run("receive on chainB from chainA", func() {
		{
			targetClassID = suite.receiverNFT(
				path.EndpointA,
				path.EndpointB,
				packet,
				true,
			)
		}
	})

	// transfer from chainB to chainC
	path1 := NewTransferPath(suite.chainB, suite.chainC)
	suite.Run("transfer from chainB to chainC", func() {
		{
			suite.coordinator.SetupConnections(path1)
			suite.coordinator.CreateChannels(path1)

			packet = suite.transferNFT(
				path1.EndpointA,
				path1.EndpointB,
				targetClassID,
				nftID,
				path.EndpointB.Chain.SenderAccount.GetAddress().String(),
				path1.EndpointB.Chain.SenderAccount.GetAddress().String(),
				true,
			)
		}
	})

	suite.Run("receive on chainC from chainB", func() {
		{
			targetClassID = suite.receiverNFT(
				path1.EndpointA,
				path1.EndpointB,
				packet,
				true,
			)
		}
	})

	suite.Run("transfer from chainC back to chainB", func() {
		{
			packet = suite.transferNFT(
				path1.EndpointB,
				path1.EndpointA,
				targetClassID,
				nftID,
				path1.EndpointB.Chain.SenderAccount.GetAddress().String(),
				path1.EndpointA.Chain.SenderAccount.GetAddress().String(),
				false,
			)
		}
	})

	suite.Run("receive on chainB from chainC", func() {
		{
			targetClassID = suite.receiverNFT(
				path1.EndpointB,
				path1.EndpointA,
				packet,
				false,
			)
		}
	})

	suite.Run("transfer from chainB back to chainA", func() {
		{
			packet = suite.transferNFT(
				path.EndpointB,
				path.EndpointA,
				targetClassID,
				nftID,
				path1.EndpointA.Chain.SenderAccount.GetAddress().String(),
				path.EndpointA.Chain.SenderAccount.GetAddress().String(),
				false,
			)
		}
	})

	suite.Run("receive on chainA from chainB", func() {
		{
			targetClassID = suite.receiverNFT(
				path.EndpointB,
				path.EndpointA,
				packet,
				false,
			)

			suite.Equal(classID, targetClassID, "wrong classID")
		}
	})
}

func (suite *KeeperTestSuite) transferNFT(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	classID, nftID string,
	sender, receiver string,
	isAwayFromOrigin bool,
) channeltypes.Packet {
	msgTransfer := &types.MsgTransfer{
		SourcePort:    fromEndpoint.ChannelConfig.PortID,
		SourceChannel: fromEndpoint.ChannelID,
		ClassId:       classID,
		TokenIds:      []string{nftID},
		Sender:        sender,
		Receiver:      receiver,
		TimeoutHeight: clienttypes.Height{
			RevisionNumber: 0,
			RevisionHeight: 100,
		},
		TimeoutTimestamp: 0,
	}

	res, err := fromEndpoint.Chain.SendMsgs(msgTransfer)
	suite.Require().NoError(err)

	//check escrow token
	if isAwayFromOrigin {
		suite.Require().Equal(
			types.GetEscrowAddress(fromEndpoint.ChannelConfig.PortID, fromEndpoint.ChannelID),
			fromEndpoint.Chain.GetSimApp().NFTKeeper.GetOwner(fromEndpoint.Chain.GetContext(), classID, nftID),
			"escrow nft failed",
		)
	} else {
		suite.Require().False(
			fromEndpoint.Chain.GetSimApp().NFTKeeper.HasNFT(fromEndpoint.Chain.GetContext(), classID, nftID),
			"burn nft failed",
		)
	}

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)
	return packet

}

func (suite *KeeperTestSuite) receiverNFT(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	packet channeltypes.Packet,
	isAwayFromOrigin bool,
) string {

	var data types.NonFungibleTokenPacketData
	err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data)
	suite.Require().NoError(err)

	// get proof of packet commitment from chainA
	err = toEndpoint.UpdateClient()
	suite.Require().NoError(err)

	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := fromEndpoint.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(
		packet, proof, proofHeight, toEndpoint.Chain.SenderAccount.GetAddress().String())
	_, err = toEndpoint.Chain.SendMsgs(recvMsg)
	suite.Require().NoError(err) // message committed

	var classID string

	if isAwayFromOrigin {
		//construct classTrace
		prefixedClassID := types.GetClassPrefix(toEndpoint.ChannelConfig.PortID, toEndpoint.ChannelID) + data.GetClassId()
		trace := types.ParseClassTrace(prefixedClassID)
		classID = trace.IBCClassID()

		// check class
		class, found := toEndpoint.Chain.GetSimApp().
			NFTKeeper.GetClass(toEndpoint.Chain.GetContext(), classID)
		suite.Require().True(found, "not found class")
		suite.Require().Equal(nft.Class{Id: classID, Uri: data.GetClassUri()}, class, "class not equal")

		// check nft
		token, found := toEndpoint.Chain.GetSimApp().
			NFTKeeper.GetNFT(toEndpoint.Chain.GetContext(), classID, data.GetTokenIds()[0])
		suite.Require().True(found, "not found class")
		suite.Require().Equal(
			nft.NFT{ClassId: classID, Id: data.GetTokenIds()[0], Uri: data.GetTokenUris()[0]},
			token,
			"nft not equal",
		)
	} else {
		unprefixedClassID := types.RemoveClassPrefix(packet.GetSourcePort(),
			packet.GetSourceChannel(), data.ClassId)
		classID = types.ParseClassTrace(unprefixedClassID).IBCClassID()

		suite.Require().Equal(
			data.GetReceiver(),
			toEndpoint.Chain.GetSimApp().
				NFTKeeper.GetOwner(toEndpoint.Chain.GetContext(), classID, data.GetTokenIds()[0]).String(),
			"nft not equal",
		)
	}
	return classID
}
