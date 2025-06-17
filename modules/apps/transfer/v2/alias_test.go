package v2_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	v11 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v11"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

// This test migrates a V1 channel and then does the following:
// It will send a transfer packet using the V1 format,
// then it will send a transfer packet using the V2 format on the same channel.
// It will then send a transfer packet back using the V2 format on the same channel.
// It checks that the escrow and receiver amounts are correct after each packet is sent.
func (suite *TransferTestSuite) TestAliasedTransferChannel() {
	path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path.Setup()

	// mock v1 format for both sides of the channel
	mockV1Format(path.EndpointA)
	mockV1Format(path.EndpointB)

	// migrate the store for both chains
	err := v11.MigrateStore(suite.chainA.GetContext(), runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), suite.chainA.App.AppCodec(), suite.chainA.App.GetIBCKeeper())
	suite.Require().NoError(err)
	err = v11.MigrateStore(suite.chainB.GetContext(), runtime.NewKVStoreService(suite.chainB.GetSimApp().GetKey(ibcexported.StoreKey)), suite.chainB.App.AppCodec(), suite.chainB.App.GetIBCKeeper())
	suite.Require().NoError(err)

	// create v2 path from the original client ids
	// the path config is only used for updating
	// the packet client ids will be the original channel identifiers
	// but they are not validated against the client ids in the path in the tests
	pathv2 := ibctesting.NewPath(suite.chainA, suite.chainB)
	pathv2.EndpointA.ClientID = path.EndpointA.ClientID
	pathv2.EndpointB.ClientID = path.EndpointB.ClientID

	// save original amount that sender has in its balance
	originalAmount := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom).Amount

	// send v1 packet with default values
	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()
	transferMsg := types.NewMsgTransfer(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		ibctesting.TestCoin, sender.String(), receiver.String(),
		suite.chainB.GetTimeoutHeight(), 0, "",
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParseV1PacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), packet.Sequence, "sequence should be 1 for first packet")

	err = path.RelayPacket(packet)
	suite.Require().NoError(err)

	// check that the escrow and receiver amounts are correct
	// after first packet
	suite.assertEscrowEqual(suite.chainA, ibctesting.TestCoin, ibctesting.DefaultCoinAmount)
	ibcDenom := types.NewDenom(
		ibctesting.TestCoin.Denom,
		types.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID),
	)
	suite.assertReceiverEqual(suite.chainB, ibcDenom.IBCDenom(), receiver, ibctesting.DefaultCoinAmount)

	// v2 packets only support timeout timestamps in UNIX time.
	timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

	// send v2 packet on aliased channel
	msgTransferAlias := types.NewMsgTransferAliased(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		ibctesting.TestCoin, sender.String(), receiver.String(),
		clienttypes.Height{}, timeoutTimestamp, "",
	)
	res, err := path.EndpointA.Chain.SendMsgs(msgTransferAlias)
	suite.Require().NoError(err, "send v2 packet failed")

	packetv2, err := ibctesting.ParseV2PacketFromEvents(res.Events)
	suite.Require().NoError(err, "parse v2 packet from events failed")
	suite.Require().Equal(uint64(2), packetv2.Sequence, "sequence should be incremented across protocol versions")

	err = path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// relay v2 packet
	err = pathv2.EndpointA.RelayPacket(packetv2)
	suite.Require().NoError(err)

	// check that the escrow and receiver amounts are correct
	// after first packet
	// this should be double the default amount since we sent the same amount twice
	// once with IBC v1 and once with IBC v2
	newAmount := ibctesting.DefaultCoinAmount.MulRaw(2)
	suite.assertEscrowEqual(suite.chainA, ibctesting.TestCoin, newAmount)
	suite.assertReceiverEqual(suite.chainB, ibcDenom.IBCDenom(), receiver, newAmount)

	// send all the tokens back using IBC v2
	// NOTE: Creating a reversed path to use helper functions
	// sender and receiver are swapped
	revPath := ibctesting.NewPath(suite.chainB, suite.chainA)
	revPath.EndpointA.ClientID = path.EndpointB.ClientID
	revPath.EndpointB.ClientID = path.EndpointA.ClientID

	revToken := types.Token{
		Denom: types.Denom{
			Trace: []types.Hop{
				types.Hop{
					PortId:    path.EndpointB.ChannelConfig.PortID,
					ChannelId: path.EndpointB.ChannelID,
				},
			},
			Base: ibctesting.TestCoin.Denom},
		Amount: ibctesting.TestCoin.Amount.MulRaw(2).String(),
	}
	revCoin, err := revToken.ToCoin()
	suite.Require().NoError(err, "convert token to coin failed")

	revTimeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Hour).Unix())

	// send v2 packet
	// using encoding here just to use both message constructor functions
	msgTransferRev := types.NewMsgTransferWithEncoding(
		path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
		revCoin, receiver.String(), sender.String(),
		clienttypes.Height{}, revTimeoutTimestamp, "", "application/json", true,
	)
	res, err = revPath.EndpointA.Chain.SendMsgs(msgTransferRev)
	suite.Require().NoError(err, "send v2 packet failed")

	packetv2, err = ibctesting.ParseV2PacketFromEvents(res.Events)
	suite.Require().NoError(err, "parse v2 packet from events failed")
	suite.Require().Equal(uint64(1), packetv2.Sequence, "sequence should be 1 on the counterparty chain")

	err = revPath.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// relay v2 packet
	err = revPath.EndpointA.RelayPacket(packetv2)
	suite.Require().NoError(err)

	// check that the balances are back to their original state
	// after the reverse packet is sent with the full amount
	suite.assertEscrowEqual(suite.chainA, ibctesting.TestCoin, sdkmath.ZeroInt())
	suite.assertReceiverEqual(suite.chainA, ibctesting.TestCoin.Denom, sender, originalAmount)
	suite.assertReceiverEqual(suite.chainB, ibcDenom.IBCDenom(), receiver, sdkmath.ZeroInt())
}

// This test ensures we can send a different application on the same channel identifier
// and that the sequences are still incremented correctly as a global app agnostic sequence.
func (suite *TransferTestSuite) TestDifferentAppPostAlias() {
	path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path.Setup()

	// mock v1 format for both sides of the channel
	mockV1Format(path.EndpointA)
	mockV1Format(path.EndpointB)

	// migrate the store for both chains
	err := v11.MigrateStore(suite.chainA.GetContext(), runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), suite.chainA.App.AppCodec(), suite.chainA.App.GetIBCKeeper())
	suite.Require().NoError(err)
	err = v11.MigrateStore(suite.chainB.GetContext(), runtime.NewKVStoreService(suite.chainB.GetSimApp().GetKey(ibcexported.StoreKey)), suite.chainB.App.AppCodec(), suite.chainB.App.GetIBCKeeper())
	suite.Require().NoError(err)

	// create v2 path from the original client ids
	// the path config is only used for updating
	// the packet client ids will be the original channel identifiers
	// but they are not validated against the client ids in the path in the tests
	pathv2 := ibctesting.NewPath(suite.chainA, suite.chainB)
	pathv2.EndpointA.ClientID = path.EndpointA.ClientID
	pathv2.EndpointB.ClientID = path.EndpointB.ClientID

	// create default packet with a timed out timestamp
	mockPayload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

	timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Hour).Unix())

	// send v2 packet with mock payload
	// over a v1 transfer channel's channel identifier
	msgSendPacket := channeltypesv2.NewMsgSendPacket(
		path.EndpointA.ChannelID,
		timeoutTimestamp,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
		mockPayload,
	)
	res, err := path.EndpointA.Chain.SendMsgs(msgSendPacket)
	suite.Require().NoError(err, "send v2 packet failed")

	packetv2, err := ibctesting.ParseV2PacketFromEvents(res.Events)
	suite.Require().NoError(err, "parse v2 packet from events failed")
	suite.Require().Equal(uint64(1), packetv2.Sequence, "sequence should be 1 for first packet")

	err = path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// relay v2 packet
	err = pathv2.EndpointA.RelayPacket(packetv2)
	suite.Require().NoError(err)

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()

	// now send a transfer v2 packet
	msgTransferAlias := types.NewMsgTransferAliased(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		ibctesting.TestCoin, sender.String(), receiver.String(),
		clienttypes.Height{}, timeoutTimestamp, "",
	)
	res, err = path.EndpointA.Chain.SendMsgs(msgTransferAlias)
	suite.Require().NoError(err, "send v2 packet failed")

	transferv2Packet, err := ibctesting.ParseV2PacketFromEvents(res.Events)
	suite.Require().NoError(err, "parse v2 packet from events failed")
	suite.Require().Equal(uint64(2), transferv2Packet.Sequence, "sequence should be incremented across applications")

	err = path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// now send a transfer v1 packet
	transferMsg := types.NewMsgTransfer(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		ibctesting.TestCoin, sender.String(), receiver.String(),
		suite.chainB.GetTimeoutHeight(), 0, "",
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	transferv1Packet, err := ibctesting.ParseV1PacketFromEvents(result.Events)
	suite.Require().NoError(err)

	err = path.RelayPacket(transferv1Packet)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(3), transferv1Packet.Sequence, "sequence should be incremented across protocol versions")

}

// assertEscrowEqual asserts that the amounts escrowed for each of the coins on chain matches the expectedAmounts
func (suite *TransferTestSuite) assertEscrowEqual(chain *ibctesting.TestChain, coin sdk.Coin, expectedAmount sdkmath.Int) {
	amount := chain.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(chain.GetContext(), coin.GetDenom())
	suite.Require().Equal(expectedAmount, amount.Amount)
}

// assertReceiverEqual asserts that the amounts received by the receiver account matches the expectedAmounts
func (suite *TransferTestSuite) assertReceiverEqual(chain *ibctesting.TestChain, denom string, receiver sdk.AccAddress, expectedAmount sdkmath.Int) {
	amount := chain.GetSimApp().BankKeeper.GetBalance(chain.GetContext(), receiver, denom)
	suite.Require().Equal(expectedAmount, amount.Amount, "receiver balance should match expected amount")
}

func mockV1Format(endpoint *ibctesting.Endpoint) {
	// mock v1 format by setting the sequence in the old key
	seq, ok := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	if !ok {
		panic("sequence not found")
	}

	// move the next sequence send back to the old v1 format key
	// so we can migrate it in our tests
	storeService := runtime.NewKVStoreService(endpoint.Chain.GetSimApp().GetKey(ibcexported.StoreKey))
	store := storeService.OpenKVStore(endpoint.Chain.GetContext())
	store.Set(v11.NextSequenceSendV1Key(endpoint.ChannelConfig.PortID, endpoint.ChannelID), sdk.Uint64ToBigEndian(seq))
	store.Delete(hostv2.NextSequenceSendKey(endpoint.ChannelID))
}
