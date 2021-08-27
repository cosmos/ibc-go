package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *KeeperTestSuite) SetupTest() {
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
	path.EndpointA.ChannelConfig.Version = types.Version
	path.EndpointB.ChannelConfig.Version = types.Version

	return path
}

// InitInterchainAccount is a helper function for starting the channel handshake
// TODO: parse identifiers from events
func InitInterchainAccount(endpoint *ibctesting.Endpoint, owner string) error {
	portID, err := types.GeneratePortID(owner, endpoint.ConnectionID, endpoint.Counterparty.ConnectionID)
	if err != nil {
		return err
	}
	channelSequence := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(endpoint.Chain.GetContext())

	if err := endpoint.Chain.GetSimApp().ICAKeeper.InitInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, endpoint.Counterparty.ConnectionID, owner); err != nil {
		return err
	}

	// commit state changes for proof verification
	endpoint.Chain.App.Commit()
	endpoint.Chain.NextBlock()

	// update port/channel ids
	endpoint.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
	endpoint.ChannelConfig.PortID = portID
	return nil
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestIsBound() {
	isBound := suite.chainA.GetSimApp().ICAKeeper.IsBound(suite.chainA.GetContext(), types.PortID)
	suite.Require().True(isBound)
}

func (suite *KeeperTestSuite) TestGetPort() {
	port := suite.chainA.GetSimApp().ICAKeeper.GetPort(suite.chainA.GetContext())
	suite.Require().Equal(types.PortID, port)
}

func (suite *KeeperTestSuite) TestGetInterchainAccountAddress() {
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(path)

	counterpartyPortID := suite.chainA.GetSimApp().ICAKeeper.GetPort(suite.chainA.GetContext())
	expectedAddr := authtypes.NewBaseAccountWithAddress(types.GenerateAddress(counterpartyPortID)).GetAddress()

	retrievdAddress, err := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), counterpartyPortID)
	suite.Require().NoError(err, "GetInterchainAccountAddress failed")
	suite.Require().Equal(expectedAddr.String(), retrievdAddress)

	retrievdAddress, err = suite.chainA.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), "invalid port")
	suite.Require().Error(err)
	suite.Require().Empty(retrievdAddress)
}

func (suite *KeeperTestSuite) TestSetInterchainAccountAddress() {
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(path)

	counterpartyPortID := suite.chainA.GetSimApp().ICAKeeper.GetPort(suite.chainA.GetContext())
	expectedAddr := authtypes.NewBaseAccountWithAddress(types.GenerateAddress(counterpartyPortID)).GetAddress()

	retrievdAddress, err := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), counterpartyPortID)
	suite.Require().NoError(err, "SetInterchainAccountAddress failed")
	suite.Require().Equal(expectedAddr.String(), retrievdAddress)
}
