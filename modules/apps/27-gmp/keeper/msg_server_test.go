package keeper_test

import (
	"errors"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

var errAny = errors.New("any error")

func (s *KeeperTestSuite) TestSendCall() {
	var msg *types.MsgSendCall

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: empty encoding defaults to ABI",
			func() {
				msg.Encoding = ""
			},
			nil,
		},
		{
			"success: protobuf encoding",
			func() {
				msg.Encoding = types.EncodingProtobuf
			},
			nil,
		},
		{
			"success: JSON encoding",
			func() {
				msg.Encoding = types.EncodingJSON
			},
			nil,
		},
		{
			"success: ABI encoding",
			func() {
				msg.Encoding = types.EncodingABI
			},
			nil,
		},
		{
			"failure: invalid sender address",
			func() {
				msg.Sender = invalid
			},
			errAny,
		},
		{
			"failure: empty sender address",
			func() {
				msg.Sender = ""
			},
			errAny,
		},
		{
			"failure: invalid encoding",
			func() {
				msg.Encoding = "invalid-encoding"
			},
			types.ErrInvalidEncoding,
		},
		{
			"failure: invalid source client - counterparty not found",
			func() {
				msg.SourceClient = invalid
			},
			errAny,
		},
		{
			"failure: payload too long",
			func() {
				msg.Payload = make([]byte, types.MaximumPayloadLength+1)
			},
			types.ErrInvalidPayload,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path := ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			sender := s.chainA.SenderAccount.GetAddress()
			recipient := s.chainB.SenderAccount.GetAddress()
			payload := s.serializeMsgs(&banktypes.MsgSend{
				FromAddress: sender.String(),
				ToAddress:   recipient.String(),
				Amount:      sdk.NewCoins(ibctesting.TestCoin),
			})

			msg = types.NewMsgSendCall(
				path.EndpointA.ClientID,
				sender.String(),
				"",
				payload,
				[]byte(testSalt),
				uint64(s.chainA.GetContext().BlockTime().Add(time.Hour).Unix()),
				types.EncodingProtobuf,
				"",
			)

			tc.malleate()

			resp, err := s.chainA.GetSimApp().GMPKeeper.SendCall(s.chainA.GetContext(), msg)

			switch {
			case tc.expErr == nil:
				s.Require().NoError(err)
				s.Require().NotNil(resp)
				s.Require().Equal(uint64(1), resp.Sequence)

			case errors.Is(tc.expErr, errAny):
				s.Require().Error(err)
				s.Require().Nil(resp)

			default:
				s.Require().ErrorIs(err, tc.expErr)
				s.Require().Nil(resp)
			}
		})
	}
}
