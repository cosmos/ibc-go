package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestMsgConnectionOpenInitEvents() {
	s.SetupTest()
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	msg := types.NewMsgConnectionOpenInit(
		path.EndpointA.ClientID,
		path.EndpointA.Counterparty.ClientID,
		path.EndpointA.Counterparty.Chain.GetPrefix(), ibctesting.DefaultOpenInitVersion, path.EndpointA.ConnectionConfig.DelayPeriod,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err)
	s.Require().NotNil(res)

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
	ibctesting.AssertEvents(&s.Suite, expectedEvents, events)
}

func (s *KeeperTestSuite) TestMsgConnectionOpenTryEvents() {
	s.SetupTest()
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	s.Require().NoError(path.EndpointA.ConnOpenInit())

	s.Require().NoError(path.EndpointB.UpdateClient())

	initProof, proofHeight := path.EndpointB.QueryConnectionHandshakeProof()

	msg := types.NewMsgConnectionOpenTry(
		path.EndpointB.ClientID, path.EndpointB.Counterparty.ConnectionID, path.EndpointB.Counterparty.ClientID,
		path.EndpointB.Counterparty.Chain.GetPrefix(), []*types.Version{ibctesting.ConnectionVersion},
		path.EndpointB.ConnectionConfig.DelayPeriod, initProof, proofHeight,
		path.EndpointB.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := path.EndpointB.Chain.SendMsgs(msg)
	s.Require().NoError(err)
	s.Require().NotNil(res)

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
	ibctesting.AssertEvents(&s.Suite, expectedEvents, events)
}

func (s *KeeperTestSuite) TestMsgConnectionOpenAckEvents() {
	s.SetupTest()
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	s.Require().NoError(path.EndpointA.ConnOpenInit())
	s.Require().NoError(path.EndpointB.ConnOpenTry())

	s.Require().NoError(path.EndpointA.UpdateClient())

	tryProof, proofHeight := path.EndpointA.QueryConnectionHandshakeProof()

	msg := types.NewMsgConnectionOpenAck(
		path.EndpointA.ConnectionID, path.EndpointA.Counterparty.ConnectionID,
		tryProof, proofHeight, ibctesting.ConnectionVersion,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := path.EndpointA.Chain.SendMsgs(msg)
	s.Require().NoError(err)
	s.Require().NotNil(res)

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
	ibctesting.AssertEvents(&s.Suite, expectedEvents, events)
}

func (s *KeeperTestSuite) TestMsgConnectionOpenConfirmEvents() {
	s.SetupTest()
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	s.Require().NoError(path.EndpointA.ConnOpenInit())
	s.Require().NoError(path.EndpointB.ConnOpenTry())
	s.Require().NoError(path.EndpointA.ConnOpenAck())

	s.Require().NoError(path.EndpointB.UpdateClient())

	connectionKey := host.ConnectionKey(path.EndpointB.Counterparty.ConnectionID)
	proof, height := path.EndpointB.Counterparty.Chain.QueryProof(connectionKey)

	msg := types.NewMsgConnectionOpenConfirm(
		path.EndpointB.ConnectionID,
		proof, height,
		path.EndpointB.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := path.EndpointB.Chain.SendMsgs(msg)
	s.Require().NoError(err)
	s.Require().NotNil(res)

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
	ibctesting.AssertEvents(&s.Suite, expectedEvents, events)
}
