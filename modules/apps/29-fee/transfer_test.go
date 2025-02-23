package fee_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// Integration test to ensure ics29 works with ics20
func (suite *FeeTestSuite) TestFeeTransfer() {
	testCases := []struct {
		name           string
		coinToTransfer sdk.Coin
	}{
		{
			"transfer single denom",
			ibctesting.TestCoin,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			feeTransferVersion := string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: transfertypes.V1}))
			path.EndpointA.ChannelConfig.Version = feeTransferVersion
			path.EndpointB.ChannelConfig.Version = feeTransferVersion
			path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
			path.EndpointB.ChannelConfig.PortID = transfertypes.PortID

			path.Setup()

			fee := types.Fee{
				RecvFee:    defaultRecvFee,
				AckFee:     defaultAckFee,
				TimeoutFee: defaultTimeoutFee,
			}

			msgs := []sdk.Msg{
				types.NewMsgPayPacketFee(fee, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, suite.chainA.SenderAccount.GetAddress().String(), nil),
				transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, tc.coinToTransfer, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), clienttypes.NewHeight(1, 100), 0, ""),
			}

			res, err := suite.chainA.SendMsgs(msgs...)
			suite.Require().NoError(err) // message committed

			// after incentivizing the packets
			originalChainASenderAccountBalance := sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))

			packet, err := ibctesting.ParsePacketFromEvents(res.Events)
			suite.Require().NoError(err)

			// register counterparty address on chainB
			payeeAddr, err := sdk.AccAddressFromBech32(ibctesting.TestAccAddress)
			suite.Require().NoError(err)

			msgRegister := types.NewMsgRegisterCounterpartyPayee(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, suite.chainB.SenderAccount.GetAddress().String(), payeeAddr.String())
			_, err = suite.chainB.SendMsgs(msgRegister)
			suite.Require().NoError(err) // message committed

			// relay packet
			err = path.RelayPacket(packet)
			suite.Require().NoError(err) // relay committed

			// ensure relayers got paid
			// relayer for forward relay: chainB.SenderAccount
			// relayer for reverse relay: chainA.SenderAccount

			// check forward relay balance
			suite.Require().Equal(
				fee.RecvFee,
				sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), payeeAddr, ibctesting.TestCoin.Denom)),
			)

			suite.Require().Equal(
				fee.AckFee, // ack fee paid, no refund needed since timeout_fee = recv_fee + ack_fee
				sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom)).Sub(originalChainASenderAccountBalance[0]))
		})
	}
}

func (suite *FeeTestSuite) TestOnesidedFeeMiddlewareTransferHandshake() {
	RemoveFeeMiddleware(suite.chainB) // remove fee middleware from chainB

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	feeTransferVersion := string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: transfertypes.V1}))
	path.EndpointA.ChannelConfig.Version = feeTransferVersion // this will be renegotiated by the Try step
	path.EndpointB.ChannelConfig.Version = ""                 // this will be overwritten by the Try step
	path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.PortID

	path.Setup()

	suite.Require().Equal(path.EndpointA.ChannelConfig.Version, transfertypes.V1)
	suite.Require().Equal(path.EndpointB.ChannelConfig.Version, transfertypes.V1)
}
