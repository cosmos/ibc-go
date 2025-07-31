package v2_test

import (
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v11"
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
func (s *TransferTestSuite) TestAliasedTransferChannel() {
	path := ibctesting.NewTransferPath(s.chainA, s.chainB)
	path.Setup()

	// mock v1 format for both sides of the channel
	s.mockV1Format(path.EndpointA)
	s.mockV1Format(path.EndpointB)

	// migrate the store for both chains
	err := v11.MigrateStore(s.chainA.GetContext(), runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), s.chainA.App.AppCodec(), s.chainA.App.GetIBCKeeper())
	s.Require().NoError(err)
	err = v11.MigrateStore(s.chainB.GetContext(), runtime.NewKVStoreService(s.chainB.GetSimApp().GetKey(ibcexported.StoreKey)), s.chainB.App.AppCodec(), s.chainB.App.GetIBCKeeper())
	s.Require().NoError(err)

	// create v2 path from the original client ids
	// the path config is only used for updating
	// the packet client ids will be the original channel identifiers
	// but they are not validated against the client ids in the path in the tests
	pathv2 := ibctesting.NewPath(s.chainA, s.chainB)
	pathv2.EndpointA.ClientID = path.EndpointA.ClientID
	pathv2.EndpointB.ClientID = path.EndpointB.ClientID

	// save original amount that sender has in its balance
	originalAmount := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom).Amount

	// send v1 packet with default values
	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	transferMsg := types.NewMsgTransfer(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		ibctesting.TestCoin, sender.String(), receiver.String(),
		s.chainB.GetTimeoutHeight(), 0, "",
	)

	result, err := s.chainA.SendMsgs(transferMsg)
	s.Require().NoError(err) // message committed

	packet, err := ibctesting.ParseV1PacketFromEvents(result.Events)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), packet.Sequence, "sequence should be 1 for first packet")

	err = path.RelayPacket(packet)
	s.Require().NoError(err)

	// check that the escrow and receiver amounts are correct
	// after first packet
	s.assertEscrowEqual(s.chainA, ibctesting.TestCoin, ibctesting.DefaultCoinAmount)
	ibcDenom := types.NewDenom(
		ibctesting.TestCoin.Denom,
		types.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID),
	)
	s.assertReceiverEqual(s.chainB, ibcDenom.IBCDenom(), receiver, ibctesting.DefaultCoinAmount)

	// v2 packets only support timeout timestamps in UNIX time.
	timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

	// send v2 packet on aliased channel
	msgTransferAlias := types.NewMsgTransferAliased(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		ibctesting.TestCoin, sender.String(), receiver.String(),
		clienttypes.Height{}, timeoutTimestamp, "",
	)
	res, err := path.EndpointA.Chain.SendMsgs(msgTransferAlias)
	s.Require().NoError(err, "send v2 packet failed")

	packetv2, err := ibctesting.ParseV2PacketFromEvents(res.Events)
	s.Require().NoError(err, "parse v2 packet from events failed")
	s.Require().Equal(uint64(2), packetv2.Sequence, "sequence should be incremented across protocol versions")

	err = path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	// relay v2 packet
	err = pathv2.EndpointA.RelayPacket(packetv2)
	s.Require().NoError(err)

	// check that the escrow and receiver amounts are correct
	// after first packet
	// this should be double the default amount since we sent the same amount twice
	// once with IBC v1 and once with IBC v2
	newAmount := ibctesting.DefaultCoinAmount.MulRaw(2)
	s.assertEscrowEqual(s.chainA, ibctesting.TestCoin, newAmount)
	s.assertReceiverEqual(s.chainB, ibcDenom.IBCDenom(), receiver, newAmount)

	// send all the tokens back using IBC v2
	// NOTE: Creating a reversed path to use helper functions
	// sender and receiver are swapped
	revPath := ibctesting.NewPath(s.chainB, s.chainA)
	revPath.EndpointA.ClientID = path.EndpointB.ClientID
	revPath.EndpointB.ClientID = path.EndpointA.ClientID

	revToken := types.Token{
		Denom: types.Denom{
			Trace: []types.Hop{
				{
					PortId:    path.EndpointB.ChannelConfig.PortID,
					ChannelId: path.EndpointB.ChannelID,
				},
			},
			Base: ibctesting.TestCoin.Denom,
		},
		Amount: ibctesting.TestCoin.Amount.MulRaw(2).String(),
	}
	revCoin, err := revToken.ToCoin()
	s.Require().NoError(err, "convert token to coin failed")

	revTimeoutTimestamp := uint64(s.chainA.GetContext().BlockTime().Add(time.Hour).Unix())

	// send v2 packet
	// using encoding here just to use both message constructor functions
	msgTransferRev := types.NewMsgTransferWithEncoding(
		path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
		revCoin, receiver.String(), sender.String(),
		clienttypes.Height{}, revTimeoutTimestamp, "", "application/json", true,
	)
	res, err = revPath.EndpointA.Chain.SendMsgs(msgTransferRev)
	s.Require().NoError(err, "send v2 packet failed")

	packetv2, err = ibctesting.ParseV2PacketFromEvents(res.Events)
	s.Require().NoError(err, "parse v2 packet from events failed")
	s.Require().Equal(uint64(1), packetv2.Sequence, "sequence should be 1 on the counterparty chain")

	err = revPath.EndpointB.UpdateClient()
	s.Require().NoError(err)

	// relay v2 packet
	err = revPath.EndpointA.RelayPacket(packetv2)
	s.Require().NoError(err)

	// check that the balances are back to their original state
	// after the reverse packet is sent with the full amount
	s.assertEscrowEqual(s.chainA, ibctesting.TestCoin, sdkmath.ZeroInt())
	s.assertReceiverEqual(s.chainA, ibctesting.TestCoin.Denom, sender, originalAmount)
	s.assertReceiverEqual(s.chainB, ibcDenom.IBCDenom(), receiver, sdkmath.ZeroInt())
}

// This test ensures we can send a different application on the same channel identifier
// and that the sequences are still incremented correctly as a global app agnostic sequence.
func (s *TransferTestSuite) TestDifferentAppPostAlias() {
	path := ibctesting.NewTransferPath(s.chainA, s.chainB)
	path.Setup()

	// mock v1 format for both sides of the channel
	s.mockV1Format(path.EndpointA)
	s.mockV1Format(path.EndpointB)

	// migrate the store for both chains
	err := v11.MigrateStore(s.chainA.GetContext(), runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), s.chainA.App.AppCodec(), s.chainA.App.GetIBCKeeper())
	s.Require().NoError(err)
	err = v11.MigrateStore(s.chainB.GetContext(), runtime.NewKVStoreService(s.chainB.GetSimApp().GetKey(ibcexported.StoreKey)), s.chainB.App.AppCodec(), s.chainB.App.GetIBCKeeper())
	s.Require().NoError(err)

	// create v2 path from the original client ids
	// the path config is only used for updating
	// the packet client ids will be the original channel identifiers
	// but they are not validated against the client ids in the path in the tests
	pathv2 := ibctesting.NewPath(s.chainA, s.chainB)
	pathv2.EndpointA.ClientID = path.EndpointA.ClientID
	pathv2.EndpointB.ClientID = path.EndpointB.ClientID

	// create default packet with a timed out timestamp
	mockPayload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

	timeoutTimestamp := uint64(s.chainA.GetContext().BlockTime().Add(time.Hour).Unix())

	// send v2 packet with mock payload
	// over a v1 transfer channel's channel identifier
	msgSendPacket := channeltypesv2.NewMsgSendPacket(
		path.EndpointA.ChannelID,
		timeoutTimestamp,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
		mockPayload,
	)
	res, err := path.EndpointA.Chain.SendMsgs(msgSendPacket)
	s.Require().NoError(err, "send v2 packet failed")

	packetv2, err := ibctesting.ParseV2PacketFromEvents(res.Events)
	s.Require().NoError(err, "parse v2 packet from events failed")
	s.Require().Equal(uint64(1), packetv2.Sequence, "sequence should be 1 for first packet")

	err = path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	// relay v2 packet
	err = pathv2.EndpointA.RelayPacket(packetv2)
	s.Require().NoError(err)

	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()

	// now send a transfer v2 packet
	msgTransferAlias := types.NewMsgTransferAliased(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		ibctesting.TestCoin, sender.String(), receiver.String(),
		clienttypes.Height{}, timeoutTimestamp, "",
	)
	res, err = path.EndpointA.Chain.SendMsgs(msgTransferAlias)
	s.Require().NoError(err, "send v2 packet failed")

	transferv2Packet, err := ibctesting.ParseV2PacketFromEvents(res.Events)
	s.Require().NoError(err, "parse v2 packet from events failed")
	s.Require().Equal(uint64(2), transferv2Packet.Sequence, "sequence should be incremented across applications")

	err = path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	// now send a transfer v1 packet
	transferMsg := types.NewMsgTransfer(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		ibctesting.TestCoin, sender.String(), receiver.String(),
		s.chainB.GetTimeoutHeight(), 0, "",
	)

	result, err := s.chainA.SendMsgs(transferMsg)
	s.Require().NoError(err) // message committed

	transferv1Packet, err := ibctesting.ParseV1PacketFromEvents(result.Events)
	s.Require().NoError(err)

	err = path.RelayPacket(transferv1Packet)
	s.Require().NoError(err)
	s.Require().Equal(uint64(3), transferv1Packet.Sequence, "sequence should be incremented across protocol versions")
}

// assertEscrowEqual asserts that the amounts escrowed for each of the coins on chain matches the expectedAmounts
func (s *TransferTestSuite) assertEscrowEqual(chain *ibctesting.TestChain, coin sdk.Coin, expectedAmount sdkmath.Int) {
	amount := chain.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(chain.GetContext(), coin.GetDenom())
	s.Require().Equal(expectedAmount, amount.Amount)
}

// assertReceiverEqual asserts that the amounts received by the receiver account matches the expectedAmounts
func (s *TransferTestSuite) assertReceiverEqual(chain *ibctesting.TestChain, denom string, receiver sdk.AccAddress, expectedAmount sdkmath.Int) {
	amount := chain.GetSimApp().BankKeeper.GetBalance(chain.GetContext(), receiver, denom)
	s.Require().Equal(expectedAmount, amount.Amount, "receiver balance should match expected amount")
}

func (s *TransferTestSuite) mockV1Format(endpoint *ibctesting.Endpoint) {
	// mock v1 format by setting the sequence in the old key
	seq, ok := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	s.Require().True(ok, "should be able to get next sequence send for channel")

	// move the next sequence send back to the old v1 format key
	// so we can migrate it in our tests
	storeService := runtime.NewKVStoreService(endpoint.Chain.GetSimApp().GetKey(ibcexported.StoreKey))
	store := storeService.OpenKVStore(endpoint.Chain.GetContext())
	err := store.Set(v11.NextSequenceSendV1Key(endpoint.ChannelConfig.PortID, endpoint.ChannelID), sdk.Uint64ToBigEndian(seq))
	s.Require().NoError(err)
	err = store.Delete(hostv2.NextSequenceSendKey(endpoint.ChannelID))
	s.Require().NoError(err)
}
