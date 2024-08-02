package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestMsgConnectionOpenInitEvents() {
	suite.SetupTest()
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupClients()

	msg := types.NewMsgConnectionOpenInit(
		path.EndpointA.ClientID,
		path.EndpointA.Counterparty.ClientID,
		path.EndpointA.Counterparty.Chain.GetPrefix(), ibctesting.DefaultOpenInitVersion, path.EndpointA.ConnectionConfig.DelayPeriod,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			types.EventTypeConnectionOpenInit,
			sdk.NewAttribute(types.AttributeKeyConnectionID, ibctesting.FirstConnectionID),
			sdk.NewAttribute(types.AttributeKeyClientID, path.EndpointA.ClientID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyClientID, path.EndpointB.ClientID),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&suite.Suite, expectedEvents, events)
}

func (suite *KeeperTestSuite) TestMsgConnectionOpenTryEvents() {
	suite.SetupTest()
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupClients()

	suite.Require().NoError(path.EndpointA.ConnOpenInit())

	suite.Require().NoError(path.EndpointB.UpdateClient())

	counterpartyClient, clientProof, consensusProof, consensusHeight, initProof, proofHeight := path.EndpointB.QueryConnectionHandshakeProof()

	msg := types.NewMsgConnectionOpenTry(
		path.EndpointB.ClientID, path.EndpointB.Counterparty.ConnectionID, path.EndpointB.Counterparty.ClientID,
		counterpartyClient, path.EndpointB.Counterparty.Chain.GetPrefix(), []*types.Version{ibctesting.ConnectionVersion}, path.EndpointB.ConnectionConfig.DelayPeriod,
		initProof, clientProof, consensusProof,
		proofHeight, consensusHeight,
		path.EndpointB.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := path.EndpointB.Chain.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			types.EventTypeConnectionOpenTry,
			sdk.NewAttribute(types.AttributeKeyConnectionID, ibctesting.FirstConnectionID),
			sdk.NewAttribute(types.AttributeKeyClientID, path.EndpointB.ClientID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyClientID, path.EndpointA.ClientID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyConnectionID, path.EndpointA.ConnectionID),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&suite.Suite, expectedEvents, events)
}

func (suite *KeeperTestSuite) TestMsgConnectionOpenAckEvents() {
	suite.SetupTest()
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupClients()

	suite.Require().NoError(path.EndpointA.ConnOpenInit())
	suite.Require().NoError(path.EndpointB.ConnOpenTry())

	suite.Require().NoError(path.EndpointA.UpdateClient())

	counterpartyClient, clientProof, consensusProof, consensusHeight, tryProof, proofHeight := path.EndpointA.QueryConnectionHandshakeProof()

	msg := types.NewMsgConnectionOpenAck(
		path.EndpointA.ConnectionID, path.EndpointA.Counterparty.ConnectionID, counterpartyClient,
		tryProof, clientProof, consensusProof,
		proofHeight, consensusHeight,
		ibctesting.ConnectionVersion,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := path.EndpointA.Chain.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			types.EventTypeConnectionOpenAck,
			sdk.NewAttribute(types.AttributeKeyConnectionID, ibctesting.FirstConnectionID),
			sdk.NewAttribute(types.AttributeKeyClientID, path.EndpointA.ClientID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyClientID, path.EndpointB.ClientID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyConnectionID, path.EndpointB.ConnectionID),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&suite.Suite, expectedEvents, events)
}

func (suite *KeeperTestSuite) TestMsgConnectionOpenConfirmEvents() {
	suite.SetupTest()
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupClients()

	suite.Require().NoError(path.EndpointA.ConnOpenInit())
	suite.Require().NoError(path.EndpointB.ConnOpenTry())
	suite.Require().NoError(path.EndpointA.ConnOpenAck())

	suite.Require().NoError(path.EndpointB.UpdateClient())

	connectionKey := host.ConnectionKey(path.EndpointB.Counterparty.ConnectionID)
	proof, height := path.EndpointB.Counterparty.Chain.QueryProof(connectionKey)

	msg := types.NewMsgConnectionOpenConfirm(
		path.EndpointB.ConnectionID,
		proof, height,
		path.EndpointB.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := path.EndpointB.Chain.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			types.EventTypeConnectionOpenConfirm,
			sdk.NewAttribute(types.AttributeKeyConnectionID, ibctesting.FirstConnectionID),
			sdk.NewAttribute(types.AttributeKeyClientID, path.EndpointB.ClientID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyClientID, path.EndpointA.ClientID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyConnectionID, path.EndpointA.ConnectionID),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&suite.Suite, expectedEvents, events)
}
