package ibc_test

import (
	"fmt"
	"time"

	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// If packet receipts are pruned, it is possible to double spend
// by resubmitting the same proof used to process the original receive.
func (suite *IBCTestSuite) TestDoubleSpendAttackOnPrunedReceipts() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)

	// configure the initial path to create a regular transfer channel
	path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointA.ChannelConfig.Version = transfertypes.Version
	path.EndpointB.ChannelConfig.Version = transfertypes.Version

	suite.coordinator.Setup(path)

	// configure the channel upgrade to upgrade to an incentivized fee enabled transfer channel
	upgradeVersion := string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: transfertypes.Version}))
	path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = upgradeVersion
	path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = upgradeVersion

	// setup double spend attack
	amount, ok := sdkmath.NewIntFromString("1000")
	suite.Require().True(ok)
	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)

	// send from chainA to chainB
	timeoutHeight := clienttypes.NewHeight(1, 110)
	msg := transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coin, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	suite.Require().NoError(err)

	// relay
	err = path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// get proof of packet commitment on source
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := path.EndpointA.Chain.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, path.EndpointB.Chain.SenderAccount.GetAddress().String())

	// receive on counterparty and update source client
	res, err = path.EndpointB.Chain.SendMsgs(recvMsg)
	suite.Require().NoError(err)

	// check that voucher exists on chain B
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))
	ibcCoin := sdk.NewCoin(voucherDenomTrace.IBCDenom(), coin.Amount)
	balance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	suite.Require().Equal(ibcCoin, balance)

	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	ack, err := ibctesting.ParseAckFromEvents(res.Events)
	suite.Require().NoError(err)

	err = path.EndpointA.AcknowledgePacket(packet, ack)
	suite.Require().NoError(err)

	err = path.EndpointA.ChanUpgradeInit()
	suite.Require().NoError(err)

	err = path.EndpointB.ChanUpgradeTry()
	suite.Require().NoError(err)

	err = path.EndpointA.ChanUpgradeAck()
	suite.Require().NoError(err)

	err = path.EndpointB.ChanUpgradeConfirm()
	suite.Require().NoError(err)

	err = path.EndpointA.ChanUpgradeOpen()
	suite.Require().NoError(err)

	// prune
	msgPrune := channeltypes.NewMsgPruneAcknowledgements(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, 1, path.EndpointB.Chain.SenderAccount.GetAddress().String())
	res, err = path.EndpointB.Chain.SendMsgs(msgPrune)
	suite.Require().NoError(err)

	recvMsg = channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, path.EndpointB.Chain.SenderAccount.GetAddress().String())

	// double spend
	res, err = path.EndpointB.Chain.SendMsgs(recvMsg)
	suite.Require().NoError(err)

	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	suite.Require().Equal(ibcCoin, balance, "successfully double spent")
}

func (suite *IBCTestSuite) TestDoubleSpendAttackOnOrderedToUnorderedUpgrade() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)

	testVersion := string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: ibctesting.FirstConnectionID,
		HostConnectionId:       ibctesting.FirstConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}))

	// configure the initial path to create a regular ica channel
	path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointA.ChannelConfig.Version = testVersion
	path.EndpointB.ChannelConfig.Version = testVersion

	suite.coordinator.SetupConnections(path)
	owner := suite.chainA.SenderAccount.GetAddress().String()
	err := SetupICAPath(path, owner)
	suite.Require().NoError(err)

	// send packet
	portID, err := icatypes.NewControllerPortID(owner)
	suite.Require().NoError(err)

	// get the address of the interchain account stored in state during handshake step
	interchainAccountAddr, found := suite.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(suite.chainA.GetContext(), path.EndpointA.ConnectionID, portID)
	suite.Require().True(found)

	// fund ica wallet
	amount := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)))
	msgBankSend := &banktypes.MsgSend{
		FromAddress: suite.chainB.SenderAccount.GetAddress().String(),
		ToAddress:   interchainAccountAddr,
		Amount:      amount,
	}

	res, err := suite.chainB.SendMsgs(msgBankSend)
	suite.Require().NotEmpty(res)
	suite.Require().NoError(err)

	// create bank transfer message that will execute on the host chain
	icaMsg := &banktypes.MsgSend{
		FromAddress: interchainAccountAddr,
		ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
		Amount:      amount,
	}

	data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{icaMsg}, icatypes.EncodingProtobuf)
	suite.Require().NoError(err)

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
		Memo: "memo",
	}

	timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Hour).UnixNano())
	connectionID := path.EndpointA.ConnectionID

	msg := icacontrollertypes.NewMsgSendTx(owner, connectionID, timeoutTimestamp, packetData)
	res, err = path.EndpointA.Chain.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	suite.Require().NoError(err)

	// relay
	err = path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// get proof of packet commitment on source
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := path.EndpointA.Chain.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, path.EndpointB.Chain.SenderAccount.GetAddress().String())

	// receive on counterparty and update source client
	res, err = path.EndpointB.Chain.SendMsgs(recvMsg)
	suite.Require().NoError(err)

	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	ack, err := ibctesting.ParseAckFromEvents(res.Events)
	suite.Require().NoError(err)

	err = path.EndpointA.AcknowledgePacket(packet, ack)
	suite.Require().NoError(err)

	// configure the channel upgrade to upgrade to an UNORDERED channel
	path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Ordering = channeltypes.UNORDERED
	path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Ordering = channeltypes.UNORDERED

	err = path.EndpointA.ChanUpgradeInit()
	suite.Require().NoError(err)

	err = path.EndpointB.ChanUpgradeTry()
	suite.Require().NoError(err)

	err = path.EndpointA.ChanUpgradeAck()
	suite.Require().NoError(err)

	err = path.EndpointB.ChanUpgradeConfirm()
	suite.Require().NoError(err)

	err = path.EndpointA.ChanUpgradeOpen()
	suite.Require().NoError(err)

	// prune
	msgPrune := channeltypes.NewMsgPruneAcknowledgements(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, 1, path.EndpointB.Chain.SenderAccount.GetAddress().String())
	res, err = path.EndpointB.Chain.SendMsgs(msgPrune)
	suite.Require().NoError(err)

	recvMsg = channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, path.EndpointB.Chain.SenderAccount.GetAddress().String())

	// double spend
	res, err = path.EndpointB.Chain.SendMsgs(recvMsg)
	fmt.Println(res)
	suite.Require().Error(err, "attempts to send from empty balance")
}

func RegisterInterchainAccount(endpoint *ibctesting.Endpoint, owner string) error {
	portID, err := icatypes.NewControllerPortID(owner)
	if err != nil {
		return err
	}

	channelSequence := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(endpoint.Chain.GetContext())

	if err := endpoint.Chain.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, owner, endpoint.ChannelConfig.Version); err != nil {
		return err
	}

	// commit state changes for proof verification
	endpoint.Chain.NextBlock()

	// update port/channel ids
	endpoint.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
	endpoint.ChannelConfig.PortID = portID

	return nil
}

// SetupICAPath invokes the InterchainAccounts entrypoint and subsequent channel handshake handlers
func SetupICAPath(path *ibctesting.Path, owner string) error {
	if err := RegisterInterchainAccount(path.EndpointA, owner); err != nil {
		return err
	}

	if err := path.EndpointB.ChanOpenTry(); err != nil {
		return err
	}

	if err := path.EndpointA.ChanOpenAck(); err != nil {
		return err
	}

	return path.EndpointB.ChanOpenConfirm()
}
