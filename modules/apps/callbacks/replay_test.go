package ibccallbacks_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/testing/simapp"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *CallbacksTestSuite) TestTransferTimeoutReplayProtection() {
	testCases := []struct {
		name         string
		transferMemo string
	}{
		{
			"success: REPLAY TIMEOUT",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			callbackCount := 0 // used to count the number of times the timeout callback is called.

			// This simulates a contract which submits a replay timeout packet:
			GetSimApp(s.chainA).MockContractKeeper.IBCOnTimeoutPacketCallbackFn = func(
				cachedCtx sdk.Context,
				packet channeltypes.Packet,
				_ sdk.AccAddress,
				_, _, _ string,
			) error {
				// only replay the timeout packet twice. We could replay it more times
				callbackCount++
				if callbackCount == 2 {
					return nil
				}

				// construct the timeoutMsg
				packetKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
				counterparty := s.path.EndpointA.Counterparty
				proof, proofHeight := counterparty.QueryProof(packetKey)
				nextSeqRecv, found := counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(counterparty.Chain.GetContext(), counterparty.ChannelConfig.PortID, counterparty.ChannelID)
				s.Require().True(found)

				timeoutMsg := channeltypes.NewMsgTimeout(
					packet, nextSeqRecv,
					proof, proofHeight, s.chainA.SenderAccount.GetAddress().String(),
				)

				// in a real scenario, this should be s.chainA.SendMsg
				// but I couldn't get it to work due to the way our testsuite is setup
				// (we increment the account sequence after full block execution)
				// (we also don't support sending messages from other accounts)
				res, err := GetSimApp(s.chainA).IBCKeeper.Timeout(cachedCtx, timeoutMsg)
				s.Require().NoError(err)
				s.Require().Equal(channeltypes.NOOP, res.Result)

				return nil
			}

			// fund escrow account
			fund := ibctesting.TestCoins.Add(ibctesting.TestCoin).Add(ibctesting.TestCoin)
			escrowAddress := transfertypes.GetEscrowAddress(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
			err := GetSimApp(s.chainA).BankKeeper.SendCoins(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), escrowAddress, fund)
			s.Require().NoError(err)

			// set total escrow for denom
			GetSimApp(s.chainA).TransferKeeper.SetTotalEscrowForDenom(s.chainA.GetContext(), fund[0])

			// save initial balance of sender
			initialBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

			// amountTransferred := ibctesting.TestCoin
			s.ExecuteTransferTimeout(tc.transferMemo)

			// check that the callback is executed 1 times
			s.Require().Equal(1, callbackCount)

			afterBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

			// expected is not a malicious amount
			s.Require().Equal(initialBalance.Amount, afterBalance.Amount)
		})
	}
}

func (s *CallbacksTestSuite) TestTransferErrorAcknowledgementReplayProtection() {
	testCases := []struct {
		name         string
		transferMemo string
	}{
		{
			"success: REPLAY ERROR ACK",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			callbackCount := 0 // used to count the number of times the ack callback is called.

			// This simulates a contract which submits a replay ack packet:
			GetSimApp(s.chainA).MockContractKeeper.IBCOnAcknowledgementPacketCallbackFn = func(
				cachedCtx sdk.Context,
				packet channeltypes.Packet,
				ack []byte,
				_ sdk.AccAddress,
				_, _, _ string,
			) error {
				// only replay the ack packet twice. We could replay it more times
				callbackCount++
				if callbackCount == 2 {
					return nil
				}

				// construct the ackMsg
				packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
				proof, proofHeight := s.path.EndpointA.Counterparty.QueryProof(packetKey)

				ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, s.chainA.SenderAccount.GetAddress().String())

				// in a real scenario, this should be s.chainA.SendMsg
				// but I couldn't get it to work due to the way our testsuite is setup
				// (we increment the account sequence after full block execution)
				// (we also don't support sending messages from other accounts)
				res, err := GetSimApp(s.chainA).IBCKeeper.Acknowledgement(cachedCtx, ackMsg)
				s.Require().NoError(err)
				s.Require().Equal(channeltypes.NOOP, res.Result) // no-op because this is a redundant replay

				return nil
			}

			// fund escrow account
			fund := ibctesting.TestCoins.Add(ibctesting.TestCoin).Add(ibctesting.TestCoin)
			escrowAddress := transfertypes.GetEscrowAddress(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
			err := GetSimApp(s.chainA).BankKeeper.SendCoins(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), escrowAddress, fund)
			s.Require().NoError(err)

			// set total escrow for denom
			GetSimApp(s.chainA).TransferKeeper.SetTotalEscrowForDenom(s.chainA.GetContext(), fund[0])

			// save initial balance of sender
			initialBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

			// amountTransferred := ibctesting.TestCoin
			s.ExecuteFailedTransfer(tc.transferMemo)

			// check that the callback is executed 1 times
			s.Require().Equal(1, callbackCount)

			expBalance := initialBalance

			afterBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

			// expected is not a malicious amount
			s.Require().Equal(expBalance.Amount, afterBalance.Amount)
		})
	}
}

func (s *CallbacksTestSuite) TestTransferSuccessAcknowledgementReplayProtection() {
	testCases := []struct {
		name         string
		transferMemo string
	}{
		{
			"success: REPLAY SUCCESS ACK",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			callbackCount := 0 // used to count the number of times the ack callback is called.

			// This simulates a contract which submits a replay ack packet:
			GetSimApp(s.chainA).MockContractKeeper.IBCOnAcknowledgementPacketCallbackFn = func(
				cachedCtx sdk.Context,
				packet channeltypes.Packet,
				ack []byte,
				_ sdk.AccAddress,
				_, _, _ string,
			) error {
				// only replay the ack packet twice. We could replay it more times
				callbackCount++
				if callbackCount == 2 {
					return nil
				}

				// construct the ackMsg
				packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
				proof, proofHeight := s.path.EndpointA.Counterparty.QueryProof(packetKey)

				ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, s.chainA.SenderAccount.GetAddress().String())

				// in a real scenario, this should be s.chainA.SendMsg
				// but I couldn't get it to work due to the way our testsuite is setup
				// (we increment the account sequence after full block execution)
				// (we also don't support sending messages from other accounts)
				res, err := GetSimApp(s.chainA).IBCKeeper.Acknowledgement(cachedCtx, ackMsg)
				s.Require().NoError(err)
				s.Require().Equal(channeltypes.NOOP, res.Result) // no-op because this is a redundant replay

				return nil
			}

			// save initial balance of sender
			initialBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

			// amountTransferred := ibctesting.TestCoin
			s.ExecuteTransfer(tc.transferMemo, true)

			// check that the callback is executed 1 times
			s.Require().Equal(1, callbackCount)

			expBalance := initialBalance.Sub(ibctesting.TestCoin)

			afterBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

			// expected is not a malicious amount
			s.Require().Equal(expBalance.Amount, afterBalance.Amount)
		})
	}
}

func (s *CallbacksTestSuite) TestTransferRecvPacketReplayProtection() {
	testCases := []struct {
		name         string
		transferMemo string
	}{
		{
			"success: REPLAY RECV PACKET",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.SuccessContract),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			callbackCount := 0 // used to count the number of times the RecvPacket callback is called.

			// Write a contract in Chain B that tries to execute receive packet 2 times!
			GetSimApp(s.chainB).MockContractKeeper.IBCReceivePacketCallbackFn = func(
				cachedCtx sdk.Context,
				packet ibcexported.PacketI,
				_ ibcexported.Acknowledgement,
				_, _ string,
			) error {
				callbackCount++
				if callbackCount == 2 {
					return nil
				}

				packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				proof, proofHeight := s.path.EndpointB.Counterparty.Chain.QueryProof(packetKey)

				recvMsg := channeltypes.NewMsgRecvPacket(packet.(channeltypes.Packet), proof, proofHeight, s.chainB.SenderAccount.GetAddress().String())

				// send again
				res, err := GetSimApp(s.chainB).IBCKeeper.RecvPacket(cachedCtx, recvMsg)
				s.Require().NoError(err)
				s.Require().Equal(channeltypes.NOOP, res.Result) // no-op because this is a redundant replay

				return nil
			}

			// save initial balance of receiver
			denom := transfertypes.NewDenom(sdk.DefaultBondDenom, transfertypes.NewHop(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID))
			initialBalance := GetSimApp(s.chainB).BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), denom.IBCDenom())

			// execute the transfer
			s.ExecuteTransfer(tc.transferMemo, true)

			// check that the callback is executed 1 times
			s.Require().Equal(1, callbackCount)

			// expected is not a malicious amount
			expBalance := initialBalance.Add(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount))

			afterBalance := GetSimApp(s.chainB).BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), denom.IBCDenom())

			s.Require().Equal(expBalance.Amount, afterBalance.Amount)
		})
	}
}

// ExecuteFailedTransfer executes a transfer message on chainA for ibctesting.TestCoin (100 "stake").
// The transfer will fail on RecvPacket and an error acknowledgement will be sent back to chainA.
func (s *CallbacksTestSuite) ExecuteFailedTransfer(memo string) {
	GetSimApp(s.chainB).TransferKeeper.SetParams(s.chainB.GetContext(), transfertypes.Params{
		ReceiveEnabled: false,
		SendEnabled:    true,
	})

	defer GetSimApp(s.chainB).TransferKeeper.SetParams(s.chainB.GetContext(), transfertypes.DefaultParams())

	escrowAddress := transfertypes.GetEscrowAddress(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	// record the balance of the escrow address before the transfer
	escrowBalance := GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	// record the balance of the receiving address before the transfer
	denom := transfertypes.NewDenom(sdk.DefaultBondDenom, transfertypes.NewHop(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID))
	receiverBalance := GetSimApp(s.chainB).BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), denom.IBCDenom())

	amount := ibctesting.TestCoin
	msg := transfertypes.NewMsgTransfer(
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		amount,
		s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(),
		clienttypes.NewHeight(1, 100), 0, memo,
	)

	res, err := s.chainA.SendMsgs(msg)
	if err != nil {
		return // we return if send packet is rejected
	}

	packet, err := ibctesting.ParseV1PacketFromEvents(res.GetEvents())
	s.Require().NoError(err)

	// relay send
	err = s.path.RelayPacket(packet)
	s.Require().NoError(err) // relay committed

	// check that the escrow address balance hasn't changed
	s.Require().Equal(escrowBalance, GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom))
	// check that the receiving address balance hasn't changed
	s.Require().Equal(receiverBalance, GetSimApp(s.chainB).BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), denom.IBCDenom()))
}
