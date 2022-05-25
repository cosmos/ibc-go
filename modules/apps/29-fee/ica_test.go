package fee_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	icahosttypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

var (
	// DefaultOwnerAddress defines a reusable bech32 address for testing purposes
	DefaultOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// DefaultPortID defines a resuable port identifier for testing purposes
	DefaultPortID, _ = icatypes.NewControllerPortID(DefaultOwnerAddress)

	// DefaultICAVersion defines a resuable interchainaccounts version string for testing purposes
	DefaultICAVersion = string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: ibctesting.FirstConnectionID,
		HostConnectionId:       ibctesting.FirstConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}))
)

// NewICAPath creates and returns a new ibctesting path configured for a fee enabled interchain accounts channel
func NewICAPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)

	feeICAVersion := string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: DefaultICAVersion}))

	path.SetChannelOrdered()
	path.EndpointA.ChannelConfig.Version = feeICAVersion
	path.EndpointB.ChannelConfig.Version = feeICAVersion
	path.EndpointA.ChannelConfig.PortID = DefaultPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.PortID

	return path
}

// SetupICAPath performs the InterchainAccounts channel creation handshake using an ibctesting path
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

	if err := path.EndpointB.ChanOpenConfirm(); err != nil {
		return err
	}

	return nil
}

// RegisterInterchainAccount invokes the the InterchainAccounts entrypoint, commits state changes and updates the testing endpoint accordingly
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

// TestFeeInterchainAccounts Integration test to ensure ics29 works with ics27
func (suite *FeeTestSuite) TestFeeInterchainAccounts() {
	path := NewICAPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := SetupICAPath(path, DefaultOwnerAddress)
	suite.Require().NoError(err)

	// assert the newly established channel is fee enabled on both ends
	suite.Require().True(suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
	suite.Require().True(suite.chainB.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))

	// escrow a packet fee for the next send sequence
	expectedFee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	msgPayPacketFee := types.NewMsgPayPacketFee(expectedFee, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, suite.chainA.SenderAccount.GetAddress().String(), nil)

	// fetch the account balance before fees are escrowed and assert the difference below
	preEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	res, err := suite.chainA.SendMsgs(msgPayPacketFee)
	suite.Require().NotNil(res)
	suite.Require().NoError(err)

	postEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	suite.Require().Equal(postEscrowBalance.AddAmount(expectedFee.Total().AmountOf(sdk.DefaultBondDenom)), preEscrowBalance)

	packetID := channeltypes.NewPacketId(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)
	packetFees, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(suite.chainA.GetContext(), packetID)
	suite.Require().True(found)
	suite.Require().Equal(expectedFee, packetFees.PacketFees[0].Fee)

	interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
	suite.Require().True(found)

	// fund the interchain account on chainB
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100000)))
	msgBankSend := &banktypes.MsgSend{
		FromAddress: suite.chainB.SenderAccount.GetAddress().String(),
		ToAddress:   interchainAccountAddr,
		Amount:      coins,
	}

	res, err = suite.chainB.SendMsgs(msgBankSend)
	suite.Require().NotEmpty(res)
	suite.Require().NoError(err)

	// prepare a simple stakingtypes.MsgDelegate to be used as the interchain account msg executed on chainB
	validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
	msgDelegate := &stakingtypes.MsgDelegate{
		DelegatorAddress: interchainAccountAddr,
		ValidatorAddress: validatorAddr.String(),
		Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000)),
	}

	data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msgDelegate})
	suite.Require().NoError(err)

	icaPacketData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}

	// ensure chainB is allowed to execute stakingtypes.MsgDelegate
	params := icahosttypes.NewParams(true, []string{sdk.MsgTypeURL(msgDelegate)})
	suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)

	// build the interchain accounts packet
	packet := buildInterchainAccountsPacket(path, icaPacketData.GetBytes(), 1)

	// write packet commitment to state on chainA and commit state
	commitment := channeltypes.CommitPacket(suite.chainA.GetSimApp().AppCodec(), packet)
	suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, commitment)
	suite.chainA.NextBlock()

	err = path.RelayPacket(packet)
	suite.Require().NoError(err)

	// ensure escrowed fees are cleaned up
	packetFees, found = suite.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(suite.chainA.GetContext(), packetID)
	suite.Require().False(found)
	suite.Require().Empty(packetFees)

	// assert the value of the account balance after fee distribution
	postDistBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	suite.Require().Equal(preEscrowBalance, postDistBalance)
}

func buildInterchainAccountsPacket(path *ibctesting.Path, data []byte, seq uint64) channeltypes.Packet {
	packet := channeltypes.NewPacket(
		data,
		1,
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		path.EndpointB.ChannelConfig.PortID,
		path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)

	return packet
}
