package keeper_test

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/nft"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *KeeperTestSuite) TestSendTransfer() {
	var (
		path    *ibctesting.Path
		err     error
		classID string
	)

	baseClassID := "cryptoCat"
	classURI := "cat_uri"
	nftID := "kitty"
	nftURI := "kittt_uri"

	testCases := []struct {
		msg              string
		malleate         func()
		isAwayFromOrigin bool
	}{
		{
			"successful transfer from chainA to chainB",
			func() {
				suite.coordinator.CreateChannels(path)
				classID = baseClassID

				nftKeeper := path.EndpointA.Chain.GetSimApp().NFTKeeper
				err = nftKeeper.SaveClass(path.EndpointA.Chain.GetContext(), nft.Class{
					Id:  classID,
					Uri: classURI,
				})
				suite.Require().NoError(err, "SaveClass error")

				err = nftKeeper.Mint(path.EndpointA.Chain.GetContext(), nft.NFT{
					ClassId: classID,
					Id:      nftID,
					Uri:     nftURI,
				}, path.EndpointA.Chain.SenderAccount.GetAddress())
				suite.Require().NoError(err, "Mint error")
			},
			true,
		},
		{
			"successful transfer from chainB to chainA",
			func() {
				suite.coordinator.CreateChannels(path)
				trace := types.ParseClassTrace(
					types.GetClassPrefix(
						path.EndpointA.ChannelConfig.PortID,
						path.EndpointA.ChannelID,
					) + baseClassID)
				path.EndpointB.Chain.GetSimApp().NFTTransferKeeper.SetClassTrace(path.EndpointB.Chain.GetContext(), trace)

				classID = trace.IBCClassID()
				nftKeeper := path.EndpointB.Chain.GetSimApp().NFTKeeper
				err = nftKeeper.SaveClass(path.EndpointB.Chain.GetContext(), nft.Class{
					Id:  classID,
					Uri: classURI,
				})
				suite.Require().NoError(err, "SaveClass error")

				err = nftKeeper.Mint(path.EndpointB.Chain.GetContext(), nft.NFT{
					ClassId: classID,
					Id:      nftID,
					Uri:     nftURI,
				}, path.EndpointB.Chain.SenderAccount.GetAddress())
				suite.Require().NoError(err, "Mint error")
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			path = NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			tc.malleate()

			if !tc.isAwayFromOrigin {
				ctx := path.EndpointB.Chain.GetContext()
				err = path.EndpointB.Chain.GetSimApp().NFTTransferKeeper.SendTransfer(
					ctx,
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					classID,
					[]string{nftID},
					path.EndpointB.Chain.SenderAccount.GetAddress(),
					path.EndpointA.Chain.SenderAccount.GetAddress().String(), clienttypes.NewHeight(0, 110), 0,
				)
				suite.Require().NoError(err)

				suite.Require().False(
					path.EndpointB.Chain.GetSimApp().NFTKeeper.HasNFT(ctx, classID, nftID),
					"burn nft failed",
				)
				return
			}

			ctx := path.EndpointA.Chain.GetContext()
			err = path.EndpointA.Chain.GetSimApp().NFTTransferKeeper.SendTransfer(
				ctx,
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				classID,
				[]string{nftID},
				path.EndpointA.Chain.SenderAccount.GetAddress(),
				path.EndpointB.Chain.SenderAccount.GetAddress().String(), clienttypes.NewHeight(0, 110), 0,
			)

			suite.Require().NoError(err)
			suite.Require().Equal(
				types.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID),
				path.EndpointA.Chain.GetSimApp().NFTKeeper.GetOwner(ctx, classID, nftID),
				"escrow nft failed",
			)
		})
	}
}

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	var (
		path              *ibctesting.Path
		trace             types.ClassTrace
		classID, receiver string
		nftIDs, nftURIs   []string
	)

	baseClassID := "cryptoCat"
	classURI := "cat_uri"
	nftID := "kitty"
	nftURI := "kittt_uri"

	testCases := []struct {
		msg              string
		malleate         func()
		isAwayFromOrigin bool // the receiving chain is the source of the coin originally
		expPass          bool
	}{
		{"success receive chain is away from origin chain", func() {}, true, true},
		{"success receive chain is not away from origin chain", func() {
			classID = types.GetClassPrefix(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
			) + baseClassID

			suite.chainB.GetSimApp().NFTKeeper.SaveClass(suite.chainB.GetContext(), nft.Class{
				Id:  baseClassID,
				Uri: classURI,
			})

			escrowAddress := types.GetEscrowAddress(
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
			)
			suite.chainB.GetSimApp().NFTKeeper.Mint(suite.chainB.GetContext(), nft.NFT{
				ClassId: baseClassID,
				Id:      nftID,
				Uri:     nftURI,
			}, escrowAddress)

		}, false, true},
		{"empty classID", func() {
			classID = ""
		}, true, false},
		{"empty nftIDs", func() {
			nftURIs = nil
		}, true, false},
		{"empty nftURIs", func() {
			nftURIs = nil
		}, true, false},
		{"invalid receiver address", func() {
			receiver = "gaia1scqhwpgsmr6vmztaa7suurfl52my6nd2kmrudl"
		}, true, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			classID = baseClassID
			receiver = suite.chainB.SenderAccount.GetAddress().String()
			nftIDs = []string{nftID}
			nftURIs = []string{nftURI}

			tc.malleate()

			trace = types.ParseClassTrace(classID)
			data := types.NewNonFungibleTokenPacketData(
				trace.GetFullClassPath(),
				classURI,
				nftIDs,
				nftURIs,
				suite.chainA.SenderAccount.GetAddress().String(),
				receiver,
			)

			packet := channeltypes.NewPacket(
				data.GetBytes(),
				1, //not check sequence
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			err := suite.chainB.GetSimApp().
				NFTTransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, data)

			if !tc.expPass {
				suite.Require().Error(err)
				return
			}

			if tc.isAwayFromOrigin {
				prefixedClassID := types.GetClassPrefix(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
				) + baseClassID
				trace = types.ParseClassTrace(prefixedClassID)

				suite.Require().Equal(
					receiver,
					suite.chainB.GetSimApp().NFTKeeper.GetOwner(suite.chainB.GetContext(), trace.IBCClassID(), nftID).String(),
					"receive packet failed",
				)

				suite.Require().True(
					suite.chainB.GetSimApp().NFTTransferKeeper.HasClassTrace(suite.chainB.GetContext(), trace.Hash()),
					"not found class trace",
				)

			} else {
				suite.Require().False(
					suite.chainB.GetSimApp().NFTKeeper.HasNFT(suite.chainB.GetContext(), classID, nftID),
					"burn nft failed")
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnAcknowledgementPacket() {
	var (
		successAck      = channeltypes.NewResultAcknowledgement([]byte{byte(1)})
		failedAck       = channeltypes.NewErrorAcknowledgement("failed packet transfer")
		path            *ibctesting.Path
		trace           types.ClassTrace
		classID         string
		nftIDs, nftURIs []string
	)

	baseClassID := "cryptoCat"
	classURI := "cat_uri"
	nftID := "kitty"
	nftURI := "kittt_uri"

	testCases := []struct {
		msg      string
		ack      channeltypes.Acknowledgement
		malleate func()
		success  bool // success of ack
		expPass  bool
	}{
		{"success ack causes no-op", successAck, func() {}, true, true},
		{"successful refund when isAwayFromOrigin is false", failedAck, func() {
			// if isAwayFromOrigin is false, OnAcknowledgementPacket will mint nft to sender again

			// mock SendTransfer
			classID = types.GetClassPrefix(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
			) + baseClassID

			ibcClassID := types.ParseClassTrace(classID).IBCClassID()
			suite.chainA.GetSimApp().NFTKeeper.SaveClass(suite.chainA.GetContext(), nft.Class{
				Id:  ibcClassID,
				Uri: classURI,
			})

		}, false, true},
		{"successful refund when isAwayFromOrigin is true", failedAck, func() {
			// if isAwayFromOrigin is true, OnAcknowledgementPacket will unescrow nft to sender

			// mock SendTransfer
			classID = types.GetClassPrefix(
				path.EndpointB.ChannelConfig.PortID,
				"channel-1",
			) + baseClassID

			ibcClassID := types.ParseClassTrace(classID).IBCClassID()
			suite.chainA.GetSimApp().NFTKeeper.SaveClass(suite.chainA.GetContext(), nft.Class{
				Id:  ibcClassID,
				Uri: classURI,
			})

			escrowAddress := types.GetEscrowAddress(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
			)

			suite.chainA.GetSimApp().NFTKeeper.Mint(suite.chainA.GetContext(), nft.NFT{
				ClassId: ibcClassID,
				Id:      nftID,
				Uri:     nftURI,
			}, escrowAddress)

		}, false, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			path = NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			classID = baseClassID
			nftIDs = []string{nftID}
			nftURIs = []string{nftURI}

			tc.malleate()

			trace = types.ParseClassTrace(classID)
			data := types.NewNonFungibleTokenPacketData(
				trace.GetFullClassPath(),
				classURI,
				nftIDs,
				nftURIs,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
			)

			packet := channeltypes.NewPacket(
				data.GetBytes(),
				1, //not check sequence
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			err := suite.chainA.GetSimApp().NFTTransferKeeper.OnAcknowledgementPacket(
				suite.chainA.GetContext(),
				packet,
				data, tc.ack,
			)

			if !tc.expPass {
				suite.Require().Error(err)
			}

			suite.Require().NoError(err, "OnAcknowledgementPacket failed")
			if tc.success {
				// if successful, nft is hosted in account a or destroyed(executed when SendTransfer)
				return
			}

			suite.Require().Equal(
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainA.GetSimApp().NFTKeeper.GetOwner(suite.chainA.GetContext(), trace.IBCClassID(), nftID).String(),
				"refund failed",
			)
		})
	}
}
