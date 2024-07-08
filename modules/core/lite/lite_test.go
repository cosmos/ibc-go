package lite_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type LiteTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (s *LiteTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))

	// TODO: remove
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	s.coordinator.CommitNBlocks(s.chainA, 2)
	s.coordinator.CommitNBlocks(s.chainB, 2)
	s.coordinator.CommitNBlocks(s.chainC, 2)
}

func TestLiteTestSuite(t *testing.T) {
	suite.Run(t, new(LiteTestSuite))
}

func (s *LiteTestSuite) TestHappyPath() {
	pathAtoB := ibctesting.NewPath(s.chainA, s.chainB)
	pathAtoB.SetupClients()

	cosmosMerklePath := commitmenttypes.NewMerklePath([]byte("ibc"), []byte(""))
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
	s.Require().NoError(err)
	_, err = pathAtoB.EndpointB.Chain.SendMsgs(&provideCounterpartyMsgB)
	s.Require().NoError(err)

	expectedCounterpartyAtoB := clienttypes.LiteCounterparty{
		ClientId:         pathAtoB.EndpointB.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok := pathAtoB.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(pathAtoB.EndpointA.Chain.GetContext(), pathAtoB.EndpointA.ClientID)
	s.Require().True(ok)
	s.Require().Equal(expectedCounterpartyAtoB, counterparty)

	expectedCounterpartyBtoA := clienttypes.LiteCounterparty{
		ClientId:         pathAtoB.EndpointA.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok = pathAtoB.EndpointB.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(pathAtoB.EndpointB.Chain.GetContext(), pathAtoB.EndpointB.ClientID)
	s.Require().True(ok)
	s.Require().Equal(expectedCounterpartyBtoA, counterparty)

	originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
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
	s.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)

	err = pathAtoB.RelayPacket(packet)
	s.Require().NoError(err)

	// check that module account escrow address has locked the tokens
	escrowAddress := transfertypes.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
	balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	s.Require().Equal(token, balance)

	// check that balance on chain A is updated
	balance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	s.Require().Equal(originalBalance.Sub(token), balance)
	// check that voucher exists on chain B
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))
	balance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	// NOTE: we are using client IDs instead of channel IDs here.
	coinSentFromAToB := transfertypes.GetTransferCoin(transfertypes.PortID, pathAtoB.EndpointB.ClientID, sdk.DefaultBondDenom, amount)
	s.Require().Equal(coinSentFromAToB, balance)

	// relay send from chain B to chain A
	// setup between chainB to chainC
	// NOTE:
	// pathBtoC.EndpointA = endpoint on chainB
	// pathBtoC.EndpointB = endpoint on chainC
	pathBtoC := ibctesting.NewTransferPath(s.chainB, s.chainC)
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
	s.Require().NoError(err)
	_, err = pathBtoC.EndpointB.Chain.SendMsgs(&provideCounterpartyMsgCtoB)
	s.Require().NoError(err)

	expectedCounterpartyBtoC := clienttypes.LiteCounterparty{
		ClientId:         pathBtoC.EndpointB.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok = pathBtoC.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(pathBtoC.EndpointA.Chain.GetContext(), pathBtoC.EndpointA.ClientID)
	s.Require().True(ok)
	s.Require().Equal(expectedCounterpartyBtoC, counterparty)

	expectedCounterpartyCtoB := clienttypes.LiteCounterparty{
		ClientId:         pathBtoC.EndpointA.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok = pathBtoC.EndpointB.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(pathBtoC.EndpointB.Chain.GetContext(), pathBtoC.EndpointB.ClientID)
	s.Require().True(ok)
	s.Require().Equal(expectedCounterpartyCtoB, counterparty)

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
	res, err = s.chainB.SendMsgs(&transferMsg)
	s.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	s.Require().NoError(err) // relay committed

	// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
	fullDenomPath := transfertypes.GetPrefixedDenom(transfertypes.PortID, pathBtoC.EndpointB.ClientID, voucherDenomTrace.GetFullDenomPath())

	// check that the balance is updated on chainC
	coinSentFromBToC := sdk.NewCoin(transfertypes.ParseDenomTrace(fullDenomPath).IBCDenom(), amount)
	balance = s.chainC.GetSimApp().BankKeeper.GetBalance(s.chainC.GetContext(), s.chainC.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
	s.Require().Equal(coinSentFromBToC, balance)

	// check that balance on chain B is empty
	balance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
	s.Require().Zero(balance.Amount.Int64())

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
	res, err = s.chainC.SendMsgs(&transferMsg)
	s.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	s.Require().NoError(err) // relay committed

	// check that balance on chain A is updated
	balance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	s.Require().Equal(originalBalance.Sub(token), balance)

	// check that balance on chain B has the transferred amount
	balance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), coinSentFromAToB.Denom)
	s.Require().Equal(coinSentFromAToB, balance)

	// check that module account escrow address is empty
	escrowAddress = transfertypes.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
	balance = s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), escrowAddress, coinSentFromAToB.Denom)
	s.Require().Zero(balance.Amount.Int64())

	// check that balance on chain C is empty
	balance = s.chainC.GetSimApp().BankKeeper.GetBalance(s.chainC.GetContext(), s.chainC.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	s.Require().Zero(balance.Amount.Int64())
}
