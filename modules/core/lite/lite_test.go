package lite_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"
	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type LiteTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *LiteTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	// TODO: remove
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

func TestLiteTestSuite(t *testing.T) {
	suite.Run(t, new(LiteTestSuite))
}

func (suite *LiteTestSuite) TestHappyPath() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupClients()

	cosmosMerklePath := commitmenttypes.NewMerklePath("ibc", "")
	provideCounterpartyMsgA := clienttypes.MsgProvideCounterparty{
		ClientId:         path.EndpointA.ClientID,
		CounterpartyId:   path.EndpointB.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
		Signer:           path.EndpointA.Chain.SenderAccount.GetAddress().String(),
	}
	provideCounterpartyMsgB := clienttypes.MsgProvideCounterparty{
		ClientId:         path.EndpointB.ClientID,
		CounterpartyId:   path.EndpointA.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
		Signer:           path.EndpointB.Chain.SenderAccount.GetAddress().String(),
	}

	// setup counterparties
	_, err := path.EndpointA.Chain.SendMsgs(&provideCounterpartyMsgA)
	suite.Require().NoError(err)
	_, err = path.EndpointB.Chain.SendMsgs(&provideCounterpartyMsgB)
	suite.Require().NoError(err)

	expectedCounterpartyA := clienttypes.LiteCounterparty{
		ClientId:         path.EndpointB.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok := path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(path.EndpointA.Chain.GetContext(), path.EndpointA.ClientID)
	suite.Require().True(ok)
	suite.Require().Equal(expectedCounterpartyA, counterparty)

	expectedCounterpartyB := clienttypes.LiteCounterparty{
		ClientId:         path.EndpointA.ClientID,
		MerklePathPrefix: &cosmosMerklePath,
	}
	counterparty, ok = path.EndpointB.Chain.App.GetIBCKeeper().ClientKeeper.GetCounterparty(path.EndpointB.Chain.GetContext(), path.EndpointB.ClientID)
	suite.Require().True(ok)
	suite.Require().Equal(expectedCounterpartyB, counterparty)

	originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	amount := math.NewInt(100)
	token := sdk.NewCoin(sdk.DefaultBondDenom, amount)

	transferMsg := transfertypes.MsgTransfer{
		SourcePort:       transfertypes.PortID,
		SourceChannel:    path.EndpointA.ClientID,
		Token:            token,
		Sender:           path.EndpointA.Chain.SenderAccount.GetAddress().String(),
		Receiver:         path.EndpointB.Chain.SenderAccount.GetAddress().String(),
		TimeoutHeight:    clienttypes.NewHeight(1, 100),
		TimeoutTimestamp: 0,
		Memo:             "",
		DestPort:         transfertypes.PortID,
		DestChannel:      path.EndpointB.ClientID,
	}
	res, err := path.EndpointA.Chain.SendMsgs(&transferMsg)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)

	err = path.RelayPacket(packet)
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
	coinSentFromAToB := transfertypes.GetTransferCoin(transfertypes.PortID, path.EndpointB.ClientID, sdk.DefaultBondDenom, amount)
	suite.Require().Equal(coinSentFromAToB, balance)

	// relay send from chain B to chain A

}
