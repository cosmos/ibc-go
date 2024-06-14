package lite_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type LiteTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *LiteTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))

	// TODO: remove
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
	suite.coordinator.CommitNBlocks(suite.chainC, 2)
}

func TestLiteTestSuite(t *testing.T) {
	suite.Run(t, new(LiteTestSuite))
}

func (suite *LiteTestSuite) TestHappyPath() {
	pathAtoB := ibctesting.NewPath(suite.chainA, suite.chainB)
	pathAtoB.SetupClients()

	cosmosMerklePath := suite.chainA.GetPrefix() // ChainA and B have the same prefix
	provideCounterpartyMsgA := clienttypes.MsgProvideCounterparty{
		ClientId:         pathAtoB.EndpointA.ClientID,
		CounterpartyId:   pathAtoB.EndpointB.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
		Signer:           pathAtoB.EndpointA.Chain.SenderAccount.GetAddress().String(),
	}
	provideCounterpartyMsgB := clienttypes.MsgProvideCounterparty{
		ClientId:         pathAtoB.EndpointB.ClientID,
		CounterpartyId:   pathAtoB.EndpointA.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
		Signer:           pathAtoB.EndpointB.Chain.SenderAccount.GetAddress().String(),
	}

	// setup counterparties
	_, err := pathAtoB.EndpointA.Chain.SendMsgs(&provideCounterpartyMsgA)
	suite.Require().NoError(err)
	_, err = pathAtoB.EndpointB.Chain.SendMsgs(&provideCounterpartyMsgB)
	suite.Require().NoError(err)

	expectedCounterpartyAtoB := clienttypes.LiteCounterparty{
		ClientId:         pathAtoB.EndpointB.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok := pathAtoB.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(pathAtoB.EndpointA.Chain.GetContext(), pathAtoB.EndpointA.ClientID)
	suite.Require().True(ok)
	suite.Require().Equal(expectedCounterpartyAtoB, counterparty)

	expectedCounterpartyBtoA := clienttypes.LiteCounterparty{
		ClientId:         pathAtoB.EndpointA.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok = pathAtoB.EndpointB.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(pathAtoB.EndpointB.Chain.GetContext(), pathAtoB.EndpointB.ClientID)
	suite.Require().True(ok)
	suite.Require().Equal(expectedCounterpartyBtoA, counterparty)

	originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	amount := math.NewInt(100)
	token := sdk.NewCoin(sdk.DefaultBondDenom, amount)

	transferMsg := transfertypes.MsgTransfer{
		SourcePort:       transfertypes.PortID,
		SourceChannel:    pathAtoB.EndpointA.ClientID,
		Token:            token,
		Sender:           pathAtoB.EndpointA.Chain.SenderAccount.GetAddress().String(),
		Receiver:         pathAtoB.EndpointB.Chain.SenderAccount.GetAddress().String(),
		TimeoutHeight:    clienttypes.NewHeight(1, 100),
		TimeoutTimestamp: 0,
		Memo:             "",
		DestPort:         transfertypes.PortID,
		DestChannel:      pathAtoB.EndpointB.ClientID,
	}
	res, err := pathAtoB.EndpointA.Chain.SendMsgs(&transferMsg)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)

	err = pathAtoB.RelayPacket(packet)
	suite.Require().NoError(err)

	// check that module account escrow address has locked the tokens
	escrowAddress := transfertypes.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
	balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	suite.Require().Equal(token, balance)

	// check that balance on chain A is updated
	balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	suite.Require().Equal(originalBalance.Sub(token), balance)
	// check that voucher exists on chain B
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))
	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	// NOTE: we are using client IDs instead of channel IDs here.
	coinSentFromAToB := transfertypes.GetTransferCoin(transfertypes.PortID, pathAtoB.EndpointB.ClientID, sdk.DefaultBondDenom, amount)
	suite.Require().Equal(coinSentFromAToB, balance)

	// relay send from chain B to chain A
	// setup between chainB to chainC
	// NOTE:
	// pathBtoC.EndpointA = endpoint on chainB
	// pathBtoC.EndpointB = endpoint on chainC
	pathBtoC := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	pathBtoC.SetupClients()

	provideCounterpartyMsgBtoC := clienttypes.MsgProvideCounterparty{
		ClientId:         pathBtoC.EndpointA.ClientID,
		CounterpartyId:   pathBtoC.EndpointB.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
		Signer:           pathBtoC.EndpointA.Chain.SenderAccount.GetAddress().String(),
	}
	provideCounterpartyMsgCtoB := clienttypes.MsgProvideCounterparty{
		ClientId:         pathBtoC.EndpointB.ClientID,
		CounterpartyId:   pathBtoC.EndpointA.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
		Signer:           pathBtoC.EndpointB.Chain.SenderAccount.GetAddress().String(),
	}

	// setup counterparties
	_, err = pathBtoC.EndpointA.Chain.SendMsgs(&provideCounterpartyMsgBtoC)
	suite.Require().NoError(err)
	_, err = pathBtoC.EndpointB.Chain.SendMsgs(&provideCounterpartyMsgCtoB)
	suite.Require().NoError(err)

	expectedCounterpartyBtoC := clienttypes.LiteCounterparty{
		ClientId:         pathBtoC.EndpointB.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok = pathBtoC.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(pathBtoC.EndpointA.Chain.GetContext(), pathBtoC.EndpointA.ClientID)
	suite.Require().True(ok)
	suite.Require().Equal(expectedCounterpartyBtoC, counterparty)

	expectedCounterpartyCtoB := clienttypes.LiteCounterparty{
		ClientId:         pathBtoC.EndpointA.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok = pathBtoC.EndpointB.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(pathBtoC.EndpointB.Chain.GetContext(), pathBtoC.EndpointB.ClientID)
	suite.Require().True(ok)
	suite.Require().Equal(expectedCounterpartyCtoB, counterparty)

	transferMsg = transfertypes.MsgTransfer{
		SourcePort:       transfertypes.PortID,
		SourceChannel:    pathBtoC.EndpointA.ClientID,
		Token:            coinSentFromAToB,
		Sender:           pathBtoC.EndpointA.Chain.SenderAccount.GetAddress().String(),
		Receiver:         pathBtoC.EndpointB.Chain.SenderAccount.GetAddress().String(),
		TimeoutHeight:    clienttypes.NewHeight(1, 100),
		TimeoutTimestamp: 0,
		Memo:             "",
		DestPort:         transfertypes.PortID,
		DestChannel:      pathBtoC.EndpointB.ClientID,
	}
	res, err = suite.chainB.SendMsgs(&transferMsg)
	suite.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.Events)
	suite.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
	fullDenomPath := transfertypes.GetPrefixedDenom(transfertypes.PortID, pathBtoC.EndpointB.ClientID, voucherDenomTrace.GetFullDenomPath())

	// check that the balance is updated on chainC
	coinSentFromBToC := sdk.NewCoin(transfertypes.ParseDenomTrace(fullDenomPath).IBCDenom(), amount)
	balance = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
	suite.Require().Equal(coinSentFromBToC, balance)

	// check that balance on chain B is empty
	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
	suite.Require().Zero(balance.Amount.Int64())

	// send from chainC back to chainB
	transferMsg = transfertypes.MsgTransfer{
		SourcePort:       transfertypes.PortID,
		SourceChannel:    pathBtoC.EndpointB.ClientID,
		Token:            coinSentFromBToC,
		Sender:           pathBtoC.EndpointB.Chain.SenderAccount.GetAddress().String(),
		Receiver:         pathBtoC.EndpointA.Chain.SenderAccount.GetAddress().String(),
		TimeoutHeight:    clienttypes.NewHeight(1, 100),
		TimeoutTimestamp: 0,
		Memo:             "",
		DestPort:         transfertypes.PortID,
		DestChannel:      pathBtoC.EndpointA.ClientID,
	}
	res, err = suite.chainC.SendMsgs(&transferMsg)
	suite.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.Events)
	suite.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	// check that balance on chain A is updated
	balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	suite.Require().Equal(originalBalance.Sub(token), balance)

	// check that balance on chain B has the transferred amount
	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromAToB.Denom)
	suite.Require().Equal(coinSentFromAToB, balance)

	// check that module account escrow address is empty
	escrowAddress = transfertypes.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddress, coinSentFromAToB.Denom)
	suite.Require().Zero(balance.Amount.Int64())

	// check that balance on chain C is empty
	balance = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	suite.Require().Zero(balance.Amount.Int64())
}
