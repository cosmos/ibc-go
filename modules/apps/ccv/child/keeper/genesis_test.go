package keeper_test

import (
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	childtypes "github.com/cosmos/ibc-go/modules/apps/ccv/child/types"
	parenttypes "github.com/cosmos/ibc-go/modules/apps/ccv/parent/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (suite *KeeperTestSuite) TestInitialGenesis() {
	genesis := suite.childChain.GetSimApp().ChildKeeper.ExportGenesis(suite.childChain.GetContext())

	suite.Require().Equal(suite.parentClient, genesis.ParentClientState)
	suite.Require().Equal(suite.parentConsState, genesis.ParentConsensusState)

	suite.Require().NotPanics(func() {
		suite.childChain.GetSimApp().ChildKeeper.InitGenesis(suite.childChain.GetContext(), genesis)
		// reset suite to reset parent client
		suite.SetupTest()
	})

	ctx := suite.childChain.GetContext()
	portId := suite.childChain.GetSimApp().ChildKeeper.GetPort(ctx)
	suite.Require().Equal(childtypes.PortID, portId)

	clientId, ok := suite.childChain.GetSimApp().ChildKeeper.GetParentClient(ctx)
	suite.Require().True(ok)
	clientState, ok := suite.childChain.App.GetIBCKeeper().ClientKeeper.GetClientState(ctx, clientId)
	suite.Require().True(ok)
	suite.Require().Equal(genesis.ParentClientState, clientState, "client state not set correctly after InitGenesis")

	suite.SetupCCVChannel()

	origTime := suite.childChain.GetContext().BlockTime()

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
	packet := channeltypes.NewPacket(pd.GetBytes(), 1, parenttypes.PortID, suite.path.EndpointB.ChannelID, childtypes.PortID, suite.path.EndpointA.ChannelID,
		clienttypes.NewHeight(1, 0), 0)
	suite.childChain.GetSimApp().ChildKeeper.OnRecvPacket(suite.childChain.GetContext(), packet, pd)

	restartGenesis := suite.childChain.GetSimApp().ChildKeeper.ExportGenesis(suite.childChain.GetContext())

	// ensure reset genesis is set correctly
	parentChannel := suite.path.EndpointA.ChannelID
	suite.Require().Equal(parentChannel, restartGenesis.ParentChannelId)
	unbondingTime := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.childChain.GetContext(), 1)
	suite.Require().Equal(uint64(origTime.Add(childtypes.UnbondingTime).UnixNano()), unbondingTime, "unbonding time is not set correctly in genesis")
	unbondingPacket, err := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingPacket(suite.childChain.GetContext(), 1)
	suite.Require().NoError(err)
	suite.Require().Equal(&packet, unbondingPacket, "unbonding packet is not set correctly in genesis")

	suite.Require().NotPanics(func() {
		suite.childChain.GetSimApp().ChildKeeper.InitGenesis(suite.childChain.GetContext(), restartGenesis)
	})
}
