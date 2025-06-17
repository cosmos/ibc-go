package v2_test

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	testclientid  = "testclientid"
	invalidPortID = "invalidportid"
)

type TransferTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain

	pathAToB *ibctesting.Path
	pathBToC *ibctesting.Path
}

func TestTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuite))
}

func (s *TransferTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))

	// setup between chainA and chainB
	// NOTE:
	// pathAToB.EndpointA = endpoint on chainA
	// pathAToB.EndpointB = endpoint on chainB
	s.pathAToB = ibctesting.NewPath(s.chainA, s.chainB)

	// setup between chainB and chainC
	// pathBToC.EndpointA = endpoint on chainB
	// pathBToC.EndpointB = endpoint on chainC
	s.pathBToC = ibctesting.NewPath(s.chainB, s.chainC)

	// setup IBC v2 paths between the chains
	s.pathAToB.SetupV2()
	s.pathBToC.SetupV2()
}

func (s *TransferTestSuite) TestOnSendPacket() {
	var payload channeltypesv2.Payload
	testCases := []struct {
		name                  string
		sourceDenomToTransfer string
		malleate              func()
		expError              error
	}{
		{
			"transfer single denom",
			sdk.DefaultBondDenom,
			func() {},
			nil,
		},
		{
			"transfer with invalid source port",
			sdk.DefaultBondDenom,
			func() {
				payload.SourcePort = invalidPortID
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"transfer with invalid destination port",
			sdk.DefaultBondDenom,
			func() {
				payload.DestinationPort = invalidPortID
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"transfer with invalid source client",
			sdk.DefaultBondDenom,
			func() {
				s.pathAToB.EndpointA.ClientID = testclientid
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"transfer with invalid destination client",
			sdk.DefaultBondDenom,
			func() {
				s.pathAToB.EndpointB.ClientID = testclientid
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"transfer with slashes in base denom",
			"base/coin",
			func() {},
			types.ErrInvalidDenomForTransfer,
		},
		{
			"transfer with slashes in ibc denom",
			fmt.Sprintf("ibc/%x", sha256.Sum256([]byte("coin"))),
			func() {},
			types.ErrInvalidDenomForTransfer,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), tc.sourceDenomToTransfer)

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			s.Require().True(ok)
			originalCoin := sdk.NewCoin(tc.sourceDenomToTransfer, amount)

			token := types.Token{
				Denom:  types.Denom{Base: originalCoin.Denom},
				Amount: originalCoin.Amount.String(),
			}

			transferData := types.NewFungibleTokenPacketData(token.Denom.Path(), token.Amount, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), "")
			bz := s.chainA.Codec.MustMarshal(&transferData)
			payload = channeltypesv2.NewPayload(types.PortID, types.PortID, types.V1, types.EncodingProtobuf, bz)

			// malleate payload
			tc.malleate()

			ctx := s.chainA.GetContext()
			cbs := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			err := cbs.OnSendPacket(ctx, s.pathAToB.EndpointA.ClientID, s.pathAToB.EndpointB.ClientID, 1, payload, s.chainA.SenderAccount.GetAddress())

			if tc.expError != nil {
				s.Require().Contains(err.Error(), tc.expError.Error())
				return
			}

			s.Require().NoError(err)

			escrowAddress := types.GetEscrowAddress(types.PortID, s.pathAToB.EndpointA.ClientID)
			// check that the balance for chainA is updated
			chainABalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			s.Require().Equal(originalBalance.Amount.Sub(amount).Int64(), chainABalance.Amount.Int64())

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			s.Require().Equal(originalCoin, chainAEscrowBalance)
		})
	}
}

func (s *TransferTestSuite) TestOnRecvPacket() {
	var payload channeltypesv2.Payload
	testCases := []struct {
		name                  string
		sourceDenomToTransfer string
		malleate              func()
		expErr                bool
	}{
		{
			"transfer single denom",
			sdk.DefaultBondDenom,
			func() {},
			false,
		},
		{
			"transfer with invalid source port",
			sdk.DefaultBondDenom,
			func() {
				payload.SourcePort = invalidPortID
			},
			true,
		},
		{
			"transfer with invalid dest port",
			sdk.DefaultBondDenom,
			func() {
				payload.DestinationPort = invalidPortID
			},
			true,
		},
		{
			"transfer with invalid source client",
			sdk.DefaultBondDenom,
			func() {
				s.pathAToB.EndpointA.ClientID = testclientid
			},
			true,
		},
		{
			"transfer with invalid destination client",
			sdk.DefaultBondDenom,
			func() {
				s.pathAToB.EndpointB.ClientID = testclientid
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), tc.sourceDenomToTransfer)

			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			s.Require().True(ok)
			originalCoin := sdk.NewCoin(tc.sourceDenomToTransfer, amount)

			msg := types.NewMsgTransferWithEncoding(types.PortID, s.pathAToB.EndpointA.ClientID, originalCoin, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), clienttypes.Height{}, timeoutTimestamp, "", types.EncodingProtobuf, false)
			resp, err := s.chainA.SendMsgs(msg)
			s.Require().NoError(err) // message committed

			packets, err := ibctesting.ParseIBCV2Packets(channeltypes.EventTypeSendPacket, resp.Events)
			s.Require().NoError(err)

			s.Require().Len(packets, 1)
			s.Require().Len(packets[0].Payloads, 1)
			payload = packets[0].Payloads[0]

			ctx := s.chainB.GetContext()
			cbs := s.chainB.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			// malleate payload after it has been sent but before OnRecvPacket callback is called
			tc.malleate()

			recvResult := cbs.OnRecvPacket(ctx, s.pathAToB.EndpointA.ClientID, s.pathAToB.EndpointB.ClientID, packets[0].Sequence, payload, s.chainB.SenderAccount.GetAddress())

			if tc.expErr {
				s.Require().Equal(channeltypesv2.PacketStatus_Failure, recvResult.Status)
				return
			}

			s.Require().Equal(channeltypesv2.PacketStatus_Success, recvResult.Status)
			s.Require().Equal(channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(), recvResult.Acknowledgement)

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
		})
	}
}

func (s *TransferTestSuite) TestOnAckPacket() {
	testCases := []struct {
		name                  string
		sourceDenomToTransfer string
	}{
		{
			"transfer single denom",
			sdk.DefaultBondDenom,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), tc.sourceDenomToTransfer)

			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			s.Require().True(ok)
			originalCoin := sdk.NewCoin(tc.sourceDenomToTransfer, amount)

			msg := types.NewMsgTransferWithEncoding(types.PortID, s.pathAToB.EndpointA.ClientID, originalCoin, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), clienttypes.Height{}, timeoutTimestamp, "", types.EncodingProtobuf, false)

			resp, err := s.chainA.SendMsgs(msg)
			s.Require().NoError(err) // message committed
			packets, err := ibctesting.ParseIBCV2Packets(channeltypes.EventTypeSendPacket, resp.Events)
			s.Require().NoError(err)

			s.Require().Len(packets, 1)
			s.Require().Len(packets[0].Payloads, 1)
			payload := packets[0].Payloads[0]

			ctx := s.chainA.GetContext()
			cbs := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

			err = cbs.OnAcknowledgementPacket(
				ctx, s.pathAToB.EndpointA.ClientID, s.pathAToB.EndpointB.ClientID,
				packets[0].Sequence, ack.Acknowledgement(), payload, s.chainA.SenderAccount.GetAddress(),
			)
			s.Require().NoError(err)

			// on successful ack, the tokens sent in packets should still be in escrow
			escrowAddress := types.GetEscrowAddress(types.PortID, s.pathAToB.EndpointA.ClientID)
			// check that the balance for chainA is updated
			chainABalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			s.Require().Equal(originalBalance.Amount.Sub(amount).Int64(), chainABalance.Amount.Int64())

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			s.Require().Equal(originalCoin, chainAEscrowBalance)

			// create a custom error ack and replay the callback to ensure it fails with IBC v2 callbacks
			errAck := channeltypes.NewErrorAcknowledgement(types.ErrInvalidAmount)
			err = cbs.OnAcknowledgementPacket(
				ctx, s.pathAToB.EndpointA.ClientID, s.pathAToB.EndpointB.ClientID,
				1, errAck.Acknowledgement(), payload, s.chainA.SenderAccount.GetAddress(),
			)
			s.Require().Error(err)

			// create the sentinel error ack and replay the callback to ensure the tokens are correctly refunded
			// we can replay the callback here because the replay protection is handled in the IBC handler
			err = cbs.OnAcknowledgementPacket(
				ctx, s.pathAToB.EndpointA.ClientID, s.pathAToB.EndpointB.ClientID,
				1, channeltypesv2.ErrorAcknowledgement[:], payload, s.chainA.SenderAccount.GetAddress(),
			)
			s.Require().NoError(err)

			// on error ack, the tokens sent in packets should be returned to sender
			// check that the balance for chainA is refunded
			chainABalance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			s.Require().Equal(originalBalance.Amount, chainABalance.Amount)

			// check that module account escrow address has no tokens
			chainAEscrowBalance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			s.Require().Equal(sdk.NewCoin(originalCoin.Denom, sdkmath.ZeroInt()), chainAEscrowBalance)
		})
	}
}

func (s *TransferTestSuite) TestOnTimeoutPacket() {
	testCases := []struct {
		name                  string
		sourceDenomToTransfer string
	}{
		{
			"transfer single denom",
			sdk.DefaultBondDenom,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			originalBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), tc.sourceDenomToTransfer)

			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
			s.Require().True(ok)
			originalCoin := sdk.NewCoin(tc.sourceDenomToTransfer, amount)

			msg := types.NewMsgTransferWithEncoding(types.PortID, s.pathAToB.EndpointA.ClientID, originalCoin, s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), clienttypes.Height{}, timeoutTimestamp, "", types.EncodingProtobuf, false)
			resp, err := s.chainA.SendMsgs(msg)
			s.Require().NoError(err) // message committed

			packets, err := ibctesting.ParseIBCV2Packets(channeltypes.EventTypeSendPacket, resp.Events)
			s.Require().NoError(err)

			s.Require().Len(packets, 1)
			s.Require().Len(packets[0].Payloads, 1)
			payload := packets[0].Payloads[0]

			// on successful send, the tokens sent in packets should be in escrow
			escrowAddress := types.GetEscrowAddress(types.PortID, s.pathAToB.EndpointA.ClientID)
			// check that the balance for chainA is updated
			chainABalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			s.Require().Equal(originalBalance.Amount.Sub(amount).Int64(), chainABalance.Amount.Int64())

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			s.Require().Equal(originalCoin, chainAEscrowBalance)

			ctx := s.chainA.GetContext()
			cbs := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			err = cbs.OnTimeoutPacket(ctx, s.pathAToB.EndpointA.ClientID, s.pathAToB.EndpointB.ClientID, packets[0].Sequence, payload, s.chainA.SenderAccount.GetAddress())
			s.Require().NoError(err)

			// on timeout, the tokens sent in packets should be returned to sender
			// check that the balance for chainA is refunded
			chainABalance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), originalCoin.Denom)
			s.Require().Equal(originalBalance.Amount, chainABalance.Amount)

			// check that module account escrow address has no tokens
			chainAEscrowBalance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddress, originalCoin.Denom)
			s.Require().Equal(sdk.NewCoin(originalCoin.Denom, sdkmath.ZeroInt()), chainAEscrowBalance)
		})
	}
}
