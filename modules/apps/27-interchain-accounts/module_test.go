package interchain_accounts_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"
	// TestPortID defines a resuable port identifier for testing purposes
	TestPortID = fmt.Sprintf("ics-27-0-0-%s", TestOwnerAddress)
	// TestVersion defines a resuable interchainaccounts version string for testing purposes
	TestVersion = types.NewAppVersion(types.VersionPrefix, types.GenerateAddress(TestPortID).String())
)

type InterchainAccountsTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *InterchainAccountsTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func NewICAPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = types.PortID
	path.EndpointB.ChannelConfig.PortID = types.PortID
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointA.ChannelConfig.Version = types.VersionPrefix
	path.EndpointB.ChannelConfig.Version = TestVersion

	return path
}

func TestICATestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsTestSuite))
}

func (suite *InterchainAccountsTestSuite) TestNegotiateAppVersion() {
	suite.SetupTest()
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	module, _, err := suite.chainA.GetSimApp().GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), types.PortID)
	suite.Require().NoError(err)

	cbs, ok := suite.chainA.GetSimApp().GetIBCKeeper().Router.GetRoute(module)
	suite.Require().True(ok)

	counterpartyPortID, err := types.GeneratePortID(TestOwnerAddress, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)
	suite.Require().NoError(err)

	counterparty := &channeltypes.Counterparty{
		PortId:    counterpartyPortID,
		ChannelId: path.EndpointB.ChannelID,
	}

	version, err := cbs.NegotiateAppVersion(suite.chainA.GetContext(), channeltypes.ORDERED, path.EndpointA.ConnectionID, types.PortID, *counterparty, types.VersionPrefix)
	suite.Require().NoError(err)
	suite.Require().NoError(types.ValidateVersion(version))
	suite.Require().Equal(TestVersion, version)
}
