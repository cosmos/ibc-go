package v2_test

import (
	"time"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
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
