package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestMsgCreateClientEvents() {
	s.SetupTest()
	path := ibctesting.NewPath(s.chainA, s.chainB)

	path.EndpointA.Counterparty.Chain.NextBlock()

	tmConfig, ok := path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig)
	s.Require().True(ok)

	height, ok := path.EndpointA.Counterparty.Chain.LatestCommittedHeader.GetHeight().(clienttypes.Height)
	s.Require().True(ok)

	clientState := ibctm.NewClientState(
		path.EndpointA.Counterparty.Chain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
		height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
	consensusState := path.EndpointA.Counterparty.Chain.LatestCommittedHeader.ConsensusState()

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	)
	s.Require().NoError(err)

	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			clienttypes.EventTypeCreateClient,
			sdk.NewAttribute(clienttypes.AttributeKeyClientID, ibctesting.FirstClientID),
			sdk.NewAttribute(clienttypes.AttributeKeyClientType, clientState.ClientType()),
			sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeight, clientState.LatestHeight.String()),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&s.Suite, expectedEvents, events)
}

func (s *KeeperTestSuite) TestMsgUpdateClientEvents() {
	s.SetupTest()
	path := ibctesting.NewPath(s.chainA, s.chainB)

	s.Require().NoError(path.EndpointA.CreateClient())

	s.chainB.Coordinator.CommitBlock(s.chainB)

	clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
	s.Require().True(ok)

	trustedHeight := clientState.LatestHeight
	header, err := s.chainB.IBCClientHeader(s.chainB.LatestCommittedHeader, trustedHeight)
	s.Require().NoError(err)
	s.Require().NotNil(header)

	msg, err := clienttypes.NewMsgUpdateClient(
		ibctesting.FirstClientID, header,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	)

	s.Require().NoError(err)

	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			clienttypes.EventTypeUpdateClient,
			sdk.NewAttribute(clienttypes.AttributeKeyClientID, ibctesting.FirstClientID),
			sdk.NewAttribute(clienttypes.AttributeKeyClientType, path.EndpointA.GetClientState().ClientType()),
			sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeight, path.EndpointA.GetClientLatestHeight().String()),
			sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeights, path.EndpointA.GetClientLatestHeight().String()),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&s.Suite, expectedEvents, events)
}
