package interchain_accounts_test

import (
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// Test initiating a ChanOpenInit using the host chain instead of the controller chain
// ChainA is the controller chain. ChainB is the host chain
func (suite *InterchainAccountsTestSuite) TestChanOpenInitWrongFlow() {
	suite.SetupTest() // reset
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	// use chainB (host) for ChanOpenInit
	msg := channeltypes.NewMsgChannelOpenInit(path.EndpointB.ChannelConfig.PortID, types.VersionPrefix, channeltypes.ORDERED, []string{path.EndpointB.ConnectionID}, path.EndpointA.ChannelConfig.PortID, types.ModuleName)
	handler := suite.chainB.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err := handler(suite.chainB.GetContext(), msg)

	suite.Require().Error(err)
}

// Test initiating a ChanOpenTry using the controller chain instead of the host chain
// ChainA is the controller chain. ChainB creates a controller port as well,
// attempting to trick chainA.
func (suite *InterchainAccountsTestSuite) TestChanOpenTryWrongFlow() {
	suite.SetupTest() // reset
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := InitInterchainAccount(path.EndpointA, TestOwnerAddress)
	suite.Require().NoError(err)

	// chainB also creates a controller port
	err = InitInterchainAccount(path.EndpointB, TestOwnerAddress)
	suite.Require().NoError(err)

	path.EndpointA.UpdateClient()
	channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
	proofInit, proofHeight := path.EndpointB.Chain.QueryProof(channelKey)

	// use chainA (controller) for ChanOpenTry
	msg := channeltypes.NewMsgChannelOpenTry(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, TestVersion, channeltypes.ORDERED, []string{path.EndpointA.ConnectionID}, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, types.VersionPrefix, proofInit, proofHeight, types.ModuleName)
	handler := suite.chainA.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err = handler(suite.chainA.GetContext(), msg)

	suite.Require().Error(err)
}

// Test initiating a ChanOpenAck using the host chain instead of the controller chain
// ChainA is the controller chain. ChainB is the host chain
func (suite *InterchainAccountsTestSuite) TestChanOpenAckWrongFlow() {
	suite.SetupTest() // reset
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := InitInterchainAccount(path.EndpointA, TestOwnerAddress)
	suite.Require().NoError(err)

	err = path.EndpointB.ChanOpenTry()
	suite.Require().NoError(err)

	// chainA maliciously sets channel to TRYOPEN
	channel := channeltypes.NewChannel(channeltypes.TRYOPEN, channeltypes.ORDERED, channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, TestVersion)
	suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)

	// commit state changes so proof can be created
	suite.chainA.App.Commit()
	suite.chainA.NextBlock()

	path.EndpointB.UpdateClient()

	// query proof from ChainA
	channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	proofTry, proofHeight := path.EndpointA.Chain.QueryProof(channelKey)

	// use chainB (host) for ChanOpenAck
	msg := channeltypes.NewMsgChannelOpenAck(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, path.EndpointA.ChannelID, TestVersion, proofTry, proofHeight, types.ModuleName)
	handler := suite.chainB.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err = handler(suite.chainB.GetContext(), msg)

	suite.Require().Error(err)
}

// Test initiating a ChanOpenConfirm using the controller chain instead of the host chain
// ChainA is the controller chain. ChainB is the host chain
func (suite *InterchainAccountsTestSuite) TestChanOpenConfirmWrongFlow() {
	suite.SetupTest() // reset
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := InitInterchainAccount(path.EndpointA, TestOwnerAddress)
	suite.Require().NoError(err)

	err = path.EndpointB.ChanOpenTry()
	suite.Require().NoError(err)

	// chainB maliciously sets channel to OPEN
	channel := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, TestVersion)
	suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel)

	// commit state changes so proof can be created
	suite.chainB.App.Commit()
	suite.chainB.NextBlock()

	path.EndpointA.UpdateClient()

	// query proof from ChainB
	channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
	proofAck, proofHeight := path.EndpointB.Chain.QueryProof(channelKey)

	// use chainA (controller) for ChanOpenConfirm
	msg := channeltypes.NewMsgChannelOpenConfirm(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, proofAck, proofHeight, types.ModuleName)
	handler := suite.chainA.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err = handler(suite.chainA.GetContext(), msg)

	suite.Require().Error(err)
}
