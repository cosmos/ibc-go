package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/nft"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func TestKeeperTestSuite1(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

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
