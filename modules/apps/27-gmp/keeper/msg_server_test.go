package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestSendCall() {
	var msg *types.MsgSendCall

	testCases := []struct {
		name        string
		malleate    func()
		expEncoding string
	}{
		{
			"success: empty encoding defaults to ABI",
			func() {
				msg.Encoding = ""
			},
			types.EncodingABI,
		},
		{
			"success: protobuf encoding",
			func() {
				msg.Encoding = types.EncodingProtobuf
			},
			types.EncodingProtobuf,
		},
		{
			"success: JSON encoding",
			func() {
				msg.Encoding = types.EncodingJSON
			},
			types.EncodingJSON,
		},
		{
			"success: ABI encoding",
			func() {
				msg.Encoding = types.EncodingABI
			},
			types.EncodingABI,
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

			s.Require().NoError(err)
			s.Require().NotNil(resp)
			s.Require().Equal(uint64(1), resp.Sequence)
		})
	}
}
