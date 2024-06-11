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

	transferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

// TestSendTransfer tests sending from chainA to chainB using both coin
// that originate on chainA and coin that originate on chainB.
func (suite *KeeperTestSuite) TestSendTransfer() {
	var (
		coin            sdk.Coin
		path            *ibctesting.Path
		sender          sdk.AccAddress
		timeoutHeight   clienttypes.Height
		memo            string
		expEscrowAmount sdkmath.Int // total amount in escrow for denom on receiving chain

	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"successful transfer of native token",
			func() {
				expEscrowAmount = sdkmath.NewInt(100)
			},
			nil,
		},
		{
			"successful transfer of native token with memo",
			func() {
				memo = "memo" //nolint:goconst
				expEscrowAmount = sdkmath.NewInt(100)
			},
			nil,
		},
		{
			"successful transfer of IBC token",
			func() {
				// send IBC token back to chainB
				denom := types.NewDenom(coin.Denom, types.NewTrace(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin = sdk.NewCoin(denom.IBCDenom(), coin.Amount)
			},
			nil,
		},
		{
			"successful transfer of native token with ics20-1",
			func() {
				expEscrowAmount = sdkmath.NewInt(100)

				// Set version to isc20-1.
				path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) {
					channel.Version = types.V1
				})
			},
			nil,
		},
		{
			"successful transfer of IBC token with memo",
			func() {
				// send IBC token back to chainB
				denom := types.NewDenom(coin.Denom, types.NewTrace(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin = sdk.NewCoin(denom.IBCDenom(), coin.Amount)
				memo = "memo"
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
				sender = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName)
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: bank send from sender account failed, insufficient balance",
			func() {
				coin = sdk.NewCoin("randomdenom", sdkmath.NewInt(100))
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"failure: denom trace not found",
			func() {
				denom := types.NewDenom("randomdenom", types.NewTrace(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin = sdk.NewCoin(denom.IBCDenom(), coin.Amount)
			},
			types.ErrDenomNotFound,
		},
		{
			"failure: bank send from module account failed, insufficient balance",
			func() {
				denom := types.NewDenom(coin.Denom, types.NewTrace(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin = sdk.NewCoin(denom.IBCDenom(), coin.Amount.Add(sdkmath.NewInt(1)))
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
			"failure: timeout height and timeout timestamp are zero",
			func() {
				timeoutHeight = clienttypes.ZeroHeight()
				expEscrowAmount = sdkmath.NewInt(100)
			},
			channeltypes.ErrInvalidPacket,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			coin = sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))
			sender = suite.chainA.SenderAccount.GetAddress()
			memo = ""
			timeoutHeight = suite.chainB.GetTimeoutHeight()
			expEscrowAmount = sdkmath.ZeroInt()

			// create IBC token on chainA
			transferMsg := types.NewMsgTransfer(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.NewCoins(coin), suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainA.GetTimeoutHeight(), 0, "", nil)
			result, err := suite.chainB.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			packet, err := ibctesting.ParsePacketFromEvents(result.Events)
			suite.Require().NoError(err)

			err = path.RelayPacket(packet)
			suite.Require().NoError(err)

			tc.malleate()

			msg := types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				sdk.NewCoins(coin),
				sender.String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				timeoutHeight, 0, // only use timeout height
				memo,
				nil,
			)

			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(suite.chainA.GetContext(), msg)

			// check total amount in escrow of sent token denom on sending chain
			amount := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
			suite.Require().Equal(expEscrowAmount, amount.Amount)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NotNil(res)
				suite.Require().NoError(err)
			} else {
				suite.Require().Nil(res)
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
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
	coin := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))
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
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))
	coin = sdk.NewCoin(denom.IBCDenom(), sdkmath.NewInt(100))
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
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrow.Amount)
}

// TestOnRecvPacket_ReceiverIsNotSource tests receiving on chainB a coin that
// originates on chainA. The bulk of the testing occurs  in the test case for
// loop since setup is intensive for all cases. The malleate function allows
// for testing invalid cases.
func (suite *KeeperTestSuite) TestOnRecvPacket_ReceiverIsNotSource() {
	var (
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
			"successful receive with memo",
			func() {
				memo = "memo"
			},
			nil,
		},
		{
			"failure: mint zero coin",
			func() {
				amount = sdkmath.ZeroInt()
			},
			types.ErrInvalidAmount,
		},
		{
			"failure: receiver is module account",
			func() {
				receiver = suite.chainB.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
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

			receiver = suite.chainB.SenderAccount.GetAddress().String() // must be explicitly changed in malleate
			memo = ""                                                   // can be explicitly changed in malleate
			amount = sdkmath.NewInt(100)                                // must be explicitly changed in malleate
			expEscrowAmount = sdkmath.ZeroInt()                         // total amount in escrow of voucher denom on receiving chain

			// denom trace of tokens received on chain B and the associated expected metadata
			denomOnB := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
			expDenomMetadataOnB := banktypes.Metadata{
				Description: fmt.Sprintf("IBC token from %s", denomOnB.Path()),
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    denomOnB.Base,
						Exponent: 0,
					},
				},
				Base:    denomOnB.IBCDenom(),
				Display: denomOnB.Path(),
				Name:    fmt.Sprintf("%s IBC token", denomOnB.Path()),
				Symbol:  strings.ToUpper(denomOnB.Base),
			}

			// send coin from chainA to chainB
			coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
			transferMsg := types.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.NewCoins(coin), suite.chainA.SenderAccount.GetAddress().String(), receiver, clienttypes.NewHeight(1, 110), 0, memo, nil)
			_, err := suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			tc.malleate()

			seq := uint64(1)
			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(sdk.DefaultBondDenom, []types.Trace{}...),
						Amount: amount.String(),
					},
				}, suite.chainA.SenderAccount.GetAddress().String(), receiver, memo, nil)
			packet := channeltypes.NewPacket(data.GetBytes(), seq, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)

			var async bool
			async, err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, data)
			suite.Require().False(async)
			// check total amount in escrow of received token denom on receiving chain
			totalEscrow := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), sdk.DefaultBondDenom)
			suite.Require().Equal(expEscrowAmount, totalEscrow.Amount)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				denomMetadata, found := suite.chainB.GetSimApp().BankKeeper.GetDenomMetaData(suite.chainB.GetContext(), denomOnB.IBCDenom())
				suite.Require().True(found)
				suite.Require().Equal(expDenomMetadataOnB, denomMetadata)
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
		denom           types.Denom
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
				denom = types.Denom{}
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
			coin := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))
			transferMsg := types.NewMsgTransfer(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.NewCoins(coin), suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), clienttypes.NewHeight(1, 110), 0, memo, nil)
			res, err := suite.chainB.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			packet, err := ibctesting.ParsePacketFromEvents(res.Events)
			suite.Require().NoError(err)

			err = path.RelayPacket(packet)
			suite.Require().NoError(err) // relay committed

			seq++

			// NOTE: trace must be explicitly changed in malleate to test invalid cases
			denom = types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))

			// send coin back from chainA to chainB
			coin = sdk.NewCoin(denom.IBCDenom(), amount)
			transferMsg = types.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.NewCoins(coin), suite.chainA.SenderAccount.GetAddress().String(), receiver, clienttypes.NewHeight(1, 110), 0, memo, nil)
			_, err = suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			tc.malleate()

			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  denom,
						Amount: amount.String(),
					},
				}, suite.chainA.SenderAccount.GetAddress().String(), receiver, memo, nil)
			packet = channeltypes.NewPacket(data.GetBytes(), seq, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)

			_, err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, data)

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

func (suite *KeeperTestSuite) TestPathForwarding() {
	amount := sdkmath.NewInt(100)

	// setup
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[1].SenderAccount
	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path2.EndpointA.ChannelConfig.PortID,
				ChannelId: path2.EndpointA.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
	)
	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, packet.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packet, forwardedPacket)
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

	// denom path: {transfer/channel-1/transfer/channel-0}
	denom := types.NewDenom(
		sdk.DefaultBondDenom,
		types.NewTrace(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID),
		types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID),
	)

	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", nil)
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
		types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID),
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
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	// execute onRecvPacket, when chaninB receives the source token the escrow amount should decrease
	var async bool
	async, err := suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packet, data)
	suite.Require().False(async)
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
				denom = types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
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
				denom = types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
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

			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  denom,
						Amount: amount.String(),
					},
				}, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", nil)
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
	amount := sdkmath.NewInt(100)
	ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed packet transfer"))

	// set up
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// fund escrow account for transfer and channel-1 on chain B
	// denom path: transfer/channel-0
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

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
		nil,
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
				denom = types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coinAmount, ok := sdkmath.NewIntFromString(amount)
				suite.Require().True(ok)
				coin := sdk.NewCoin(denom.IBCDenom(), coinAmount)
				expEscrowAmount = sdkmath.ZeroInt()

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

			amount = sdkmath.NewInt(100).String() // must be explicitly changed
			sender = suite.chainA.SenderAccount.GetAddress().String()
			expEscrowAmount = sdkmath.ZeroInt()

			tc.malleate()

			data := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  denom,
						Amount: amount,
					},
				}, sender, suite.chainB.SenderAccount.GetAddress().String(), "", nil)
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
	amount := sdkmath.NewInt(100)

	// set up
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// fund escrow account for transfer and channel-1 on chain B
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

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
		}, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), "", nil)
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

func (suite *KeeperTestSuite) TestEscrowsAreSetAfterForwarding() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topolgy:

		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake

		We want to trigger:
		1. A sends B over channel0.
		2. B onRecv . 2.1(B sends A over channel1) Atomic Actions
		At this point we want to assert:
		A: escrowA = amount,denom
		B: escrowB = amount,transfer/channel-0/denom

	*/

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()
	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[1].SenderAccount
	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path2.EndpointB.ChannelConfig.PortID,
				ChannelId: path2.EndpointB.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainA.Amount)

	// denom path: transfer/channel-0
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that Escrow B has amount
	coin = sdk.NewCoin(denom.IBCDenom(), amount)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)
}

// This test is probably overcomplicated. Could have used RecvPacketWithResult directly.
func (suite *KeeperTestSuite) TestHappyPathForwarding() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topolgy:

		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake

		We want to trigger:
		1. A sends B over channel0.
		2. B onRecv . 2.1(B sends A over channel1) Atomic Actions
		At this point we want to assert:
		A: escrowA = amount,denom
		B: escrowB = amount,transfer/channel-0/denom
		3. A OnRecv
		At this point we want to assert:
		C: finalReceiver = amount,transfer/channel-1/transfer/channel-0/denom
	*/

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// transfer/channel-1/transfer/channel-0/denom
	denomABA := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID), types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that initially the final receiver address has 0 ABA coins
	coin := sdk.NewCoin(denomABA.IBCDenom(), amount)
	preCoinOnA := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccounts[1].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), preCoinOnA.Amount, "final receiver has not zero balance")

	coin = sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[1].SenderAccount
	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path2.EndpointB.ChannelConfig.PortID,
				ChannelId: path2.EndpointB.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	denom := types.Denom{Base: sdk.DefaultBondDenom}
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, sender.GetAddress().String(), receiver.GetAddress().String(), "", &forwardingPath)
	packetRecv := channeltypes.NewPacket(data.GetBytes(), 2, path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)

	var async bool
	async, err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packetRecv, data)
	// If forwarding has been triggered then the async must be true.
	suite.Require().True(async)
	suite.Require().Nil(err)

	// denomTrace path: transfer/channel-0
	denom = types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that Escrow B has amount
	coin = sdk.NewCoin(denom.IBCDenom(), amount)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(amount, totalEscrowChainB.Amount, "escrow account on B is different than amount")

	// Check that Escrow A has amount
	coin = sdk.NewCoin(sdk.DefaultBondDenom, amount)
	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(amount, totalEscrowChainA.Amount, "escrow account on A is different than amount")

	// Now during the onRecvPacket above a new msgTransfer has been sent
	// We need to receive the packet on the final hand

	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	data = types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, types.GetForwardAddress(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID).String(), receiver.GetAddress().String(), "", nil)
	packetRecv = channeltypes.NewPacket(data.GetBytes(), 3, path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID, path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)

	// execute onRecvPacket, when chaninA receives the tokens the escrow amount on B should increase to amount
	async, err = suite.chainA.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainA.GetContext(), packetRecv, data)
	suite.Require().False(async)
	suite.Require().NoError(err)

	// Check that the final receiver has received the expected tokens.
	coin = sdk.NewCoin(denomABA.IBCDenom(), amount)
	postCoinOnA := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccounts[1].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnA.Amount, "final receiver balance has not increased")
}

// Simplification of the above test.
func (suite *KeeperTestSuite) TestSimplifiedHappyPathForwarding() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topolgy:

		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake

		We want to trigger:
		1. A sends B over channel0.
		2. B onRecv . 2.1(B sends A over channel1) Atomic Actions
		At this point we want to assert:
		A: escrowA = amount,denom
		B: escrowB = amount,transfer/channel-0/denom

	*/

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()
	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[1].SenderAccount
	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path2.EndpointB.ChannelConfig.PortID,
				ChannelId: path2.EndpointB.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow A has amount
	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainA.Amount)

	// denomTrace path: transfer/channel-0
	denomTrace := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that Escrow B has amount
	coin = sdk.NewCoin(denomTrace.IBCDenom(), amount)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path2.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointA.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// transfer/channel-1/transfer/channel-0/denom
	denomTraceABA := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID), types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that the final receiver has received the expected tokens.
	coin = sdk.NewCoin(denomTraceABA.IBCDenom(), amount)
	postCoinOnA := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccounts[1].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnA.Amount, "final receiver balance has not increased")
}

// This test replicates the Acknowledgement Failure Scenario 5
// Currently seems like the middle hop is not reverting state changes when an error occurs.
// In turn the final hop properly reverts changes. There may be an error in the way async ack are managed
// or in the way i'm trying to activate the OnAck function.
func (suite *KeeperTestSuite) TestAcknowledgementFailureScenario5Forwarding() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topolgy:

		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain C
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake

		We want to trigger:
		0. A sends B over channel0 [path1]
		1. B sends C over channel1 [path2].
		2. C recvs - This represent the checkpoint we will need to verify at the of the test
		3. C --> [path2] B --> [path1] A.
		4. OnRecv in B works properly and trigger the packet forwarding to A
		5. Modify the balance of escrowA to cause an error during the onRecv
		6. OnRecv on A fails. Error Ack is written in A, relayed to B and finally to C.
		At this point we want to assert:
		Everything has been reverted at checkpoint values.
		- C has amount of transfer/channel-1/transfer/channel-0/stake
		- B totalEscrow has amount of transfer/channel-0/stake
	*/

	// Testing Topology

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	path2.Setup()

	// First we want to execute 0.

	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainB.SenderAccounts[0].SenderAccount

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		nil,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow B has amount
	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainA.Amount)

	// transfer/channel-0/denom
	denomAB := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check the coins have been received on B
	coin = sdk.NewCoin(denomAB.IBCDenom(), amount)
	postCoinOnB := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnB.Amount, "final receiver balance has not increased")

	// A --> B Simple transfer happened properly.

	// Now we want to trigger B -> C
	sender = suite.chainB.SenderAccounts[0].SenderAccount
	receiver = suite.chainC.SenderAccounts[0].SenderAccount

	transferMsg = types.NewMsgTransfer(
		path2.EndpointA.ChannelConfig.PortID,
		path2.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		nil,
	)

	result, err = suite.chainB.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow B has amount
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	// transfer/channel-1/transfer/channel-0/denom
	denomABC := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID), types.NewTrace(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID))

	// Check the coins have been received on C
	coin = sdk.NewCoin(denomABC.IBCDenom(), amount)
	postCoinOnC := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnC.Amount, "final receiver balance has not increased")

	// B -> C Simple transfer happened properly.

	// Now we want to trigger C -> B -> A
	// The coin we want to send out is exactly the one we received on C
	// coin = sdk.NewCoin(denomTraceBC.IBCDenom(), amount)

	sender = suite.chainC.SenderAccounts[0].SenderAccount
	receiver = suite.chainA.SenderAccounts[0].SenderAccount // Receiver is the A chain account

	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path1.EndpointB.ChannelConfig.PortID,
				ChannelId: path1.EndpointB.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg = types.NewMsgTransfer(
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
	)

	result, err = suite.chainC.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// Voucher have been burned on chain C
	postCoinOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnC.Amount, "Vouchers have not been burned")

	// parse the packet from result events and recv packet on chainB
	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path2.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointA.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// We have successfully received the packet on B and forwarded it to A.
	// Lets try to retrieve it in order to save it
	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, packet.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packet, forwardedPacket)

	// Voucher have been burned on chain B
	coin = sdk.NewCoin(denomAB.IBCDenom(), amount)
	postCoinOnB = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnB.Amount, "Vouchers have not been burned")

	// Now we can receive the packet on A.
	// To trigger an error during the OnRecv, we have to manipulate the balance present in the escrow of A
	// of denom

	// parse the packet from result events and recv packet on chainA
	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	// manipulate escrow account for denom on chain A
	coin = sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(99))
	suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), coin)
	totalEscrowChainA = suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(99), totalEscrowChainA.Amount)

	err = path1.EndpointA.UpdateClient()
	suite.Require().NoError(err)
	// suite.Require().Equal(packet, forwardedPacket)

	result, err = path1.EndpointA.RecvPacketWithResult(packet)
	suite.Require().Error(err)
	suite.Require().Nil(result)
	// In theory now an error ack should have been written on chain A
	// NOW WE HAVE TO SEND ACK TO B, PROPAGTE ACK TO C, CHECK FINAL RESULTS

	// Reconstruct packet data
	denom := types.ExtractDenomFromPath(denomAB.Path())
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, types.GetForwardAddress(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID).String(), suite.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(), "", nil)
	packetRecv := channeltypes.NewPacket(data.GetBytes(), 3, path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)
	ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed packet transfer"))

	// err = path1.EndpointA.AcknowledgePacket(packetRecv, ack.Acknowledgement())
	err = suite.chainB.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainB.GetContext(), packetRecv, data, ack)
	suite.Require().NoError(err)

	// Check that Escrow B has been refunded amount
	// NOTE This is failing. The revertInFlightsChanges sohuld mint back voucher to chainBescrow
	// but this is not happening. It may be a problem related with how we're writing async acks.
	//
	coin = sdk.NewCoin(denomAB.IBCDenom(), amount)
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	denom = types.ExtractDenomFromPath(denomABC.Path())
	data = types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, suite.chainC.SenderAccounts[0].SenderAccount.GetAddress().String(), suite.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(), "", nil)
	// suite.chainC.SenderAccounts[0].SenderAccount.GetAddress().String() This should be forward account of B
	packet = channeltypes.NewPacket(data.GetBytes(), 3, path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID, path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)

	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// Check the status of account on chain C before executing ack.
	coin = sdk.NewCoin(denomABC.IBCDenom(), amount)
	postCoinOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnC.Amount, "Final Hop balance has been refunded before Ack execution")

	// Execute ack
	err = suite.chainC.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainC.GetContext(), packet, data, ack)
	// err = path2.EndpointB.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)

	// Check that everythig has been reverted
	//
	// Check the vouchers transfer/channel-1/transfer/channel-0/denom have been refunded on C
	coin = sdk.NewCoin(denomABC.IBCDenom(), amount)
	postCoinOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnC.Amount, "final receiver balance has not increased")

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)
}

/*
// TODO

	Test scenarios for failures ack

Check out the notion page: https://www.notion.so/interchain/ICS20-v2-path-forwarding-091f1ac788e84a538261c5a247cb5924

// TODO
Test async ack is properly relayed to middle hop after forwarding transfer completition

// TODO
Tiemout during forwarding after middle hop execution reverts properly the state changes
*/
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
			"success: no new field with memo v2",
			func() {
				jsonString := fmt.Sprintf(`{"tokens":[{"denom": {"base": "atom", "trace": []},"amount":"100"}],"sender":"%s","receiver":"%s"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
			nil,
		},
		{
			"success: no new field without memo",
			func() {
				jsonString := fmt.Sprintf(`{"tokens":[{"denom": {"base": "atom", "trace": []},"amount":"100"}],"sender":"%s","receiver":"%s"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
			nil,
		},
		{
			"failure: invalid packet data v2",
			func() {
				packetData = []byte("invalid packet data")
			},
			ibcerrors.ErrInvalidType,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: new field v2",
			func() {
				jsonString := fmt.Sprintf(`{"tokens":[{"denom": {"base": "atom", "trace": []},"amount":"100"}],"sender":"%s","receiver":"%s", "new_field":"value"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			ibcerrors.ErrInvalidType,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: missing field v2",
			func() {
				jsonString := fmt.Sprintf(`{"tokens":[{"denom": {"trace": []},"amount":"100"}],"sender":"%s","receiver":"%s"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			types.ErrInvalidDenomForTransfer,
			ibcerrors.ErrInvalidType,
		},
		{
			"success: no new field with memo",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V1
				path.EndpointB.ChannelConfig.Version = types.V1
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s","memo":"memo"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
			nil,
		},
		{
			"success: no new field without memo",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V1
				path.EndpointB.ChannelConfig.Version = types.V1
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			nil,
			nil,
		},
		{
			"failure: invalid packet data",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V1
				path.EndpointB.ChannelConfig.Version = types.V1
				packetData = []byte("invalid packet data")
			},
			ibcerrors.ErrInvalidType,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: new field",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V1
				path.EndpointB.ChannelConfig.Version = types.V1
				jsonString := fmt.Sprintf(`{"denom":"denom","amount":"100","sender":"%s","receiver":"%s","memo":"memo","new_field":"value"}`, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String())
				packetData = []byte(jsonString)
			},
			ibcerrors.ErrInvalidType,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: missing field",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V1
				path.EndpointB.ChannelConfig.Version = types.V1
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
		bz     []byte
		tokens types.Tokens
	)

	testCases := []struct {
		name        string
		appVersion  string
		malleate    func()
		expResult   func(bz []byte)
		expPanicErr error
	}{
		{
			"success",
			types.V1,
			func() {},
			func(bz []byte) {
				expPacketData := types.NewFungibleTokenPacketData("", "", "", "", "")
				suite.Require().Equal(bz, expPacketData.GetBytes())
			},
			nil,
		},
		{
			"success: version 2",
			types.V2,
			func() {},
			func(bz []byte) {
				expPacketData := types.NewFungibleTokenPacketDataV2(types.Tokens{types.Token{}}, "", "", "", nil)
				suite.Require().Equal(bz, expPacketData.GetBytes())
			},
			nil,
		},
		{
			"failure: must have single coin if using version 1.",
			types.V1,
			func() {
				tokens = types.Tokens{}
			},
			nil,
			fmt.Errorf("length of tokens must be equal to 1 if using %s version", types.V1),
		},
		{
			"failure: invalid version",
			ibcmock.Version,
			func() {},
			nil,
			fmt.Errorf("app version must be one of %s", types.SupportedVersions),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tokens = types.Tokens{types.Token{}}

			tc.malleate()

			createFunc := func() {
				bz = transferkeeper.CreatePacketDataBytesFromVersion(tc.appVersion, "", "", "", tokens, nil)
			}

			expPanic := tc.expPanicErr != nil
			if expPanic {
				suite.Require().PanicsWithError(tc.expPanicErr.Error(), createFunc)
			} else {
				createFunc()
				tc.expResult(bz)
			}
		})
	}
}
