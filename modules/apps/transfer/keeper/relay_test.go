package keeper_test

import (
	"errors"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	transferkeeper "github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	ibcmock "github.com/cosmos/ibc-go/v9/testing/mock"
)

var (
	zeroAmount    = sdkmath.NewInt(0)
	defaultAmount = ibctesting.DefaultCoinAmount
)

// TestSendTransfer tests sending from chainA to chainB using both coin
// that originate on chainA and coin that originate on chainB.
func (suite *KeeperTestSuite) TestSendTransfer() {
	var (
		coins            sdk.Coins
		path             *ibctesting.Path
		sender           sdk.AccAddress
		memo             string
		forwarding       *types.Forwarding
		expEscrowAmounts []sdkmath.Int // total amounts in escrow for denom on receiving chain
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
			"successful transfer with non-empty forwarding hops and ics20-2",
			func() {
				forwarding = types.NewForwarding(false, types.NewHop(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
				))
			},
			nil,
		},
		{
			"successful transfer of IBC token",
			func() {
				// send IBC token back to chainB
				denom := types.NewDenom(ibctesting.TestCoin.Denom, types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coins = sdk.NewCoins(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount))

				expEscrowAmounts = []sdkmath.Int{zeroAmount}
			},
			nil,
		},
		{
			"successful transfer of IBC token with memo",
			func() {
				// send IBC token back to chainB
				denom := types.NewDenom(ibctesting.TestCoin.Denom, types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coins = sdk.NewCoins(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount))
				memo = "memo"

				expEscrowAmounts = []sdkmath.Int{zeroAmount}
			},
			nil,
		},
		{
			"successful transfer of native token with ics20-1",
			func() {
				coins = sdk.NewCoins(coins[0])

				// Set version to isc20-1.
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.Version = types.V1
				})
			},
			nil,
		},
		{
			"successful transfer with empty forwarding hops and ics20-1",
			func() {
				coins = sdk.NewCoins(coins[0])

				// Set version to isc20-1.
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.Version = types.V1
				})
			},
			nil,
		},
		{
			"failure: source channel not found",
			func() {
				// channel references wrong ID
				path.EndpointA.ChannelID = ibctesting.InvalidID
			},
			channeltypes.ErrChannelNotFound,
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
				coins = sdk.NewCoins(sdk.NewCoin("randomdenom", defaultAmount))
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: denom trace not found",
			func() {
				denom := types.NewDenom("randomdenom", types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coins = sdk.NewCoins(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount))
			},
			types.ErrDenomNotFound,
		},
		{
			"failure: bank send from module account failed, insufficient balance",
			func() {
				denom := types.NewDenom(ibctesting.TestCoin.Denom, types.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coins = sdk.NewCoins(sdk.NewCoin(denom.IBCDenom(), ibctesting.TestCoin.Amount.Add(sdkmath.NewInt(1))))
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: channel capability not found",
			func() {
				capability := suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				// Release channel capability
				suite.chainA.GetSimApp().ScopedTransferKeeper.ReleaseCapability(suite.chainA.GetContext(), capability) //nolint:errcheck // ignore error for testing
			},
			channeltypes.ErrChannelCapabilityNotFound,
		},
		{
			"failure: forwarding hops is not empty with ics20-1",
			func() {
				// Set version to isc20-1.
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.Version = types.V1
				})

				forwarding = types.NewForwarding(false, types.NewHop(
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
				))
			},
			ibcerrors.ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			// create IBC token on chainA
			transferMsg := types.NewMsgTransfer(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.NewCoins(ibctesting.TestCoin), suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainA.GetTimeoutHeight(), 0, "", nil)

			result, err := suite.chainB.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			packet, err := ibctesting.ParsePacketFromEvents(result.Events)
			suite.Require().NoError(err)

			err = path.RelayPacket(packet)
			suite.Require().NoError(err)

			// Value that can malleated for Transfer we are testing.
			coins = ibctesting.TestCoins
			sender = suite.chainA.SenderAccount.GetAddress()
			memo = ""
			expEscrowAmounts = []sdkmath.Int{defaultAmount, defaultAmount}
			forwarding = nil

			tc.malleate()

			msg := types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				coins,
				sender.String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				suite.chainB.GetTimeoutHeight(), 0, // only use timeout height
				memo,
				forwarding,
			)

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(suite.chainA.GetContext(), msg)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NotNil(res)
				suite.Require().NoError(err)
			} else {
				suite.Require().Nil(res)
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)

				// We do not expect escrowed amounts in error cases.
				expEscrowAmounts = []sdkmath.Int{zeroAmount, zeroAmount}
			}
			// Assert amounts escrowed are as expected.
			suite.assertEscrowEqual(suite.chainA, coins, expEscrowAmounts)
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
		sdk.NewCoins(coin),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainB.GetTimeoutHeight(), 0, "",
		nil,
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
		sdk.NewCoins(coin),
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(), 0, "",
		nil,
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
	var packetData types.FungibleTokenPacketDataV2

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
				packetData.Tokens[0].Amount = zeroAmount.String()
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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			receiver := suite.chainB.SenderAccount.GetAddress().String() // must be explicitly changed in malleate

			// send coins from chainA to chainB
			transferMsg := types.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.TestCoins, suite.chainA.SenderAccount.GetAddress().String(), receiver, clienttypes.NewHeight(1, 110), 0, "", nil)
			_, err := suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			var tokens []types.Token
			for _, coin := range ibctesting.TestCoins {
				tokens = append(tokens, types.Token{Denom: types.NewDenom(coin.Denom), Amount: coin.Amount.String()})
			}
			packetData = types.NewFungibleTokenPacketDataV2(tokens, suite.chainA.SenderAccount.GetAddress().String(), receiver, "", ibctesting.EmptyForwardingPacketData)
			packet := channeltypes.NewPacket(packetData.GetBytes(), uint64(1), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)

			tc.malleate()

			var denoms []types.Denom
			for _, token := range packetData.Tokens {
				// construct expected denom B will construct after running Recv logic.
				denoms = append(denoms, types.NewDenom(token.Denom.Base, types.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)))
			}

			err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, packetData)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				// Check denom metadata for of tokens received on chain B.
				for _, denom := range denoms {
					actualMetadata, found := suite.chainB.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainB.GetContext(), denom.IBCDenom())

					suite.Require().True(found)
					suite.Require().Equal(metadataFromDenom(denom), actualMetadata)
				}
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)

				// Check denom metadata absence for cases where recv fails.
				for _, denom := range denoms {
					_, found := suite.chainB.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainB.GetContext(), denom.IBCDenom())

					suite.Require().False(found)
				}
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
		packetData       types.FungibleTokenPacketDataV2
		expEscrowAmounts []sdkmath.Int // total amount in escrow for denom on receiving chain
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
				packetData.Tokens[0].Amount = sdkmath.NewInt(50).String()
				// expect 50 remaining for coin 1, nothing for coin 2.
				expEscrowAmounts = []sdkmath.Int{sdkmath.NewInt(50), zeroAmount}
			},
			nil,
		},
		{
			"failure: empty coin",
			func() {
				packetData.Tokens[0].Amount = zeroAmount.String()
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: tries to unescrow more tokens than allowed",
			func() {
				packetData.Tokens[0].Amount = sdkmath.NewInt(1000000).String()
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: empty denom",
			func() {
				packetData.Tokens[0].Denom = types.Denom{}
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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			receiver := suite.chainB.SenderAccount.GetAddress().String() // must be explicitly changed in malleate
			expEscrowAmounts = []sdkmath.Int{zeroAmount, zeroAmount}     // total amount in escrow of voucher denom on receiving chain

			// send coins from chainA to chainB, receive them, acknowledge them
			transferMsg := types.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.TestCoins, suite.chainA.SenderAccount.GetAddress().String(), receiver, clienttypes.NewHeight(1, 110), 0, "", nil)
			_, err := suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			var tokens []types.Token
			for _, coin := range ibctesting.TestCoins {
				tokens = append(tokens, types.Token{Denom: types.NewDenom(coin.Denom, types.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)), Amount: coin.Amount.String()})
			}
			packetData = types.NewFungibleTokenPacketDataV2(tokens, suite.chainA.SenderAccount.GetAddress().String(), receiver, "", ibctesting.EmptyForwardingPacketData)

			tc.malleate()

			packet := channeltypes.NewPacket(packetData.GetBytes(), uint64(1), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)
			err = suite.chainA.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainA.GetContext(), packet, packetData)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				_, found := suite.chainA.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainA.GetContext(), sdk.DefaultBondDenom)
				suite.Require().False(found)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.expError.Error())

				// Expect escrowed amount to stay same on failure.
				expEscrowAmounts = []sdkmath.Int{defaultAmount, defaultAmount}
			}

			// Assert amounts escrowed are as expected, we do not malleate amount escrowed in initial transfer.
			suite.assertEscrowEqual(suite.chainA, ibctesting.TestCoins, expEscrowAmounts)
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

	seq := uint64(1)
	amount := defaultAmount
	timeout := suite.chainA.GetTimeoutHeight()

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

	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", ibctesting.EmptyForwardingPacketData)
	packet := channeltypes.NewPacket(
		data.GetBytes(),
		seq,
		path2.EndpointA.ChannelConfig.PortID,
		path2.EndpointA.ChannelID,
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		timeout, 0,
	)

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
	err := suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, data)
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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			amount = defaultAmount // must be explicitly changed
			expEscrowAmount = zeroAmount

			tc.malleate()

			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  denom,
						Amount: amount.String(),
					},
				}, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", ibctesting.EmptyForwardingPacketData)
			packet := channeltypes.NewPacket(data.GetBytes(), 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)
			preAcknowledgementBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom.IBCDenom())

			err := suite.chainA.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainA.GetContext(), packet, data, tc.ack)

			// check total amount in escrow of sent token denom on sending chain
			totalEscrow := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), denom.IBCDenom())
			suite.Require().Equal(expEscrowAmount, totalEscrow.Amount)

			expPass := tc.expError == nil
			if expPass {
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

	seq := uint64(1)
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

	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		},
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainA.SenderAccount.GetAddress().String(),
		"",
		ibctesting.EmptyForwardingPacketData,
	)
	packet := channeltypes.NewPacket(
		data.GetBytes(),
		seq,
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		path2.EndpointA.ChannelConfig.PortID,
		path2.EndpointA.ChannelID,
		suite.chainA.GetTimeoutHeight(), 0,
	)

	suite.chainB.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainB.GetContext(), coin)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(defaultAmount, totalEscrowChainB.Amount)

	err := suite.chainB.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainB.GetContext(), packet, data, ack)
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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			amount = defaultAmount.String() // must be explicitly changed
			sender = suite.chainA.SenderAccount.GetAddress().String()
			expEscrowAmount = zeroAmount

			tc.malleate()

			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  denom,
						Amount: amount,
					},
				}, sender, suite.chainB.SenderAccount.GetAddress().String(), "", ibctesting.EmptyForwardingPacketData)
			packet := channeltypes.NewPacket(data.GetBytes(), 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)
			preTimeoutBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom.IBCDenom())

			err := suite.chainA.GetSimApp().TransferKeeper.OnTimeoutPacket(suite.chainA.GetContext(), packet, data)

			postTimeoutBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denom.IBCDenom())
			deltaAmount := postTimeoutBalance.Amount.Sub(preTimeoutBalance.Amount)

			// check total amount in escrow of sent token denom on sending chain
			totalEscrow := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), denom.IBCDenom())
			suite.Require().Equal(expEscrowAmount, totalEscrow.Amount)

			expPass := tc.expError == nil
			if expPass {
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

	seq := uint64(1)
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

	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), "", ibctesting.EmptyForwardingPacketData)
	packet := channeltypes.NewPacket(
		data.GetBytes(),
		seq,
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		path2.EndpointA.ChannelConfig.PortID,
		path2.EndpointA.ChannelID,
		suite.chainA.GetTimeoutHeight(), 0,
	)

	suite.chainB.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainB.GetContext(), coin)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(defaultAmount, totalEscrowChainB.Amount)

	err := suite.chainB.GetSimApp().TransferKeeper.OnTimeoutPacket(suite.chainB.GetContext(), packet, data)
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
		tc := tc
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

			expPass := tc.expError == nil
			if expPass {
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
		tokens           types.Tokens
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
			"success: version 2",
			types.V2,
			func() {},
			func(bz []byte, err error) {
				expPacketData := types.NewFungibleTokenPacketDataV2(tokens, sender, receiver, "", ibctesting.EmptyForwardingPacketData)
				suite.Require().Equal(bz, expPacketData.GetBytes())
				suite.Require().NoError(err)
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
			"failure: fails v2 validation",
			types.V2,
			func() {
				sender = ""
			},
			func(bz []byte, err error) {
				suite.Require().Nil(bz)
				suite.Require().ErrorIs(err, ibcerrors.ErrInvalidAddress)
			},
		},
		{
			"failure: must have single coin if using version 1.",
			types.V1,
			func() {
				tokens = types.Tokens{}
			},
			func(bz []byte, err error) {
				suite.Require().Nil(bz)
				suite.Require().ErrorIs(err, ibcerrors.ErrInvalidRequest)
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

			tokens = types.Tokens{
				{
					Amount: ibctesting.TestCoin.Amount.String(),
					Denom:  types.NewDenom(ibctesting.TestCoin.Denom),
				},
			}

			sender = suite.chainA.SenderAccount.GetAddress().String()
			receiver = suite.chainB.SenderAccount.GetAddress().String()

			tc.malleate()

			bz, err := transferkeeper.CreatePacketDataBytesFromVersion(tc.appVersion, sender, receiver, "", tokens, nil)

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
func (suite *KeeperTestSuite) assertEscrowEqual(chain *ibctesting.TestChain, coins sdk.Coins, expectedAmounts []sdkmath.Int) {
	for i, coin := range coins {
		amount := chain.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(chain.GetContext(), coin.GetDenom())
		suite.Require().Equal(expectedAmounts[i], amount.Amount)
	}
}
