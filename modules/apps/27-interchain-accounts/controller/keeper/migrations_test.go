package keeper_test

import (
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/keeper"
)

func (suite *KeeperTestSuite) TestMigrateChannelCapability() {
	suite.SetupTest()

	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	suite.Require().NoError(err)

	err = keeper.MigrateChannelCapability(
		suite.chainA.GetContext(),
		suite.chainA.Codec,
		suite.chainA.GetSimApp().GetMemKey(capabilitytypes.MemStoreKey),
		*suite.chainA.GetSimApp().CapabilityKeeper,
		"mockicacontroller",
	)

	suite.Require().NoError(err)

	// TODO: follow up assertions
}
