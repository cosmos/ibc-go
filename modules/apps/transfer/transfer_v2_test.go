package transfer_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

type TransferV2TestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *TransferV2TestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}

// Constructs the following sends based on the established channels/connections
// 1 - from chainA to chainB
// 2 - from chainB to chainC
// 3 - from chainC to chainB
func (suite *TransferV2TestSuite) TestHandleMsgV2Transfer() {
	suite.SetupTest() // reset

	// setup between chainA and chainB
	// NOTE:
	// pathAToB.EndpointA = endpoint on chainA
	// pathAToB.EndpointB = endpoint on chainB
	pathAToB := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	pathAToB.Setup()
	traceAToB := types.NewHop(pathAToB.EndpointB.ChannelConfig.PortID, pathAToB.EndpointB.ChannelID)

	originalBalances := sdk.NewCoins()
	originalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	originalBalances = originalBalances.Add(originalBalance)

	timeoutHeight := clienttypes.NewHeight(1, 110)

	amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
	suite.Require().True(ok)
	originalCoins := sdk.NewCoins()

	coinToSendToB := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	originalCoins = originalCoins.Add(coinToSendToB)

	// send from chainA to chainB
	msg := types.NewMsgTransfer(pathAToB.EndpointA.ChannelConfig.PortID, pathAToB.EndpointA.ChannelID, originalCoins, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "", nil)
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed
	suite.Require().NoError(pathAToB.EndpointB.UpdateClient())

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	suite.Require().NoError(err)

	// relay send
	err = pathAToB.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
	coinsSentFromAToB := sdk.NewCoins()
	for _, coin := range originalCoins {
		// check that the balance for chainA is updated
		chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
		suite.Require().Equal(originalBalances.AmountOf(coin.Denom).Sub(amount).Int64(), chainABalance.Amount.Int64())

		// check that module account escrow address has locked the tokens
		chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, coin.Denom)
		suite.Require().Equal(coin, chainAEscrowBalance)

		// check that voucher exists on chain B
		chainBDenom := types.NewDenom(coin.Denom, traceAToB)
		chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), chainBDenom.IBCDenom())
		coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), amount)
		suite.Require().Equal(coinSentFromAToB, chainBBalance)

		coinsSentFromAToB = coinsSentFromAToB.Add(coinSentFromAToB)
	}

	suite.Require().NoError(pathAToB.EndpointB.UpdateClient())
	suite.Require().NoError(pathAToB.EndpointA.UpdateClient())

	ftpd := types.FungibleTokenPacketDataV2{
		Tokens: []types.Token{
			{
				// "transfer/channel-0/stake"
				Denom:  types.NewDenom(sdk.DefaultBondDenom, traceAToB),
				Amount: "100",
			},
		},
		Sender:     suite.chainB.SenderAccount.GetAddress().String(),
		Receiver:   suite.chainA.SenderAccount.GetAddress().String(),
		Memo:       "",
		Forwarding: types.ForwardingPacketData{},
	}

	bz, err := suite.chainB.Codec.Marshal(&ftpd)
	suite.Require().NoError(err)

	timeoutTimestamp := suite.chainB.GetTimeoutTimestamp()
	msgSendPacket := &channeltypesv2.MsgSendPacket{
		SourceId:         pathAToB.EndpointB.ChannelID,
		TimeoutTimestamp: timeoutTimestamp,
		PacketData: []channeltypes.PacketData{
			{
				SourcePort:      "transfer",
				DestinationPort: "transfer",
				Payload: channeltypes.Payload{
					Encoding: "json",
					Version:  types.V2,
					Value:    bz,
				},
			},
		},
		Signer: suite.chainB.SenderAccount.GetAddress().String(),
	}

	res, err = suite.chainB.SendMsgs(msgSendPacket)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NoError(pathAToB.EndpointA.UpdateClient())
	suite.Require().NoError(pathAToB.EndpointB.UpdateClient())

	packetKey := host.PacketCommitmentKey(host.SentinelV2PortID, pathAToB.EndpointB.ClientID, 1)
	proof, proofHeight := pathAToB.EndpointB.QueryProof(packetKey)
	suite.Require().NotNil(proof)
	suite.Require().False(proofHeight.IsZero())

	packetV2 := channeltypesv2.NewPacketV2(1, pathAToB.EndpointB.ChannelID, pathAToB.EndpointA.ChannelID, timeoutTimestamp, channeltypes.PacketData{
		SourcePort:      "transfer",
		DestinationPort: "transfer",
		Payload: channeltypes.Payload{
			Version:  types.V2,
			Encoding: "json",
			Value:    bz,
		},
	})

	msgRecvPacket := &channeltypesv2.MsgRecvPacket{
		Packet:          packetV2,
		ProofCommitment: proof,
		ProofHeight:     proofHeight,
		Signer:          suite.chainA.SenderAccount.GetAddress().String(),
	}

	res, err = suite.chainA.SendMsgs(msgRecvPacket)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NoError(pathAToB.EndpointB.UpdateClient())

}

func TestTransferV2TestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferV2TestSuite))
}
