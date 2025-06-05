package v2_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	v11 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v11"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *TransferTestSuite) TestAliasedTransferChannel() {
	path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path.Setup()

	// mock v1 format for both sides of the channel
	mockV1Format(path.EndpointA)
	mockV1Format(path.EndpointB)

	// migrate the store for both chains
	v11.MigrateStore(suite.chainA.GetContext(), runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), suite.chainA.App.AppCodec(), suite.chainA.App.GetIBCKeeper())
	v11.MigrateStore(suite.chainB.GetContext(), runtime.NewKVStoreService(suite.chainB.GetSimApp().GetKey(ibcexported.StoreKey)), suite.chainB.App.AppCodec(), suite.chainB.App.GetIBCKeeper())

	// create v2 path from the original client ids
	// the path config is only used for updating
	// the packet client ids will be the original channel identifiers
	// but they are not validated against the client ids in the path in the tests
	pathv2 := ibctesting.NewPath(suite.chainA, suite.chainB)
	pathv2.EndpointA.ClientID = path.EndpointA.ClientID
	pathv2.EndpointB.ClientID = path.EndpointB.ClientID

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

	// create v2 packet with default values on same channel id
	token := types.Token{
		Denom:  types.Denom{Base: ibctesting.TestCoin.Denom},
		Amount: ibctesting.TestCoin.Amount.String(),
	}

	transferData := types.NewFungibleTokenPacketData(token.Denom.Path(), token.Amount, sender.String(), receiver.String(), "")
	bz := suite.chainA.Codec.MustMarshal(&transferData)
	payload := channeltypesv2.NewPayload(types.PortID, types.PortID, types.V1, types.EncodingProtobuf, bz)

	timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Hour).Unix())

	// send v2 packet
	msgSendPacket := channeltypesv2.NewMsgSendPacket(
		path.EndpointA.ChannelID,
		timeoutTimestamp,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
		payload,
	)
	res, err := path.EndpointA.Chain.SendMsgs(msgSendPacket)
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
	store.Set(v11.NextSequenceSendKey(endpoint.ChannelConfig.PortID, endpoint.ChannelID), sdk.Uint64ToBigEndian(seq))
	store.Delete(hostv2.NextSequenceSendKey(endpoint.ChannelID))
}
