package child_test

import (
	"github.com/cosmos/ibc-go/modules/apps/ccv/child/keeper"
	ccvtypes "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/stretchr/testify/suite"
)

type ChildTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains
	parentChain *ibctesting.TestChain
	childChain  *ibctesting.TestChain

	keeper keeper.Keeper
}

func (suite *ChildTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.parentChain = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.childChain = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func (suite *ChildTestSuite) SetupNewChain() {
	// get parent client and consensus state
	tmConfig := ibctesting.NewTendermintConfig()
	height := suite.parentChain.LastHeader.GetHeight().(clienttypes.Height)
	clientState := ibctmtypes.NewClientState(suite.parentChain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
		height, commitmenttypes.GetSDKSpecs(), nil, tmConfig.AllowUpdateAfterExpiry, tmConfig.AllowUpdateAfterMisbehaviour)
	consensusState := suite.parentChain.LastHeader.ConsensusState()

	// construct child genesis state with hardcoded parent client and consensus state and initialize child genesis.
	genState := ccvtypes.NewInitialChildGenesisState(clientState, consensusState)
	suite.childChain.GetSimApp().ChildKeeper.InitGenesis(suite.childChain.GetContext(), genState)
}

func (suite *ChildTestSuite) TestOnChanOpenInit() {

}
