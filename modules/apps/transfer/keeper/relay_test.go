package keeper_test

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	transferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

var (
	zeroAmount    = sdkmath.NewInt(0)
	defaultAmount = ibctesting.DefaultCoinAmount
)

// TestSendTransfer tests sending from chainA to chainB using both coin
// that originate on chainA and coin that originate on chainB.
func (suite *KeeperTestSuite) TestSendTransfer() {
	var (
		coin            sdk.Coin
		path            *ibctesting.Path
		sender          sdk.AccAddress
		memo            string
		expEscrowAmount sdkmath.Int // total amounts in escrow for denom on receiving chain
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"successful transfer of native token",
			func() {},
			nil,
		},
		{
			"successful transfer of native token with memo",
			func() {
				memo = "memo" //nolint:goconst
			},
			nil,
		},
		{
			"successful transfer of IBC token",
			func() {
				// send IBC token back to chainB
				denom := types.NewDenom(ibctesting.TestCoin.Denom, types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin = sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount)

				expEscrowAmount = zeroAmount
			},
			nil,
		},
		{
			"successful transfer of IBC token with memo",
			func() {
				// send IBC token back to chainB
				denom := types.NewDenom(ibctesting.TestCoin.Denom, types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin = sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount)
				memo = "memo"

				expEscrowAmount = zeroAmount
			},
			nil,
		},
		{
			"successful transfer of entire balance",
			func() {
				coin = sdk.NewCoin(coin.Denom, types.UnboundedSpendLimit())
				var ok bool
				expEscrowAmount, ok = sdkmath.NewIntFromString(ibctesting.DefaultGenesisAccBalance)
				suite.Require().True(ok)
			},
			nil,
		},
		{
			"successful transfer of entire spendable balance with vesting account",
			func() {
				// create vesting account
				vestingAccPrivKey := secp256k1.GenPrivKey()
				vestingAccAddress := sdk.AccAddress(vestingAccPrivKey.PubKey().Address())

				vestingCoins := sdk.NewCoins(sdk.NewCoin(coin.Denom, ibctesting.DefaultCoinAmount))
				_, err := suite.chainA.SendMsgs(vestingtypes.NewMsgCreateVestingAccount(
					suite.chainA.SenderAccount.GetAddress(),
					vestingAccAddress,
					vestingCoins,
					suite.chainA.GetContext().BlockTime().Add(time.Hour).Unix(),
					false,
				))
				suite.Require().NoError(err)
				sender = vestingAccAddress

				// transfer some spendable coins to vesting account
				transferCoin := sdk.NewCoin(coin.Denom, sdkmath.NewInt(42))
				_, err = suite.chainA.SendMsgs(banktypes.NewMsgSend(suite.chainA.SenderAccount.GetAddress(), vestingAccAddress, sdk.NewCoins(transferCoin)))
				suite.Require().NoError(err)

				coin = sdk.NewCoin(coin.Denom, types.UnboundedSpendLimit())
				expEscrowAmount = transferCoin.Amount
			},
			nil,
		},
		{
			"failure: no spendable coins for vesting account",
			func() {
				// create vesting account
				vestingAccPrivKey := secp256k1.GenPrivKey()
				vestingAccAddress := sdk.AccAddress(vestingAccPrivKey.PubKey().Address())

				vestingCoin := sdk.NewCoin(coin.Denom, ibctesting.DefaultCoinAmount)
				_, err := suite.chainA.SendMsgs(vestingtypes.NewMsgCreateVestingAccount(
					suite.chainA.SenderAccount.GetAddress(),
					vestingAccAddress,
					sdk.NewCoins(vestingCoin),
					suite.chainA.GetContext().BlockTime().Add(time.Hour).Unix(),
					false,
				))
				suite.Require().NoError(err)
				sender = vestingAccAddress

				// just to prove that the vesting account has a balance (but not spendable)
				vestingAccBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), vestingAccAddress, coin.Denom)
				suite.Require().Equal(vestingCoin.Amount.Int64(), vestingAccBalance.Amount.Int64())
				vestinSpendableBalance := suite.chainA.GetSimApp().BankKeeper.SpendableCoins(suite.chainA.GetContext(), vestingAccAddress)
				suite.Require().Zero(vestinSpendableBalance.AmountOf(coin.Denom).Int64())

				coin = sdk.NewCoin(coin.Denom, types.UnboundedSpendLimit())
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: sender account is blocked",
			func() {
				sender = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(minttypes.ModuleName)
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: bank send from sender account failed, insufficient balance",
			func() {
				coin = sdk.NewCoin("randomdenom", defaultAmount)
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: denom trace not found",
			func() {
				denom := types.NewDenom("randomdenom", types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin = sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount)
			},
			types.ErrDenomNotFound,
		},
		{
			"failure: bank send from module account failed, insufficient balance",
			func() {
				denom := types.NewDenom(ibctesting.TestCoin.Denom, types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin = sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount.Add(sdkmath.NewInt(1)))
			},
			sdkerrors.ErrInsufficientFunds,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			// create IBC token on chainA
			transferMsg := types.NewMsgTransfer(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, ibctesting.TestCoin, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainA.GetTimeoutHeight(), 0, "")

			result, err := suite.chainB.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			packet, err := ibctesting.ParsePacketFromEvents(result.Events)
			suite.Require().NoError(err)

			err = path.RelayPacket(packet)
			suite.Require().NoError(err)

			// Value that can malleated for Transfer we are testing.
			coin = ibctesting.TestCoin
			sender = suite.chainA.SenderAccount.GetAddress()
			memo = ""
			expEscrowAmount = defaultAmount

			tc.malleate()

			msg := types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				coin,
				sender.String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				suite.chainB.GetTimeoutHeight(), 0, // only use timeout height
				memo,
			)

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(suite.chainA.GetContext(), msg)

			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Nil(res)
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)

				// We do not expect escrowed amounts in error cases.
				expEscrowAmount = zeroAmount
			}
			// Assert amounts escrowed are as expected.
			suite.assertEscrowEqual(suite.chainA, coin, expEscrowAmount)
		})
	}
}

func (suite *KeeperTestSuite) TestSendTransferSetsTotalEscrowAmountForSourceIBCToken() {
	/*
		Given the following flow of tokens:

		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake
		                                  ^
		                                  |
		                             SendTransfer

		This test will transfer vouchers of denom "transfer/channel-0/stake" from chain B
		to chain A over channel-1 to assert that total escrow amount is stored on chain B
		for vouchers of denom "transfer/channel-0/stake" because chain B acts as source

		Set up:
		- Two transfer channels between chain A and chain B (channel-0 and channel-1).
		- Tokens of native denom "stake" on chain A transferred to chain B over channel-0
		and vouchers minted with denom trace "transfer/channel-0/stake".

		Execute:
		- Transfer vouchers of denom trace "transfer/channel-0/stake" from chain B to chain A
		over channel-1.

		Assert:
		- The vouchers are not of a native denom (because they are of an IBC denom), but chain B
		is the source, then the value for total escrow amount should still be stored for the IBC
		denom that corresponds to the trace "transfer/channel-0/stake".
	*/

	// set up
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// create IBC token on chain B with denom trace "transfer/channel-0/stake"
	coin := ibctesting.TestCoin
	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		coin,
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainB.GetTimeoutHeight(), 0, "",
	)
	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)

	err = path1.RelayPacket(packet)
	suite.Require().NoError(err)

	// execute
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))
	coin = sdk.NewCoin(denom.IBCDenom(), defaultAmount)
	msg := types.NewMsgTransfer(
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		coin,
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(), 0, "",
	)

	res, err := suite.chainB.GetSimApp().TransferKeeper.Transfer(suite.chainB.GetContext(), msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// check total amount in escrow of sent token on sending chain
	totalEscrow := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(defaultAmount, totalEscrow.Amount)
}

// TestOnRecvPacket_ReceiverIsNotSource tests receiving on chainB a coin that
// originates on chainA. The bulk of the testing occurs  in the test case for
// loop since setup is intensive for all cases. The malleate function allows
// for testing invalid cases.
func (suite *KeeperTestSuite) TestOnRecvPacket_ReceiverIsNotSource() {
	var packetData types.InternalTransferRepresentation

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"successful receive",
			func() {},
			nil,
		},
		{
			"successful receive with memo",
			func() {
				packetData.Memo = "memo"
			},
			nil,
		},
		{
			"failure: mint zero coin",
			func() {
				packetData.Token.Amount = zeroAmount.String()
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: receiver is module account",
			func() {
				packetData.Receiver = suite.chainB.GetSimApp().AccountKeeper.GetModuleAddress(minttypes.ModuleName).String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: receiver is invalid",
			func() {
				packetData.Receiver = "invalid-address"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: receive is disabled",
			func() {
				suite.chainB.GetSimApp().TransferKeeper.SetParams(suite.chainB.GetContext(),
					types.Params{
						ReceiveEnabled: false,
					})
			},
			types.ErrReceiveDisabled,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			receiver := suite.chainB.SenderAccount.GetAddress().String() // must be explicitly changed in malleate

			// send coins from chainA to chainB
			transferMsg := types.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.TestCoin, suite.chainA.SenderAccount.GetAddress().String(), receiver, clienttypes.NewHeight(1, 110), 0, "")
			_, err := suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			token := types.Token{Denom: types.NewDenom(transferMsg.Token.Denom), Amount: transferMsg.Token.Amount.String()}
			packetData = types.NewInternalTransferRepresentation(token, suite.chainA.SenderAccount.GetAddress().String(), receiver, "")
			sourcePort := path.EndpointA.ChannelConfig.PortID
			sourceChannel := path.EndpointA.ChannelID
			destinationPort := path.EndpointB.ChannelConfig.PortID
			destinationChannel := path.EndpointB.ChannelID

			tc.malleate()

			denom := types.NewDenom(token.Denom.Base, types.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))

			err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(
				suite.chainB.GetContext(),
				packetData,
				sourcePort,
				sourceChannel,
				destinationPort,
				destinationChannel,
			)

			if tc.expError == nil {
				suite.Require().NoError(err)

				// Check denom metadata for of tokens received on chain B.
				actualMetadata, found := suite.chainB.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainB.GetContext(), denom.IBCDenom())

				suite.Require().True(found)
				suite.Require().Equal(metadataFromDenom(denom), actualMetadata)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)

				// Check denom metadata absence for cases where recv fails.
				_, found := suite.chainB.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainB.GetContext(), denom.IBCDenom())

				suite.Require().False(found)
			}
		})
	}
}

// TestOnRecvPacket_ReceiverIsSource tests receiving on chainB a coin that
// originated on chainB, but was previously transferred to chainA. The bulk
// of the testing occurs in the test case for loop since setup is intensive
// for all cases. The malleate function allows for testing invalid cases.
func (suite *KeeperTestSuite) TestOnRecvPacket_ReceiverIsSource() {
	var (
		packetData      types.InternalTransferRepresentation
		expEscrowAmount sdkmath.Int // total amount in escrow for denom on receiving chain
	)

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"successful receive",
			func() {},
			nil,
		},
		{
			"successful receive with memo",
			func() {
				packetData.Memo = "memo"
			},
			nil,
		},
		{
			"successful receive of half the amount",
			func() {
				packetData.Token.Amount = sdkmath.NewInt(50).String()
				// expect 50 remaining
				expEscrowAmount = sdkmath.NewInt(50)
			},
			nil,
		},
		{
			"failure: empty coin",
			func() {
				packetData.Token.Amount = zeroAmount.String()
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: tries to unescrow more tokens than allowed",
			func() {
				packetData.Token.Amount = sdkmath.NewInt(1000000).String()
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: empty denom",
			func() {
				packetData.Token.Denom = types.Denom{}
			},
			types.ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid receiver address",
			func() {
				packetData.Receiver = "gaia1scqhwpgsmr6vmztaa7suurfl52my6nd2kmrudl"
			},
			errors.New("failed to decode receiver address"),
		},
		{
			"failure: receiver is module account",
			func() {
				packetData.Receiver = suite.chainB.GetSimApp().AccountKeeper.GetModuleAddress(minttypes.ModuleName).String()
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			receiver := suite.chainB.SenderAccount.GetAddress().String() // must be explicitly changed in malleate
			expEscrowAmount = zeroAmount                                 // total amount in escrow of voucher denom on receiving chain

			// send coins from chainA to chainB, receive them, acknowledge them
			transferMsg := types.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.TestCoin, suite.chainA.SenderAccount.GetAddress().String(), receiver, clienttypes.NewHeight(1, 110), 0, "")
			_, err := suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			token := types.Token{Denom: types.NewDenom(transferMsg.Token.Denom, types.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)), Amount: transferMsg.Token.Amount.String()}
			packetData = types.NewInternalTransferRepresentation(token, suite.chainA.SenderAccount.GetAddress().String(), receiver, "")
			sourcePort := path.EndpointB.ChannelConfig.PortID
			sourceChannel := path.EndpointB.ChannelID
			destinationPort := path.EndpointA.ChannelConfig.PortID
			destinationChannel := path.EndpointA.ChannelID

			tc.malleate()

			err = suite.chainA.GetSimApp().TransferKeeper.OnRecvPacket(
				suite.chainA.GetContext(),
				packetData,
				sourcePort,
				sourceChannel,
				destinationPort,
				destinationChannel,
			)

			if tc.expError == nil {
				suite.Require().NoError(err)

				_, found := suite.chainA.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainA.GetContext(), sdk.DefaultBondDenom)
				suite.Require().False(found)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.expError.Error())

				// Expect escrowed amount to stay same on failure.
				expEscrowAmount = defaultAmount
			}

			// Assert amounts escrowed are as expected, we do not malleate amount escrowed in initial transfer.
			suite.assertEscrowEqual(suite.chainA, ibctesting.TestCoin, expEscrowAmount)
		})
	}
}

func (suite *KeeperTestSuite) TestOnRecvPacketSetsTotalEscrowAmountForSourceIBCToken() {
	/*
		Given the following flow of tokens:

		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A (channel-1)             -> (channel-1) chain B
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake    transfer/channel-0/stake
		                                                                                                                   ^
		                                                                                                                   |
		                                                                                                              OnRecvPacket

		This test will assert that on receiving vouchers of denom "transfer/channel-0/stake"
		on chain B the total escrow amount is updated on because chain B acted as source
		when vouchers were transferred to chain A over channel-1.

		Setup:
		- Two transfer channels between chain A and chain B.
		- Vouchers of denom trace "transfer/channel-0/stake" on chain B are in escrow
		account for port ID transfer and channel ID channel-1.

		Execute:
		- Receive vouchers of denom trace "transfer/channel-0/stake" from chain A to chain B
		over channel-1.

		Assert:
		- The vouchers are not of a native denom (because they are of an IBC denom), but chain B
		is the source, then the value for total escrow amount should still be updated for the IBC
		denom that corresponds to the trace "transfer/channel-0/stake" when the vouchers are
		received back on chain B.
	*/

	amount := defaultAmount

	// setup
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// denom path: {transfer/channel-1/transfer/channel-0}
	denom := types.NewDenom(
		sdk.DefaultBondDenom,
		types.NewHop(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID),
		types.NewHop(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID),
	)

	data := types.NewInternalTransferRepresentation(
		types.Token{
			Denom:  denom,
			Amount: amount.String(),
		}, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "")
	sourcePort := path2.EndpointA.ChannelConfig.PortID
	sourceChannel := path2.EndpointA.ChannelID
	destinationPort := path2.EndpointB.ChannelConfig.PortID
	destinationChannel := path2.EndpointB.ChannelID

	// fund escrow account for transfer and channel-1 on chain B
	// denom path: transfer/channel-0
	denom = types.NewDenom(
		sdk.DefaultBondDenom,
		types.NewHop(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID),
	)

	escrowAddress := types.GetEscrowAddress(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID)
	coin := sdk.NewCoin(denom.IBCDenom(), amount)
	suite.Require().NoError(
		banktestutil.FundAccount(
			suite.chainB.GetContext(),
			suite.chainB.GetSimApp().BankKeeper,
			escrowAddress,
			sdk.NewCoins(coin),
		),
	)

	suite.chainB.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainB.GetContext(), coin)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(defaultAmount, totalEscrowChainB.Amount)

	// execute onRecvPacket, when chaninB receives the source token the escrow amount should decrease
	err := suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(
		suite.chainB.GetContext(),
		data,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
	)
	suite.Require().NoError(err)

	// check total amount in escrow of sent token on receiving chain
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(zeroAmount, totalEscrowChainB.Amount)
}

// TestOnAcknowledgementPacket tests that successful acknowledgement is a no-op
// and failure acknowledment leads to refund when attempting to send from chainA
// to chainB. If sender is source then the denomination being refunded has no
// trace.
func (suite *KeeperTestSuite) TestOnAcknowledgementPacket() {
	var (
		successAck      = channeltypes.NewResultAcknowledgement([]byte{byte(1)})
		failedAck       = channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed packet transfer"))
		denom           types.Denom
		amount          sdkmath.Int
		path            *ibctesting.Path
		expEscrowAmount sdkmath.Int
	)

	testCases := []struct {
		msg      string
		ack      channeltypes.Acknowledgement
		malleate func()
		expError error
	}{
		{
			"success ack: no-op",
			successAck,
			func() {
				denom = types.NewDenom(sdk.DefaultBondDenom, types.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
			},
			nil,
		},
		{
			"failed ack: successful refund of native coin",
			failedAck,
			func() {
				escrow := types.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				denom = types.NewDenom(sdk.DefaultBondDenom)
				coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)

				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrow, sdk.NewCoins(coin)))

				// set escrow amount that would have been stored after successful execution of MsgTransfer
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, amount))
			},
			nil,
		},
		{
			"failed ack: successful refund of IBC voucher",
			failedAck,
			func() {
				escrow := types.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				denom = types.NewDenom(sdk.DefaultBondDenom, types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin := sdk.NewCoin(denom.IBCDenom(), amount)

				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrow, sdk.NewCoins(coin)))
			},
			nil,
		},
		{
			"failed ack: funds cannot be refunded because escrow account has zero balance",
			failedAck,
			func() {
				denom = types.NewDenom(sdk.DefaultBondDenom)

				// set escrow amount that would have been stored after successful execution of MsgTransfer
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, amount))
				expEscrowAmount = defaultAmount
			},
			sdkerrors.ErrInsufficientFunds,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			amount = defaultAmount // must be explicitly changed
			expEscrowAmount = zeroAmount

			tc.malleate()

			data := types.NewInternalTransferRepresentation(
				types.Token{
					Denom:  denom,
					Amount: amount.String(),
				}, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "")
			sourcePort := path.EndpointA.ChannelConfig.PortID
			sourceChannel := path.EndpointA.ChannelID
			preAcknowledgementBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom.IBCDenom())

			err := suite.chainA.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainA.GetContext(), sourcePort, sourceChannel, data, tc.ack)

			// check total amount in escrow of sent token denom on sending chain
			totalEscrow := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), denom.IBCDenom())
			suite.Require().Equal(expEscrowAmount, totalEscrow.Amount)

			if tc.expError == nil {
				suite.Require().NoError(err)
				postAcknowledgementBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom.IBCDenom())
				deltaAmount := postAcknowledgementBalance.Amount.Sub(preAcknowledgementBalance.Amount)

				if tc.ack.Success() {
					suite.Require().Equal(int64(0), deltaAmount.Int64(), "successful ack changed balance")
				} else {
					suite.Require().Equal(amount, deltaAmount, "failed ack did not trigger refund")
				}
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnAcknowledgementPacketSetsTotalEscrowAmountForSourceIBCToken() {
	/*
		This test is testing the following scenario. Given tokens travelling like this:

		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A (channel-1)
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake
		                                 ^
		                                 |
		                         OnAcknowledgePacket

		We want to assert that on failed acknowledgment of vouchers sent with denom trace
		"transfer/channel-0/stake" on chain B the total escrow amount is updated.

		Set up:
		- Two transfer channels between chain A and chain B.
		- Vouckers of denom "transfer/channel-0/stake" on chain B are in escrow
		account for port ID transfer and channel ID channel-1.

		Execute:
		- Acknowledge vouchers of denom trace "transfer/channel-0/stake" sent from chain B
		to chain B over channel-1.

		Assert:
		- The vouchers are not of a native denom (because they are of an IBC denom), but chain B
		is the source, then the value for total escrow amount should still be updated for the IBC
		denom that corresponds to the trace "transfer/channel-0/stake" when processing the failed
		acknowledgement.
	*/

	amount := defaultAmount
	ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed packet transfer"))

	// set up
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// fund escrow account for transfer and channel-1 on chain B
	// denom path: transfer/channel-0
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	escrowAddress := types.GetEscrowAddress(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID)
	coin := sdk.NewCoin(denom.IBCDenom(), amount)
	suite.Require().NoError(
		banktestutil.FundAccount(
			suite.chainB.GetContext(),
			suite.chainB.GetSimApp().BankKeeper,
			escrowAddress,
			sdk.NewCoins(coin),
		),
	)

	data := types.NewInternalTransferRepresentation(
		types.Token{
			Denom:  denom,
			Amount: amount.String(),
		},
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainA.SenderAccount.GetAddress().String(),
		"",
	)
	sourcePort := path2.EndpointB.ChannelConfig.PortID
	sourceChannel := path2.EndpointB.ChannelID

	suite.chainB.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainB.GetContext(), coin)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(defaultAmount, totalEscrowChainB.Amount)

	err := suite.chainB.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainB.GetContext(), sourcePort, sourceChannel, data, ack)
	suite.Require().NoError(err)

	// check total amount in escrow of sent token on sending chain
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(zeroAmount, totalEscrowChainB.Amount)
}

// TestOnTimeoutPacket tests private refundPacket function since it is a simple
// wrapper over it. The actual timeout does not matter since IBC core logic
// is not being tested. The test is timing out a send from chainA to chainB
// so the refunds are occurring on chainA.
func (suite *KeeperTestSuite) TestOnTimeoutPacket() {
	var (
		path            *ibctesting.Path
		amount          string
		sender          string
		denom           types.Denom
		expEscrowAmount sdkmath.Int
	)

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"successful timeout: sender is source of coin",
			func() {
				escrow := types.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				denom = types.NewDenom(sdk.DefaultBondDenom)
				coinAmount, ok := sdkmath.NewIntFromString(amount)
				suite.Require().True(ok)
				coin := sdk.NewCoin(denom.IBCDenom(), coinAmount)
				expEscrowAmount = zeroAmount

				// funds the escrow account to have balance
				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrow, sdk.NewCoins(coin)))
				// set escrow amount that would have been stored after successful execution of MsgTransfer
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), coin)
			},
			nil,
		},
		{
			"successful timeout: sender is not source of coin",
			func() {
				escrow := types.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				denom = types.NewDenom(sdk.DefaultBondDenom, types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coinAmount, ok := sdkmath.NewIntFromString(amount)
				suite.Require().True(ok)
				coin := sdk.NewCoin(denom.IBCDenom(), coinAmount)
				expEscrowAmount = zeroAmount

				// funds the escrow account to have balance
				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrow, sdk.NewCoins(coin)))
			},
			nil,
		},
		{
			"failure: funds cannot be refunded because escrow account has no balance for non-native coin",
			func() {
				denom = types.NewDenom("bitcoin")
				var ok bool
				expEscrowAmount, ok = sdkmath.NewIntFromString(amount)
				suite.Require().True(ok)

				// set escrow amount that would have been stored after successful execution of MsgTransfer
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(denom.IBCDenom(), expEscrowAmount))
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: funds cannot be refunded because escrow account has no balance for native coin",
			func() {
				denom = types.NewDenom(sdk.DefaultBondDenom)
				var ok bool
				expEscrowAmount, ok = sdkmath.NewIntFromString(amount)
				suite.Require().True(ok)

				// set escrow amount that would have been stored after successful execution of MsgTransfer
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(denom.IBCDenom(), expEscrowAmount))
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: cannot mint because sender address is invalid",
			func() {
				denom = types.NewDenom(sdk.DefaultBondDenom)
				amount = sdkmath.OneInt().String()
				sender = "invalid address"
			},
			errors.New("decoding bech32 failed"),
		},
		{
			"failure: invalid amount",
			func() {
				denom = types.NewDenom(sdk.DefaultBondDenom)
				amount = "invalid"
			},
			types.ErrInvalidAmount,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			amount = defaultAmount.String() // must be explicitly changed
			sender = suite.chainA.SenderAccount.GetAddress().String()
			expEscrowAmount = zeroAmount

			tc.malleate()

			data := types.NewInternalTransferRepresentation(
				types.Token{
					Denom:  denom,
					Amount: amount,
				}, sender, suite.chainB.SenderAccount.GetAddress().String(), "")
			sourcePort := path.EndpointA.ChannelConfig.PortID
			sourceChannel := path.EndpointA.ChannelID
			preTimeoutBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom.IBCDenom())

			err := suite.chainA.GetSimApp().TransferKeeper.OnTimeoutPacket(suite.chainA.GetContext(), sourcePort, sourceChannel, data)

			postTimeoutBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom.IBCDenom())
			deltaAmount := postTimeoutBalance.Amount.Sub(preTimeoutBalance.Amount)

			// check total amount in escrow of sent token denom on sending chain
			totalEscrow := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), denom.IBCDenom())
			suite.Require().Equal(expEscrowAmount, totalEscrow.Amount)

			if tc.expError == nil {
				suite.Require().NoError(err)
				amountParsed, ok := sdkmath.NewIntFromString(amount)
				suite.Require().True(ok)
				suite.Require().Equal(amountParsed, deltaAmount, "successful timeout did not trigger refund")
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.expError.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnTimeoutPacketSetsTotalEscrowAmountForSourceIBCToken() {
	/*
		Given the following flow of tokens:

		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A (channel-1)
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake
		                                 ^
		                                 |
		                           OnTimeoutPacket

		We want to assert that on timeout of vouchers sent with denom trace
		"transfer/channel-0/stake" on chain B the total escrow amount is updated.

		Set up:
		- Two transfer channels between chain A and chain B.
		- Vouckers of denom "transfer/channel-0/stake" on chain B are in escrow
		account for port ID transfer and channel ID channel-1.

		Execute:
		- Timeout vouchers of denom trace "transfer/channel-0/stake" sent from chain B
		to chain B over channel-1.

		Assert:
		- The vouchers are not of a native denom (because they are of an IBC denom), but chain B
		is the source, then the value for total escrow amount should still be updated for the IBC
		denom that corresponds to the trace "transfer/channel-0/stake" when processing the timeout.
	*/

	amount := defaultAmount

	// set up
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// fund escrow account for transfer and channel-1 on chain B
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	escrowAddress := types.GetEscrowAddress(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID)
	coin := sdk.NewCoin(denom.IBCDenom(), amount)
	suite.Require().NoError(
		banktestutil.FundAccount(
			suite.chainB.GetContext(),
			suite.chainB.GetSimApp().BankKeeper,
			escrowAddress,
			sdk.NewCoins(coin),
		),
	)

	data := types.NewInternalTransferRepresentation(
		types.Token{
			Denom:  denom,
			Amount: amount.String(),
		}, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), "")
	sourcePort := path2.EndpointB.ChannelConfig.PortID
	sourceChannel := path2.EndpointB.ChannelID

	suite.chainB.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainB.GetContext(), coin)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(defaultAmount, totalEscrowChainB.Amount)

	err := suite.chainB.GetSimApp().TransferKeeper.OnTimeoutPacket(suite.chainB.GetContext(), sourcePort, sourceChannel, data)
	suite.Require().NoError(err)

	// check total amount in escrow of sent token on sending chain
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(zeroAmount, totalEscrowChainB.Amount)
}

func (suite *KeeperTestSuite) TestPacketForwardsCompatibility() {
	// We are testing a scenario where a packet in the future has a new populated
	// field called "new_field". And this packet is being sent to this module which
	// doesn't have this field in the packet data. The module should be able to handle
	// this packet without any issues.

	// the test also ensures that an ack is written for any malformed or bad packet data.

	var packetData []byte
	var path *ibctesting.Path

	testCases := []struct {
		msg         string
		malleate    func()
		expError    error
		expAckError error
	}{
		{
			"success: no new field with memo",
			func() {
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s","memo":"memo"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
			nil,
		},
		{
			"success: no new field without memo",
			func() {
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
			nil,
		},
		{
			"failure: invalid packet data",
			func() {
				packetData = []byte("invalid packet data")
			},
			ibcerrors.ErrInvalidType,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: new field",
			func() {
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s","memo":"memo","new_field":"value"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			ibcerrors.ErrInvalidType,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: missing field",
			func() {
				jsonString := fmt.Sprintf(`{"amount":"100","sender":%s","receiver":"%s"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			ibcerrors.ErrInvalidType,
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset
			packetData = nil

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.EndpointA.ChannelConfig.Version = types.V1
			path.EndpointB.ChannelConfig.Version = types.V1

			tc.malleate()

			path.Setup()

			timeoutHeight := suite.chainB.GetTimeoutHeight()

			seq, err := path.EndpointB.SendPacket(timeoutHeight, 0, packetData)
			suite.Require().NoError(err)

			packet := channeltypes.NewPacket(packetData, seq, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, timeoutHeight, 0)

			// receive packet on chainA
			err = path.RelayPacket(packet)

			if tc.expError == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorContains(err, tc.expError.Error())
				ackBz, ok := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetPacketAcknowledgement(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, seq)
				suite.Require().True(ok)

				// an ack should be written for the malformed / bad packet data.
				expectedAck := channeltypes.NewErrorAcknowledgement(tc.expAckError)
				expBz := channeltypes.CommitAcknowledgement(expectedAck.Acknowledgement())
				suite.Require().Equal(expBz, ackBz)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCreatePacketDataBytesFromVersion() {
	var (
		token            types.Token
		sender, receiver string
	)

	testCases := []struct {
		name       string
		appVersion string
		malleate   func()
		expResult  func(bz []byte, err error)
	}{
		{
			"success",
			types.V1,
			func() {},
			func(bz []byte, err error) {
				expPacketData := types.NewFungibleTokenPacketData(ibctesting.TestCoin.Denom, ibctesting.TestCoin.Amount.String(), sender, receiver, "")
				suite.Require().Equal(bz, expPacketData.GetBytes())
				suite.Require().NoError(err)
			},
		},
		{
			"failure: version 2",
			"ics20-2",
			func() {},
			func(bz []byte, err error) {
				suite.Require().Nil(bz)
				suite.Require().Error(err, ibcerrors.ErrInvalidVersion)
			},
		},
		{
			"failure: fails v1 validation",
			types.V1,
			func() {
				sender = ""
			},
			func(bz []byte, err error) {
				suite.Require().Nil(bz)
				suite.Require().ErrorIs(err, ibcerrors.ErrInvalidAddress)
			},
		},
		{
			"failure: invalid version",
			ibcmock.Version,
			func() {},
			func(bz []byte, err error) {
				suite.Require().Nil(bz)
				suite.Require().ErrorIs(err, types.ErrInvalidVersion)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			token = types.Token{
				Amount: ibctesting.TestCoin.Amount.String(),
				Denom:  types.NewDenom(ibctesting.TestCoin.Denom),
			}

			sender = suite.chainA.SenderAccount.GetAddress().String()
			receiver = suite.chainB.SenderAccount.GetAddress().String()

			tc.malleate()

			bz, err := transferkeeper.CreatePacketDataBytesFromVersion(tc.appVersion, sender, receiver, "", token)

			tc.expResult(bz, err)
		})
	}
}

// metadataFromDenom creates a banktypes.Metadata from a given types.Denom
func metadataFromDenom(denom types.Denom) banktypes.Metadata {
	return banktypes.Metadata{
		Description: fmt.Sprintf("IBC token from %s", denom.Path()),
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom.Base,
				Exponent: 0,
			},
		},
		Base:    denom.IBCDenom(),
		Display: denom.Path(),
		Name:    fmt.Sprintf("%s IBC token", denom.Path()),
		Symbol:  strings.ToUpper(denom.Base),
	}
}

// assertEscrowEqual asserts that the amounts escrowed for each of the coins on chain matches the expectedAmounts
func (suite *KeeperTestSuite) assertEscrowEqual(chain *ibctesting.TestChain, coin sdk.Coin, expectedAmount sdkmath.Int) {
	amount := chain.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(chain.GetContext(), coin.GetDenom())
	suite.Require().Equal(expectedAmount, amount.Amount)
}
