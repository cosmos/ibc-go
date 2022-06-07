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

// The following test describes the entire cross-chain process of nft-transfer.
// The execution sequence of the cross-chain process is:
// A -> B -> C -> A -> C -> B ->A
func (suite *KeeperTestSuite) TestSendAndReceive() {
	pathA2B := NewTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(pathA2B)
	suite.coordinator.CreateChannels(pathA2B)

	classID := "cryptoCat"
	classURI := "cat_uri"
	nftID := "kitty"
	nftURI := "kittt_uri"

	var targetClassID string
	var packet channeltypes.Packet

	//============================== setup start===============================
	nftKeeper := pathA2B.EndpointA.Chain.GetSimApp().NFTKeeper
	err := nftKeeper.SaveClass(pathA2B.EndpointA.Chain.GetContext(), nft.Class{
		Id:  classID,
		Uri: classURI,
	})
	suite.Require().NoError(err, "SaveClass error")

	err = nftKeeper.Mint(pathA2B.EndpointA.Chain.GetContext(), nft.NFT{
		ClassId: classID,
		Id:      nftID,
		Uri:     nftURI,
	}, pathA2B.EndpointA.Chain.SenderAccount.GetAddress())
	//============================== setup end===============================

	suite.Run("transfer forward A->B", func() {
		{
			packet = suite.transferNFT(
				pathA2B.EndpointA,
				pathA2B.EndpointB,
				classID,
				nftID,
				pathA2B.EndpointA.Chain.SenderAccount.GetAddress().String(),
				pathA2B.EndpointB.Chain.SenderAccount.GetAddress().String(),
			)

			targetClassID = suite.receiverNFT(
				pathA2B.EndpointA,
				pathA2B.EndpointB,
				packet,
			)
		}
	})

	// transfer from chainB to chainC
	pathB2C := NewTransferPath(suite.chainB, suite.chainC)
	suite.Run("transfer forward B->C", func() {
		{
			suite.coordinator.SetupConnections(pathB2C)
			suite.coordinator.CreateChannels(pathB2C)

			packet = suite.transferNFT(
				pathB2C.EndpointA,
				pathB2C.EndpointB,
				targetClassID,
				nftID,
				pathA2B.EndpointB.Chain.SenderAccount.GetAddress().String(),
				pathB2C.EndpointB.Chain.SenderAccount.GetAddress().String(),
			)

			targetClassID = suite.receiverNFT(
				pathB2C.EndpointA,
				pathB2C.EndpointB,
				packet,
			)
		}
	})

	// transfer from chainC to chainA
	pathC2A := NewTransferPath(suite.chainC, suite.chainA)
	suite.Run("transfer forward C->A", func() {
		{
			suite.coordinator.SetupConnections(pathC2A)
			suite.coordinator.CreateChannels(pathC2A)

			packet = suite.transferNFT(
				pathC2A.EndpointA,
				pathC2A.EndpointB,
				targetClassID,
				nftID,
				pathB2C.EndpointB.Chain.SenderAccount.GetAddress().String(),
				pathC2A.EndpointB.Chain.SenderAccount.GetAddress().String(),
			)

			targetClassID = suite.receiverNFT(
				pathC2A.EndpointA,
				pathC2A.EndpointB,
				packet,
			)
		}
	})

	suite.Run("transfer back A->C", func() {
		{
			packet = suite.transferNFT(
				pathC2A.EndpointB,
				pathC2A.EndpointA,
				targetClassID,
				nftID,
				pathC2A.EndpointB.Chain.SenderAccount.GetAddress().String(),
				pathB2C.EndpointB.Chain.SenderAccount.GetAddress().String(),
			)

			targetClassID = suite.receiverNFT(
				pathC2A.EndpointB,
				pathC2A.EndpointA,
				packet,
			)
		}
	})

	suite.Run("transfer back C->B", func() {
		{
			packet = suite.transferNFT(
				pathB2C.EndpointB,
				pathB2C.EndpointA,
				targetClassID,
				nftID,
				pathB2C.EndpointB.Chain.SenderAccount.GetAddress().String(),
				pathB2C.EndpointA.Chain.SenderAccount.GetAddress().String(),
			)

			targetClassID = suite.receiverNFT(
				pathB2C.EndpointB,
				pathB2C.EndpointA,
				packet,
			)
		}
	})

	suite.Run("transfer back B->A", func() {
		{
			packet = suite.transferNFT(
				pathA2B.EndpointB,
				pathA2B.EndpointA,
				targetClassID,
				nftID,
				pathB2C.EndpointA.Chain.SenderAccount.GetAddress().String(),
				pathA2B.EndpointA.Chain.SenderAccount.GetAddress().String(),
			)

			targetClassID = suite.receiverNFT(
				pathA2B.EndpointB,
				pathA2B.EndpointA,
				packet,
			)
		}
	})
	suite.Equal(classID, targetClassID, "wrong classID")
}

func (suite *KeeperTestSuite) transferNFT(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	classID, nftID string,
	sender, receiver string,
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

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	var data types.NonFungibleTokenPacketData
	err = types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data)
	suite.Require().NoError(err)

	isAwayFromOrigin := types.IsAwayFromOrigin(packet.SourcePort, packet.SourceChannel, data.ClassId)

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
	return packet

}

func (suite *KeeperTestSuite) receiverNFT(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	packet channeltypes.Packet,
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

	isAwayFromOrigin := types.IsAwayFromOrigin(packet.SourcePort, packet.SourceChannel, data.ClassId)
	if isAwayFromOrigin {
		//construct classTrace
		prefixedClassID := types.GetClassPrefix(toEndpoint.ChannelConfig.PortID, toEndpoint.ChannelID) + data.GetClassId()
		trace := types.ParseClassTrace(prefixedClassID)
		classID = trace.IBCClassID()
	} else {
		unprefixedClassID := types.RemoveClassPrefix(packet.GetSourcePort(),
			packet.GetSourceChannel(), data.ClassId)
		classID = types.ParseClassTrace(unprefixedClassID).IBCClassID()
	}

	// check class
	class, found := toEndpoint.Chain.GetSimApp().
		NFTKeeper.GetClass(toEndpoint.Chain.GetContext(), classID)
	suite.Require().True(found, "not found class")
	suite.Require().Equal(nft.Class{Id: classID, Uri: data.GetClassUri()}, class, "class not equal")

	// check nft owner
	suite.Require().Equal(
		data.GetReceiver(),
		toEndpoint.Chain.GetSimApp().
			NFTKeeper.GetOwner(toEndpoint.Chain.GetContext(), classID, data.GetTokenIds()[0]).String(),
		"nft not equal",
	)

	// check nft
	token, found := toEndpoint.Chain.GetSimApp().
		NFTKeeper.GetNFT(toEndpoint.Chain.GetContext(), classID, data.GetTokenIds()[0])
	suite.Require().True(found, "not found class")
	suite.Require().Equal(
		nft.NFT{ClassId: classID, Id: data.GetTokenIds()[0], Uri: data.GetTokenUris()[0]},
		token,
		"nft not equal",
	)
	return classID
}
