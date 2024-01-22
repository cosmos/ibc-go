package keeper_test

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *KeeperTestSuite) TestMsgCreateClientEvents() {
	suite.SetupTest()
	path := ibctesting.NewPath(suite.chainA, suite.chainB)

	path.EndpointA.Counterparty.Chain.NextBlock()

	tmConfig, ok := path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig)
	suite.Require().True(ok)

	height := path.EndpointA.Counterparty.Chain.LatestCommittedHeader.GetHeight().(clienttypes.Height)
	clientState := ibctm.NewClientState(
		path.EndpointA.Counterparty.Chain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
		height, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
	consensusState := path.EndpointA.Counterparty.Chain.LatestCommittedHeader.ConsensusState()

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	)
	suite.Require().NoError(err)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			clienttypes.EventTypeCreateClient,
			sdk.NewAttribute(clienttypes.AttributeKeyClientID, ibctesting.FirstClientID),
			sdk.NewAttribute(clienttypes.AttributeKeyClientType, clientState.ClientType()),
			sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeight, clientState.GetLatestHeight().String()),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&suite.Suite, expectedEvents, events)
}

func (suite *KeeperTestSuite) TestMsgUpdateClientEvents() {
	suite.SetupTest()
	path := ibctesting.NewPath(suite.chainA, suite.chainB)

	suite.Require().NoError(path.EndpointA.CreateClient())

	suite.chainB.Coordinator.CommitBlock(suite.chainB)

	header, err := suite.chainA.ConstructUpdateTMClientHeader(suite.chainB, ibctesting.FirstClientID)
	suite.Require().NoError(err)
	suite.Require().NotNil(header)

	msg, err := clienttypes.NewMsgUpdateClient(
		ibctesting.FirstClientID, header,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	)

	suite.Require().NoError(err)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			clienttypes.EventTypeUpdateClient,
			sdk.NewAttribute(clienttypes.AttributeKeyClientID, ibctesting.FirstClientID),
			sdk.NewAttribute(clienttypes.AttributeKeyClientType, path.EndpointA.GetClientState().ClientType()),
			sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeight, path.EndpointA.GetClientState().GetLatestHeight().String()),
			sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeights, path.EndpointA.GetClientState().GetLatestHeight().String()),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&suite.Suite, expectedEvents, events)
}

func (suite *KeeperTestSuite) TestMsgUpgradeClientEvents() {
	var upgradedClient exported.ClientState
	var upgradedClientProof, upgradedConsensusStateProof []byte

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(path)

	clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
	revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

	newChainID, err := clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
	suite.Require().NoError(err)

	upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+trustingPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.GetLatestHeight().GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
	upgradedClient = upgradedClient.ZeroCustomFields()
	upgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
	suite.Require().NoError(err)

	upgradedConsState := &ibctm.ConsensusState{
		NextValidatorsHash: []byte("nextValsHash"),
	}

	lastHeight := clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

	upgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
	suite.Require().NoError(err)

	// zero custom fields and store in upgrade store
	err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)
	suite.Require().NoError(err)
	err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz)
	suite.Require().NoError(err)

	// commit upgrade store changes and update clients
	suite.coordinator.CommitBlock(suite.chainB)
	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
	suite.Require().True(found)

	upgradedClientProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())
	upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), cs.GetLatestHeight().GetRevisionHeight())

	msg, err := clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState, upgradedClientProof, upgradedConsensusStateProof, path.EndpointA.Chain.SenderAccount.GetAddress().String())
	suite.Require().NotNil(msg)
	suite.Require().NoError(err)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	events := res.Events
	expectedEvents := sdk.Events{
		sdk.NewEvent(
			clienttypes.EventTypeUpgradeClient,
			sdk.NewAttribute(clienttypes.AttributeKeyClientID, ibctesting.FirstClientID),
			sdk.NewAttribute(clienttypes.AttributeKeyClientType, path.EndpointA.GetClientState().ClientType()),
			sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeight, path.EndpointA.GetClientState().GetLatestHeight().String()),
		),
	}.ToABCIEvents()

	var indexSet map[string]struct{}
	expectedEvents = sdk.MarkEventsToIndex(expectedEvents, indexSet)
	ibctesting.AssertEvents(&suite.Suite, expectedEvents, events)
}
