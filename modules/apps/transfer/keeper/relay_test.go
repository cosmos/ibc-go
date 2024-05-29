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

	convertinternal "github.com/cosmos/ibc-go/v8/modules/apps/transfer/internal/convert"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// TestSendTransfer tests sending from chainA to chainB with multiple coins.
// It has cases for coins both native to chain A and coins received to chain
// A from B.
func (suite *KeeperTestSuite) TestSendTransfer() {
	var (
		msg  *types.MsgTransfer
		path *ibctesting.Path
	)

	// NoOp function for failure cases.
	noOpValidation := func(_ sdk.Coins) {}
	testCases := []struct {
		name     string
		malleate func()
		validate func(coins sdk.Coins)
		expError error
	}{
		{
			"successful transfer of single native coin",
			func() {},
			func(coins sdk.Coins) {
				// Sent single coin.
				coin := coins[0]
				amount := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
				suite.Require().Equal(amount, coin)
			},
			nil,
		},
		{
			"successful transfer of single native coin with memo",
			func() {
				msg.Memo = "memo" //nolint:goconst
			},
			func(coins sdk.Coins) {
				// Sent single coin.
				coin := coins[0]
				amount := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())

				suite.Require().Equal(amount, coin)
			},
			nil,
		},
		{
			"successful transfer of [native coin, native coin]",
			func() {
				// does not make sense as a transfer but testing total escrowed is incremented.
				msg.Tokens = append(msg.Tokens, ibctesting.TestCoin)
			},
			func(coins sdk.Coins) {
				// Escrowed amount should be equal to sum of coins sent.
				totalExpectedEscrowed := sdkmath.NewInt(0)
				for _, coin := range coins {
					totalExpectedEscrowed = totalExpectedEscrowed.Add(coin.Amount)
				}

				totalEscrowed := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.DefaultBondDenom)
				suite.Require().Equal(totalEscrowed.Amount, totalExpectedEscrowed)
			},
			nil,
		},
		{
			"successful transfer of [native coin, IBC coin]",
			func() {
				// send IBC coin back to chainB
				coin := types.GetTransferCoin(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.DefaultBondDenom, sdkmath.NewInt(100))

				msg.Tokens = append(msg.Tokens, coin)
			},
			func(coins sdk.Coins) {
				// Escrowed amount for native should equal coin, for IBC coin it should be zero.
				for _, coin := range coins {
					amount := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
					if strings.HasPrefix(coin.GetDenom(), "ibc/") {
						suite.Require().Equal(amount, sdk.NewCoin(coin.GetDenom(), sdkmath.NewInt(0)))
					} else {
						suite.Require().Equal(amount.Amount.Int64(), coin.Amount.Int64())
					}
				}
			},
			nil,
		},
		{
			"successful transfer of [native coin, IBC coin] with memo",
			func() {
				// send IBC coin back to chainB
				coin := types.GetTransferCoin(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.DefaultBondDenom, sdkmath.NewInt(100))

				msg.Tokens = append(msg.Tokens, coin)
				msg.Memo = "memo"
			},
			func(coins sdk.Coins) {
				// Escrowed amount for native should equal coin, for IBC coin it should be zero.
				for _, coin := range coins {
					amount := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
					if strings.HasPrefix(coin.GetDenom(), "ibc/") {
						suite.Require().Equal(amount, sdk.NewCoin(coin.GetDenom(), sdkmath.NewInt(0)))
					} else {
						suite.Require().Equal(amount.Amount.Int64(), coin.Amount.Int64())
					}
				}
			},
			nil,
		},
		{
			"failure: source channel not found",
			func() {
				// channel references wrong ID
				msg.SourceChannel = ibctesting.InvalidID
			},
			noOpValidation,
			ibcerrors.ErrInvalidRequest,
		},
		{
			"failure: sender account is blocked",
			func() {
				msg.Sender = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
			},
			noOpValidation,
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: bank send from sender account failed, insufficient balance",
			func() {
				msg.Tokens = []sdk.Coin{sdk.NewCoin("randomdenom", sdkmath.NewInt(100))}
			},
			noOpValidation,
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: denom trace not found",
			func() {
				coin := types.GetTransferCoin(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, "randomdenom", sdkmath.NewInt(100))
				msg.Tokens = []sdk.Coin{coin}
			},
			noOpValidation,
			types.ErrTraceNotFound,
		},
		{
			"failure: bank send from module account failed, insufficient balance",
			func() {
				coin := types.GetTransferCoin(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.DefaultBondDenom, sdkmath.NewInt(101))
				msg.Tokens = []sdk.Coin{coin}
			},
			noOpValidation,
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: channel capability not found",
			func() {
				capability := suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				// Release channel capability
				suite.chainA.GetSimApp().ScopedTransferKeeper.ReleaseCapability(suite.chainA.GetContext(), capability) //nolint:errcheck // ignore error for testing
			},
			noOpValidation,
			channeltypes.ErrChannelCapabilityNotFound,
		},
		{
			"failure: timeout height and timeout timestamp are zero",
			func() {
				msg.TimeoutHeight = clienttypes.ZeroHeight()
			},
			noOpValidation,
			channeltypes.ErrInvalidPacket,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			chainAAddress := suite.chainA.SenderAccount.GetAddress().String()
			chainBAddress := suite.chainB.SenderAccount.GetAddress().String()

			// Message from B -> A to create IBC coin on chain A.
			msg = types.NewMsgTransfer(
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				ibctesting.TestCoins,
				chainBAddress,
				chainAAddress,
				suite.chainB.GetTimeoutHeight(),
				0, // only use timeout height
				"",
			)

			result, err := suite.chainB.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err := ibctesting.ParsePacketFromEvents(result.Events)
			suite.Require().NoError(err)

			err = path.RelayPacket(packet)
			suite.Require().NoError(err)

			// Malleable message for test cases, transfer from A -> B.
			msg = types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				ibctesting.TestCoins,
				chainAAddress,
				chainBAddress,
				suite.chainA.GetTimeoutHeight(), 0, // only use timeout height
				"",
			)

			tc.malleate()

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(suite.chainA.GetContext(), msg)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NotNil(res)
				suite.Require().NoError(err)
			} else {
				suite.Require().Nil(res)
				suite.Require().ErrorIs(err, tc.expError)
			}

			// Let tests do any necessary post tx validation.
			tc.validate(msg.GetCoins())
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
		in this case.

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
	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		ibctesting.TestCoins,
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
	trace := types.ParseDenomTrace(types.GetPrefixedDenom(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, sdk.DefaultBondDenom))
	coin := sdk.NewCoin(trace.IBCDenom(), sdkmath.NewInt(100))
	msg := types.NewMsgTransfer(
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		sdk.NewCoins(coin),
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(), 0, "",
	)

	res, err := suite.chainB.GetSimApp().TransferKeeper.Transfer(suite.chainB.GetContext(), msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// check total amount in escrow of sent token on sending chain
	totalEscrow := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrow.Amount)
}

// TestOnRecvPacket_ReceiverIsNotSource tests receiving on chainB coins that
// originate on chainA. The bulk of the testing occurs  in the test case for
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
				packetData.Tokens[0].Amount = sdkmath.ZeroInt().String()
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: receiver is module account",
			func() {
				packetData.Receiver = suite.chainB.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
			},
			sdkerrors.ErrUnauthorized,
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

			chainAAddress := suite.chainA.SenderAccount.GetAddress().String()
			chainBAddress := suite.chainB.SenderAccount.GetAddress().String()

			// denom trace of tokens received on chain B and the associated expected metadata
			denomTraceOnB := types.ParseDenomTrace(types.GetPrefixedDenom(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.DefaultBondDenom))
			expDenomMetadataOnB := banktypes.Metadata{
				Description: fmt.Sprintf("IBC token from %s", denomTraceOnB.GetFullDenomPath()),
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    denomTraceOnB.GetBaseDenom(),
						Exponent: 0,
					},
				},
				Base:    denomTraceOnB.IBCDenom(),
				Display: denomTraceOnB.GetFullDenomPath(),
				Name:    fmt.Sprintf("%s IBC token", denomTraceOnB.GetFullDenomPath()),
				Symbol:  strings.ToUpper(denomTraceOnB.GetBaseDenom()),
			}

			// initiate transfer of coins from chainA to chainB
			coins := append(ibctesting.TestCoins, ibctesting.TestCoin)
			transferMsg := types.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coins, chainAAddress, chainBAddress, clienttypes.NewHeight(1, 110), 0, "")

			var tokens types.Tokens
			for _, coin := range transferMsg.GetCoins() {
				tokens = append(tokens, types.Token{types.Denom{coin.Denom, []string{}}, coin.Amount.String()})
			}

			packetData = types.NewFungibleTokenPacketDataV2(tokens, chainAAddress, chainBAddress, "")

			tc.malleate()

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(suite.chainA.GetContext(), transferMsg)
			suite.Require().NoError(err) // message committed

			packet := channeltypes.NewPacket(packetData.GetBytes(), res.Sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)

			err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, packetData)

			// check total amount in escrow of received token denom on receiving chain
			totalEscrow := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), denomTraceOnB.IBCDenom())
			suite.Require().Equal(sdkmath.NewInt(0), totalEscrow.Amount)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				denomMetadata, found := suite.chainB.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainB.GetContext(), denomTraceOnB.IBCDenom())
				suite.Require().True(found)
				suite.Require().Equal(expDenomMetadataOnB, denomMetadata)

				// Ensure all tokens got through by checking supply created.
				expectedSupply := sdkmath.NewInt(0)
				for _, coin := range transferMsg.GetCoins() {
					expectedSupply = expectedSupply.Add(coin.Amount)
				}

				supply := suite.chainB.GetSimApp().BankKeeper.GetSupply(suite.chainB.GetContext(), denomTraceOnB.IBCDenom())
				suite.Require().Equal(expectedSupply.String(), supply.Amount.String())
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
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
		denomTrace      types.DenomTrace
		amount          sdkmath.Int
		receiver        string
		memo            string
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
			"successful receive of half the amount",
			func() {
				amount = sdkmath.NewInt(50)
				expEscrowAmount = sdkmath.NewInt(50)
			},
			nil,
		},
		{
			"successful receive with memo",
			func() {
				memo = "memo"
			},
			nil,
		},
		{
			"failure: empty coin",
			func() {
				amount = sdkmath.ZeroInt()
				expEscrowAmount = sdkmath.NewInt(100)
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: empty denom",
			func() {
				denomTrace = types.DenomTrace{}
				expEscrowAmount = sdkmath.NewInt(100)
			},
			types.ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid receiver address",
			func() {
				receiver = "gaia1scqhwpgsmr6vmztaa7suurfl52my6nd2kmrudl"
				expEscrowAmount = sdkmath.NewInt(100)
			},
			errors.New("failed to decode receiver address"),
		},
		{
			"failure: tries to unescrow more tokens than allowed",
			func() {
				amount = sdkmath.NewInt(1000000)
				expEscrowAmount = sdkmath.NewInt(100)
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: receiver is module account",
			func() {
				receiver = suite.chainB.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
				expEscrowAmount = sdkmath.NewInt(100)
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

			receiver = suite.chainB.SenderAccount.GetAddress().String() // must be explicitly changed in malleate
			memo = ""                                                   // can be explicitly changed in malleate
			amount = sdkmath.NewInt(100)                                // must be explicitly changed in malleate
			expEscrowAmount = sdkmath.ZeroInt()                         // total amount in escrow of voucher denom on receiving chain

			seq := uint64(1)

			// send coin from chainB to chainA, receive them, acknowledge them
			transferMsg := types.NewMsgTransfer(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, ibctesting.TestCoins, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), clienttypes.NewHeight(1, 110), 0, memo)
			res, err := suite.chainB.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			packet, err := ibctesting.ParsePacketFromEvents(res.Events)
			suite.Require().NoError(err)

			err = path.RelayPacket(packet)
			suite.Require().NoError(err) // relay committed

			seq++

			// NOTE: trace must be explicitly changed in malleate to test invalid cases
			denomTrace = types.ParseDenomTrace(types.GetPrefixedDenom(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.DefaultBondDenom))

			// send coin back from chainA to chainB
			coin := sdk.NewCoin(denomTrace.IBCDenom(), amount)
			transferMsg = types.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.NewCoins(coin), suite.chainA.SenderAccount.GetAddress().String(), receiver, clienttypes.NewHeight(1, 110), 0, memo)
			_, err = suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			tc.malleate()

			denom, trace := convertinternal.ExtractDenomAndTraceFromV1Denom(denomTrace.GetFullDenomPath())
			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  denom,
							Trace: trace,
						},
						Amount: amount.String(),
					},
				}, suite.chainA.SenderAccount.GetAddress().String(), receiver, memo)
			packet = channeltypes.NewPacket(data.GetBytes(), seq, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)

			err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, data)

			// check total amount in escrow of received token denom on receiving chain
			totalEscrow := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), sdk.DefaultBondDenom)
			suite.Require().Equal(expEscrowAmount, totalEscrow.Amount)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				_, found := suite.chainB.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainB.GetContext(), sdk.DefaultBondDenom)
				suite.Require().False(found)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.expError.Error())
			}
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
	amount := sdkmath.NewInt(100)
	timeout := suite.chainA.GetTimeoutHeight()

	// setup
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// denomTrace path: {transfer/channel-1/transfer/channel-0}
	denomTrace := types.DenomTrace{
		BaseDenom: sdk.DefaultBondDenom,
		Path:      fmt.Sprintf("%s/%s/%s/%s", path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID),
	}

	denom, trace := convertinternal.ExtractDenomAndTraceFromV1Denom(denomTrace.GetFullDenomPath())
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom: types.Denom{
					Base:  denom,
					Trace: trace,
				},
				Amount: amount.String(),
			},
		}, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "")
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
	// denomTrace path: transfer/channel-0
	denomTrace = types.DenomTrace{
		BaseDenom: sdk.DefaultBondDenom,
		Path:      fmt.Sprintf("%s/%s", path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID),
	}
	escrowAddress := types.GetEscrowAddress(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID)
	coin := sdk.NewCoin(denomTrace.IBCDenom(), amount)
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
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	// execute onRecvPacket, when chaninB receives the source token the escrow amount should decrease
	err := suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, data)
	suite.Require().NoError(err)

	// check total amount in escrow of sent token on receiving chain
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.ZeroInt(), totalEscrowChainB.Amount)
}

// TestOnAcknowledgementPacket tests that successful acknowledgement is a no-op
// and failure acknowledment leads to refund when attempting to send from chainA
// to chainB. If sender is source then the denomination being refunded has no
// trace.
func (suite *KeeperTestSuite) TestOnAcknowledgementPacket() {
	var (
		successAck      = channeltypes.NewResultAcknowledgement([]byte{byte(1)})
		failedAck       = channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed packet transfer"))
		denomTrace      types.DenomTrace
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
				denomTrace = types.ParseDenomTrace(types.GetPrefixedDenom(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.DefaultBondDenom))
			},
			nil,
		},
		{
			"failed ack: successful refund of native coin",
			failedAck,
			func() {
				escrow := types.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				denomTrace = types.ParseDenomTrace(sdk.DefaultBondDenom)
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
				denomTrace = types.ParseDenomTrace(types.GetPrefixedDenom(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.DefaultBondDenom))
				coin := sdk.NewCoin(denomTrace.IBCDenom(), amount)

				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrow, sdk.NewCoins(coin)))
			},
			nil,
		},
		{
			"failed ack: funds cannot be refunded because escrow account has zero balance",
			failedAck,
			func() {
				denomTrace = types.ParseDenomTrace(sdk.DefaultBondDenom)

				// set escrow amount that would have been stored after successful execution of MsgTransfer
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(sdk.DefaultBondDenom, amount))
				expEscrowAmount = sdkmath.NewInt(100)
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

			amount = sdkmath.NewInt(100) // must be explicitly changed
			expEscrowAmount = sdkmath.ZeroInt()

			tc.malleate()

			denom, trace := convertinternal.ExtractDenomAndTraceFromV1Denom(denomTrace.GetFullDenomPath())
			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  denom,
							Trace: trace,
						},
						Amount: amount.String(),
					},
				}, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "")
			packet := channeltypes.NewPacket(data.GetBytes(), 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)
			preAcknowledgementBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denomTrace.IBCDenom())

			err := suite.chainA.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainA.GetContext(), packet, data, tc.ack)

			// check total amount in escrow of sent token denom on sending chain
			totalEscrow := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), denomTrace.IBCDenom())
			suite.Require().Equal(expEscrowAmount, totalEscrow.Amount)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				postAcknowledgementBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denomTrace.IBCDenom())
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
	amount := sdkmath.NewInt(100)
	ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed packet transfer"))

	// set up
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// fund escrow account for transfer and channel-1 on chain B
	// denomTrace path = transfer/channel-0
	denomTrace := types.DenomTrace{
		BaseDenom: sdk.DefaultBondDenom,
		Path:      fmt.Sprintf("%s/%s", path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID),
	}
	escrowAddress := types.GetEscrowAddress(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID)
	coin := sdk.NewCoin(denomTrace.IBCDenom(), amount)
	suite.Require().NoError(
		banktestutil.FundAccount(
			suite.chainB.GetContext(),
			suite.chainB.GetSimApp().BankKeeper,
			escrowAddress,
			sdk.NewCoins(coin),
		),
	)

	denom, trace := convertinternal.ExtractDenomAndTraceFromV1Denom(denomTrace.GetFullDenomPath())
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom: types.Denom{
					Base:  denom,
					Trace: trace,
				},
				Amount: amount.String(),
			},
		},
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainA.SenderAccount.GetAddress().String(),
		"",
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
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	err := suite.chainB.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainB.GetContext(), packet, data, ack)
	suite.Require().NoError(err)

	// check total amount in escrow of sent token on sending chain
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.ZeroInt(), totalEscrowChainB.Amount)
}

// TestOnTimeoutPacket tests private refundPacket function since it is a simple
// wrapper over it. The actual timeout does not matter since IBC core logic
// is not being tested. The test is timing out a send from chainA to chainB
// so the refunds are occurring on chainA.
func (suite *KeeperTestSuite) TestOnTimeoutPacket() {
	var (
		path            *ibctesting.Path
		amount          sdkmath.Int
		sender          string
		denomTrace      types.DenomTrace
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
				denomTrace = types.ParseDenomTrace(sdk.DefaultBondDenom)
				coin := sdk.NewCoin(denomTrace.IBCDenom(), amount)
				expEscrowAmount = sdkmath.ZeroInt()

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
				denomTrace = types.ParseDenomTrace(types.GetPrefixedDenom(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.DefaultBondDenom))
				coin := sdk.NewCoin(denomTrace.IBCDenom(), amount)
				expEscrowAmount = sdkmath.ZeroInt()

				// funds the escrow account to have balance
				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrow, sdk.NewCoins(coin)))
			},
			nil,
		},
		{
			"failure: funds cannot be refunded because escrow account has no balance for non-native coin",
			func() {
				denomTrace = types.ParseDenomTrace("bitcoin")
				expEscrowAmount = amount

				// set escrow amount that would have been stored after successful execution of MsgTransfer
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(denomTrace.IBCDenom(), amount))
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: funds cannot be refunded because escrow account has no balance for native coin",
			func() {
				denomTrace = types.ParseDenomTrace(sdk.DefaultBondDenom)
				expEscrowAmount = amount

				// set escrow amount that would have been stored after successful execution of MsgTransfer
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), sdk.NewCoin(denomTrace.IBCDenom(), amount))
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: cannot mint because sender address is invalid",
			func() {
				denomTrace = types.ParseDenomTrace(sdk.DefaultBondDenom)
				amount = sdkmath.OneInt()
				sender = "invalid address"
			},
			errors.New("decoding bech32 failed"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			amount = sdkmath.NewInt(100) // must be explicitly changed
			sender = suite.chainA.SenderAccount.GetAddress().String()
			expEscrowAmount = sdkmath.ZeroInt()

			tc.malleate()

			denom, trace := convertinternal.ExtractDenomAndTraceFromV1Denom(denomTrace.GetFullDenomPath())
			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  denom,
							Trace: trace,
						},
						Amount: amount.String(),
					},
				}, sender, suite.chainB.SenderAccount.GetAddress().String(), "")
			packet := channeltypes.NewPacket(data.GetBytes(), 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)
			preTimeoutBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denomTrace.IBCDenom())

			err := suite.chainA.GetSimApp().TransferKeeper.OnTimeoutPacket(suite.chainA.GetContext(), packet, data)

			postTimeoutBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), denomTrace.IBCDenom())
			deltaAmount := postTimeoutBalance.Amount.Sub(preTimeoutBalance.Amount)

			// check total amount in escrow of sent token denom on sending chain
			totalEscrow := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), denomTrace.IBCDenom())
			suite.Require().Equal(expEscrowAmount, totalEscrow.Amount)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(amount.Int64(), deltaAmount.Int64(), "successful timeout did not trigger refund")
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
	amount := sdkmath.NewInt(100)

	// set up
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// fund escrow account for transfer and channel-1 on chain B
	denomTrace := types.DenomTrace{
		BaseDenom: sdk.DefaultBondDenom,
		Path:      fmt.Sprintf("%s/%s", path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID),
	}
	escrowAddress := types.GetEscrowAddress(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID)
	coin := sdk.NewCoin(denomTrace.IBCDenom(), amount)
	suite.Require().NoError(
		banktestutil.FundAccount(
			suite.chainB.GetContext(),
			suite.chainB.GetSimApp().BankKeeper,
			escrowAddress,
			sdk.NewCoins(coin),
		),
	)

	denom, trace := convertinternal.ExtractDenomAndTraceFromV1Denom(denomTrace.GetFullDenomPath())
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom: types.Denom{
					Base:  denom,
					Trace: trace,
				},
				Amount: amount.String(),
			},
		}, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), "")
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
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	err := suite.chainB.GetSimApp().TransferKeeper.OnTimeoutPacket(suite.chainB.GetContext(), packet, data)
	suite.Require().NoError(err)

	// check total amount in escrow of sent token on sending chain
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.ZeroInt(), totalEscrowChainB.Amount)
}

func (suite *KeeperTestSuite) TestPacketForwardsCompatibility() {
	// We are testing a scenario where a packet in the future has a new populated
	// field called "new_field". And this packet is being sent to this module which
	// doesn't have this field in the packet data. The module should be able to handle
	// this packet without any issues.

	var packetData []byte

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"success: new field",
			func() {
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s","memo":"memo","new_field":"value"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
		},
		{
			"success: no new field with memo",
			func() {
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s","memo":"memo"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
		},
		{
			"success: no new field without memo",
			func() {
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
		},
		{
			"failure: invalid packet data",
			func() {
				packetData = []byte("invalid packet data")
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: missing field",
			func() {
				jsonString := fmt.Sprintf(`{"amount":"100","sender":%s","receiver":"%s"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset
			packetData = nil

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.EndpointA.ChannelConfig.Version = types.V1
			path.EndpointB.ChannelConfig.Version = types.V1
			path.Setup()

			tc.malleate()

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
			}
		})
	}
}
