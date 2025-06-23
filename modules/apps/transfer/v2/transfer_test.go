package v2_test

import (
	"time"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

func (s *TransferTestSuite) TestTransferV2Flow() {
	originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
	s.Require().True(ok)
	originalCoin := sdk.NewCoin(sdk.DefaultBondDenom, amount)

	token := types.Token{
		Denom:  types.Denom{Base: originalCoin.Denom},
		Amount: originalCoin.Amount.String(),
	}

	transferData := types.NewFungibleTokenPacketData(token.Denom.Path(), token.Amount, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), "")
	bz := s.chainA.Codec.MustMarshal(&transferData)
	payload := channeltypesv2.NewPayload(types.PortID, types.PortID, types.V1, types.EncodingProtobuf, bz)

	// Set a timeout of 1 hour from the current block time on receiver chain
	timeout := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

	packet, err := s.pathAToB.EndpointA.MsgSendPacket(timeout, payload)
	s.Require().NoError(err)

	err = s.pathAToB.EndpointA.RelayPacket(packet)
	s.Require().NoError(err)

	escrowAddress := types.GetEscrowAddress(types.PortID, s.pathAToB.EndpointA.ClientID)
	// check that the balance for chainA is updated
	chainABalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
	s.Require().Equal(originalBalance.Amount.Sub(amount).Int64(), chainABalance.Amount.Int64())

	// check that module account escrow address has locked the tokens
	chainAEscrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, originalCoin.Denom)
	s.Require().Equal(originalCoin, chainAEscrowBalance)

	traceAToB := types.NewHop(types.PortID, s.pathAToB.EndpointB.ClientID)

	// check that voucher exists on chain B
	chainBDenom := types.NewDenom(originalCoin.Denom, traceAToB)
	chainBBalance := s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), chainBDenom.IBCDenom())
	coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), amount)
	s.Require().Equal(coinSentFromAToB, chainBBalance)
}

func (s *TransferTestSuite) TestMultiPayloadTransferV2Flow() {
	mockPayload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)
	mockErrPayload := mockv2.NewErrorMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

	var (
		timeout  uint64
		payload  channeltypesv2.Payload
		payloads []channeltypesv2.Payload
	)

	type expResult int
	const (
		success expResult = iota
		sendError
		recvError
		ackError
		timeoutError
	)

	testCases := []struct {
		name     string
		malleate func()
		expRes   expResult
	}{
		{
			name: "success with transfer payloads",
			malleate: func() {
				payloads = []channeltypesv2.Payload{payload, payload}
			},
			expRes: success,
		},
		{
			name: "success with transfer and mock payloads",
			malleate: func() {
				payloads = []channeltypesv2.Payload{payload, mockPayload, mockPayload, payload}
			},
			expRes: success,
		},
		{
			name: "send error should revert transfer",
			malleate: func() {
				// mock the send packet callback to return an error
				s.pathAToB.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnSendPacket = func(ctx sdk.Context, sourceChannel, destinationChannel string, sequence uint64, data channeltypesv2.Payload, signer sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
				payloads = []channeltypesv2.Payload{payload, mockPayload, payload}
			},
			expRes: sendError,
		},
		{
			name: "recv error on mock should revert transfer",
			malleate: func() {
				payloads = []channeltypesv2.Payload{payload, mockPayload, mockErrPayload, payload}
			},
			expRes: recvError,
		},
		{
			name: "ack error on mock should block refund on acknowledgement",
			malleate: func() {
				// mock the acknowledgement packet callback to return an error
				s.pathAToB.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnAcknowledgementPacket = func(ctx sdk.Context, sourceChannel, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, acknowledgement []byte, relayer sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
				payloads = []channeltypesv2.Payload{payload, mockPayload, mockPayload, payload}
			},
			expRes: ackError,
		},
		{
			name: "timeout error on mock should block refund on timeout",
			malleate: func() {
				// mock the timeout packet callback to return an error
				s.pathAToB.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnTimeoutPacket = func(ctx sdk.Context, sourceChannel, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
				// set the timeout to be 1 second from now so that the packet will timeout
				timeout = uint64(s.chainB.GetContext().BlockTime().Add(time.Second).Unix())
				payloads = []channeltypesv2.Payload{payload, mockPayload, mockPayload, payload}
			},
			expRes: timeoutError,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

			// total amount is the sum of all amounts in the payloads which is always 2 * amount
			totalAmount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			s.Require().True(ok)
			amount := totalAmount.QuoRaw(2) // divide by 2 to account for the two payloads
			originalCoin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
			totalCoin := sdk.NewCoin(originalCoin.Denom, totalAmount)

			token := types.Token{
				Denom:  types.Denom{Base: originalCoin.Denom},
				Amount: originalCoin.Amount.String(),
			}

			transferData := types.NewFungibleTokenPacketData(token.Denom.Path(), token.Amount, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), "")
			bz := s.chainA.Codec.MustMarshal(&transferData)

			payload = channeltypesv2.NewPayload(types.PortID, types.PortID, types.V1, types.EncodingProtobuf, bz)

			escrowAddress := types.GetEscrowAddress(types.PortID, s.pathAToB.EndpointA.ClientID)

			// Set a timeout of 1 hour from the current block time on receiver chain
			timeout = uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// malleate the test case to set up the payloads
			// and modulate test case behavior
			tc.malleate()

			packet, sendErr := s.pathAToB.EndpointA.MsgSendPacket(timeout, payloads...)

			if tc.expRes == sendError {
				s.Require().Error(sendErr, "expected error when sending packet with send error")
			} else {
				s.Require().NoError(sendErr, "unexpected error when sending packet")

				// relay the packet
				relayErr := s.pathAToB.EndpointA.RelayPacket(packet)

				// relayer should have error in response on ack error and timeout error
				// recv error should not return an error since the error is handled as error acknowledgement
				if tc.expRes == ackError || tc.expRes == timeoutError {
					s.Require().Error(relayErr, "expected error when relaying packet with acknowledgement error or timeout error")
				} else {
					s.Require().NoError(relayErr, "unexpected error when relaying packet")
				}
			}

			ctxA := s.pathAToB.EndpointA.Chain.GetContext()
			ctxB := s.pathAToB.EndpointB.Chain.GetContext()

			// GET TRANSFER STATE AFTER RELAY FOR TESTING CHECKS
			// get account balances after relaying packet
			chainABalance := s.chainA.GetSimApp().BankKeeper.GetBalance(ctxA, s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			chainAEscrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(ctxA, escrowAddress, originalCoin.Denom)

			traceAToB := types.NewHop(types.PortID, s.pathAToB.EndpointB.ClientID)

			// get chain B balance for voucer
			chainBDenom := types.NewDenom(originalCoin.Denom, traceAToB)
			chainBBalance := s.chainB.GetSimApp().BankKeeper.GetBalance(ctxB, s.chainB.SenderAccount.GetAddress(), chainBDenom.IBCDenom())

			// calculate the expected coin sent from chain A to chain B
			coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), amount.MulRaw(2))

			// GET IBC STATE AFTER RELAY FOR TESTING CHECKS
			nextSequenceSend, ok := s.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2.GetNextSequenceSend(s.pathAToB.EndpointA.Chain.GetContext(), s.pathAToB.EndpointA.ClientID)
			s.Require().True(ok)

			packetCommitment := s.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketCommitment(ctxA, packet.SourceClient, packet.Sequence)
			hasReceipt := s.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.HasPacketReceipt(ctxB, packet.DestinationClient, packet.Sequence)
			hasAck := s.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.HasPacketAcknowledgement(ctxB, packet.DestinationClient, packet.Sequence)

			switch tc.expRes {
			case success:
				// check transfer state after successful relay
				// check that the balance for chainA is updated
				s.Require().Equal(originalBalance.Amount.Sub(totalAmount), chainABalance.Amount, "chain A balance should be deducted after successful transfer")
				// check that module account escrow address has locked the tokens
				s.Require().Equal(totalCoin, chainAEscrowBalance, "escrow balance should be locked after successful transfer")
				// check that voucher exists on chain B
				s.Require().Equal(coinSentFromAToB, chainBBalance, "voucher balance should be updated after successful transfer")

				// check IBC state after successful relay
				s.Require().Equal(uint64(2), nextSequenceSend, "next sequence send was not incremented correctly")
				// packet commitment should be cleared
				s.Require().Nil(packetCommitment)

				// packet receipt and acknowledgement should be written
				s.Require().True(hasReceipt, "packet receipt should exist")
				s.Require().True(hasAck, "packet acknowledgement should exist")
			case sendError:
				// check transfer state after send error
				// check that the balance for chainA is unchanged
				s.Require().Equal(originalBalance.Amount, chainABalance.Amount, "chain A balance should be unchanged after send error")
				// check that module account escrow address has not locked the tokens
				s.Require().Equal(sdk.NewCoin(originalCoin.Denom, sdkmath.ZeroInt()), chainAEscrowBalance, "escrow balance should be zero after send error")
				// check that voucher does not exist on chain B
				s.Require().Equal(sdk.NewCoin(chainBDenom.IBCDenom(), sdkmath.ZeroInt()), chainBBalance, "voucher balance should be zero after send error")

				// check IBC state after send error
				s.Require().Equal(uint64(1), nextSequenceSend, "next sequence send should not be incremented after send error")
				// packet commitment should not exist
				s.Require().Nil(packetCommitment, "packet commitment should not exist after send error")
				// packet receipt and acknowledgement should not be written
				s.Require().False(hasReceipt, "packet receipt should not exist after send error")
				s.Require().False(hasAck, "packet acknowledgement should not exist after send error")
			case recvError:
				// check transfer state after receive error
				// check that the balance for chainA is refunded after error acknowledgement is relayed
				s.Require().Equal(originalBalance.Amount, chainABalance.Amount, "chain A balance should be unchanged after receive error")
				// check that module account escrow address has reverted the locked tokens
				s.Require().Equal(sdk.NewCoin(originalCoin.Denom, sdkmath.ZeroInt()), chainAEscrowBalance, "escrow balance should be reverted after receive error")
				// check that voucher does not exist on chain B
				s.Require().Equal(sdk.NewCoin(chainBDenom.IBCDenom(), sdkmath.ZeroInt()), chainBBalance, "voucher balance should be zero after receive error")

				// check IBC state after receive error
				s.Require().Equal(uint64(2), nextSequenceSend, "next sequence send should be incremented after receive error")
				// packet commitment should be cleared
				s.Require().Nil(packetCommitment, "packet commitment should be cleared after receive error")
				// packet receipt should be written
				s.Require().True(hasReceipt, "packet receipt should exist after receive error")
				// packet acknowledgement should be written
				s.Require().True(hasAck, "packet acknowledgement should exist after receive error")
			case ackError:
				// check transfer state after acknowledgement error
				// check that the balance for chainA is still deducted since acknowledgement failed
				s.Require().Equal(originalBalance.Amount.Sub(totalAmount), chainABalance.Amount, "chain A balance should still be deducted after acknowledgement error")
				// check that module account escrow address has still locked the tokens
				s.Require().Equal(totalCoin, chainAEscrowBalance, "escrow balance should still be locked after acknowledgement error")
				// check that voucher does not exist on chain B since receive returned error acknowledgement
				s.Require().Equal(sdk.NewCoin(chainBDenom.IBCDenom(), totalAmount), chainBBalance, "voucher balance should be zero after acknowledgement error")

				// check IBC state after acknowledgement error
				s.Require().Equal(uint64(2), nextSequenceSend, "next sequence send should be incremented after acknowledgement error")
				// packet commitment should not be cleared
				s.Require().NotNil(packetCommitment, "packet commitment should not be cleared after acknowledgement error")
				// packet receipt should be written
				s.Require().True(hasReceipt, "packet receipt should exist after acknowledgement error")
				// packet acknowledgement should be written
				s.Require().True(hasAck, "packet acknowledgement should exist after acknowledgement error")
			case timeoutError:
				// check transfer state after acknowledgement error
				// check that the balance for chainA is still deducted since acknowledgement failed
				s.Require().Equal(originalBalance.Amount.Sub(totalAmount), chainABalance.Amount, "chain A balance should still be deducted after timeout error")
				// check that module account escrow address has still locked the tokens
				s.Require().Equal(totalCoin, chainAEscrowBalance, "escrow balance should still be locked after timeout error")
				// check that voucher does not exist on chain B since receive returned error acknowledgement
				s.Require().Equal(sdk.NewCoin(chainBDenom.IBCDenom(), sdkmath.ZeroInt()), chainBBalance, "voucher balance should be zero after timeout error")

				// check IBC state after timeout error
				// check IBC state after acknowledgement error
				s.Require().Equal(uint64(2), nextSequenceSend, "next sequence send should be incremented after timeout error")
				// packet commitment should not be cleared
				s.Require().NotNil(packetCommitment, "packet commitment should not be cleared after timeout error")
				// packet receipt should not be written
				s.Require().False(hasReceipt, "packet receipt should not exist after timeout error")
				// packet acknowledgement should not be written
				s.Require().False(hasAck, "packet acknowledgement should not exist after timeout error")
			}
		})
	}
}
