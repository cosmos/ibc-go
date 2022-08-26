package keeper_test

import (
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
	ibcmock "github.com/cosmos/ibc-go/v5/testing/mock"
)

func (suite *KeeperTestSuite) TestMigrateChannelCapability() {
	suite.SetupTest()

	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	suite.Require().NoError(err)

	capName := host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

	// assert the capability is owned by the auth module pre migration
	cap, found := suite.chainA.GetSimApp().ScopedICAMockKeeper.GetCapability(suite.chainA.GetContext(), capName)
	suite.Require().NotNil(cap)
	suite.Require().True(found)

	cap, found = suite.chainA.GetSimApp().ScopedICAControllerKeeper.GetCapability(suite.chainA.GetContext(), capName)
	suite.Require().Nil(cap)
	suite.Require().False(found)

	err = keeper.MigrateChannelCapability(
		suite.chainA.GetContext(),
		suite.chainA.Codec,
		suite.chainA.GetSimApp().GetMemKey(capabilitytypes.MemStoreKey),
		suite.chainA.GetSimApp().CapabilityKeeper,
		ibcmock.ModuleName+types.SubModuleName,
	)

	suite.Require().NoError(err)

	// assert the capability is now owned by the controller submodule
	cap, found = suite.chainA.GetSimApp().ScopedICAControllerKeeper.GetCapability(suite.chainA.GetContext(), capName)
	suite.Require().NotNil(cap)
	suite.Require().True(found)

	cap, found = suite.chainA.GetSimApp().ScopedICAMockKeeper.GetCapability(suite.chainA.GetContext(), capName)
	suite.Require().Nil(cap)
	suite.Require().False(found)
}
