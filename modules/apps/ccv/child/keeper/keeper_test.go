package keeper_test

import (
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/stretchr/testify/suite"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains
	parentChain *ibctesting.TestChain
	childChain  *ibctesting.TestChain

	parentClient    *ibctmtypes.ClientState
	parentConsState *ibctmtypes.ConsensusState

	ctx sdk.Context
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.parentChain = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.childChain = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	tmConfig := ibctesting.NewTendermintConfig()

	// commit a block on parent chain before creating client
	suite.coordinator.CommitBlock(suite.parentChain)

	// create client and consensus state of parent chain to initialize child chain genesis.
	height := suite.parentChain.LastHeader.GetHeight().(clienttypes.Height)
	UpgradePath := []string{"upgrade", "upgradedIBCState"}

	suite.parentClient = ibctmtypes.NewClientState(
		suite.parentChain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
		height, commitmenttypes.GetSDKSpecs(), UpgradePath, tmConfig.AllowUpdateAfterExpiry, tmConfig.AllowUpdateAfterMisbehaviour,
	)
	suite.parentConsState = suite.parentChain.LastHeader.ConsensusState()

	childGenesis := types.NewInitialChildGenesisState(suite.parentClient, suite.parentConsState)
	suite.childChain.GetSimApp().ChildKeeper.InitGenesis(suite.childChain.GetContext(), childGenesis)

	suite.ctx = suite.childChain.GetContext()
}

func (suite *KeeperTestSuite) TestParentClient() {
	parentClient, ok := suite.childChain.GetSimApp().ChildKeeper.GetParentClient(suite.ctx)
	suite.Require().True(ok)

	clientState, ok := suite.childChain.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, parentClient)
	suite.Require().Equal(suite.parentClient, clientState, "stored client state does not match genesis parent client")
}

func (suite *KeeperTestSuite) TestParentChannel() {
	_, ok := suite.childChain.GetSimApp().ChildKeeper.GetParentChannel(suite.ctx)
	suite.Require().False(ok)
	suite.childChain.GetSimApp().ChildKeeper.SetParentChannel(suite.ctx, "channelID")
	channelID, ok := suite.childChain.GetSimApp().ChildKeeper.GetParentChannel(suite.ctx)
	suite.Require().True(ok)
	suite.Require().Equal("channelID", channelID)
}

func (suite *KeeperTestSuite) TestPendingChanges() {
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

	suite.childChain.GetSimApp().ChildKeeper.SetPendingChanges(suite.ctx, pd)
	gotPd, ok := suite.childChain.GetSimApp().ChildKeeper.GetPendingChanges(suite.ctx)
	suite.Require().True(ok)
	suite.Require().Equal(pd, gotPd, "packet data in store does not equal packet data set")
	suite.childChain.GetSimApp().ChildKeeper.DeletePendingChanges(suite.ctx)
	gotPd, ok = suite.childChain.GetSimApp().ChildKeeper.GetPendingChanges(suite.ctx)
	suite.Require().False(ok)
	suite.Require().Nil(gotPd, "got non-nil pending changes after Delete")
}

func (suite *KeeperTestSuite) TestUnbondingTime() {
	suite.childChain.GetSimApp().ChildKeeper.SetUnbondingTime(suite.ctx, 1, 10)
	suite.childChain.GetSimApp().ChildKeeper.SetUnbondingTime(suite.ctx, 2, 25)
	suite.childChain.GetSimApp().ChildKeeper.SetUnbondingTime(suite.ctx, 5, 15)
	suite.childChain.GetSimApp().ChildKeeper.SetUnbondingTime(suite.ctx, 6, 40)

	suite.childChain.GetSimApp().ChildKeeper.DeleteUnbondingTime(suite.ctx, 6)

	suite.Require().Equal(10, suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, 1))
	suite.Require().Equal(25, suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, 2))
	suite.Require().Equal(15, suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, 5))
	suite.Require().Equal(0, suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, 3))
	suite.Require().Equal(0, suite.childChain.GetSimApp().ChildKeeper.GetUnbondingTime(suite.ctx, 6))

	orderedTimes := [][]uint64{{1, 10}, {2, 25}, {5, 15}}
	i := 0

	suite.childChain.GetSimApp().ChildKeeper.IterateUnbondingTime(suite.ctx, func(seq, time uint64) bool {
		// require that we iterate through unbonding time in order of sequence
		suite.Require().Equal(orderedTimes[i][0], seq)
		suite.Require().Equal(orderedTimes[i][1], time)
		i++
		return false
	})
}

func (suite *KeeperTestSuite) TestUnbondingPacket() {
	pk1, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	suite.Require().NoError(err)
	pk2, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	suite.Require().NoError(err)

	for i := 0; i < 5; i++ {
		pd := types.NewValidatorSetChangePacketData(
			[]abci.ValidatorUpdate{
				{
					PubKey: pk1,
					Power:  int64(i),
				},
				{
					PubKey: pk2,
					Power:  int64(i + 5),
				},
			},
		)
		packet := channeltypes.NewPacket(pd.GetBytes(), uint64(i), "parent", "channel-1", "child", "channel-1",
			clienttypes.NewHeight(1, 0), 0)
		suite.childChain.GetSimApp().ChildKeeper.SetUnbondingPacket(suite.ctx, uint64(i), packet)
	}

	// ensure that packets are iterated by sequence
	i := 0
	suite.childChain.GetSimApp().ChildKeeper.IterateUnbondingPacket(suite.ctx, func(seq uint64, packet channeltypes.Packet) bool {
		suite.Require().Equal(i, seq)
		gotPacket, err := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingPacket(suite.ctx, seq)
		suite.Require().NoError(err)
		suite.Require().Equal(packet, gotPacket, "packet from get and iteration do not match")
		i++
		return false
	})

	suite.childChain.GetSimApp().ChildKeeper.DeleteUnbondingPacket(suite.ctx, 0)
	gotPacket, err := suite.childChain.GetSimApp().ChildKeeper.GetUnbondingPacket(suite.ctx, 0)
	suite.Require().Error(err)
	suite.Require().Nil(gotPacket, "packet is not nil after delete")
}

func (suite *KeeperTestSuite) SetupCCVChannel() {

}
