package ibccallbacks_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// CallbacksTestSuite defines the needed instances and methods to test callbacks
type CallbacksTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	path *ibctesting.Path
}

// setupChains sets up a coordinator with 2 test chains.
func (suite *CallbacksTestSuite) setupChains() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

// SetupTransferTest sets up a transfer channel between chainA and chainB
func (suite *CallbacksTestSuite) SetupTransferTest() {
	suite.setupChains()

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	suite.path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	suite.path.EndpointA.ChannelConfig.Version = transfertypes.Version
	suite.path.EndpointB.ChannelConfig.Version = transfertypes.Version

	suite.coordinator.Setup(suite.path)
}

// SetupICATest sets up an interchain accounts channel between chainA and chainB
func (suite *CallbacksTestSuite) SetupICATest() {
	suite.setupChains()

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)

	// ICAVersion defines a reusable interchainaccounts version string for testing purposes
	ICAVersion := icatypes.NewDefaultMetadataString(suite.path.EndpointA.ConnectionID, suite.path.EndpointB.ConnectionID)

	suite.path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	suite.path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	suite.path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	suite.path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	suite.path.EndpointA.ChannelConfig.Version = ICAVersion
	suite.path.EndpointB.ChannelConfig.Version = ICAVersion
}

func TestIBCCallbacksTestSuite(t *testing.T) {
	suite.Run(t, new(CallbacksTestSuite))
}
