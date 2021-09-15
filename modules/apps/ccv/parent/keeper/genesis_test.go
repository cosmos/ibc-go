package keeper_test

import (
	"fmt"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (suite *KeeperTestSuite) TestGenesis() {
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

	// set some chain-channel pairs before exporting
	ctx := suite.parentChain.GetContext()
	for i := 0; i < 4; i++ {
		suite.parentChain.GetSimApp().ParentKeeper.SetChainToChannel(ctx, fmt.Sprintf("chainid-%d", i), fmt.Sprintf("channel-%d", i))
		suite.parentChain.GetSimApp().ParentKeeper.SetChannelToChain(ctx, fmt.Sprintf("channel-%d", i), fmt.Sprintf("chainid-%d", i))
		suite.parentChain.GetSimApp().ParentKeeper.SetChannelStatus(ctx, fmt.Sprintf("channel-%d", i), types.Status(i))
		for i := 3; i < 6; i++ {
			suite.parentChain.GetSimApp().ParentKeeper.SetUnbondingPacketData(ctx, fmt.Sprintf("chainid-%d", i), uint64(i), pd)
		}
	}

	genState := suite.parentChain.GetSimApp().ParentKeeper.ExportGenesis(suite.parentChain.GetContext())

	suite.childChain.GetSimApp().ParentKeeper.InitGenesis(suite.childChain.GetContext(), genState)

	ctx = suite.childChain.GetContext()
	for i := 0; i < 4; i++ {
		expectedChainId := fmt.Sprintf("chainid-%d", i)
		expectedChannelId := fmt.Sprintf("channelid-%d", i)
		channelID, channelOk := suite.childChain.GetSimApp().ParentKeeper.GetChainToChannel(ctx, expectedChainId)
		chainID, chainOk := suite.childChain.GetSimApp().ParentKeeper.GetChannelToChain(ctx, expectedChannelId)
		suite.Require().True(channelOk)
		suite.Require().True(chainOk)
		suite.Require().Equal(expectedChainId, chainID, "did not store correct chain id for given channel id")
		suite.Require().Equal(expectedChannelId, channelID, "did not store correct channel id for given chain id")

		status := suite.childChain.GetSimApp().ParentKeeper.GetChannelStatus(ctx, channelID)
		suite.Require().Equal(int32(i), status, "status is unexpected for given channel id: %s", channelID)

		for j := 3; j < 6; j++ {
			suite.childChain.GetSimApp().ParentKeeper.GetUnbondingPacketData(ctx, chainID, uint64(j))
		}
	}
}
