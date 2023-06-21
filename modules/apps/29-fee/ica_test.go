package fee_test

import (
	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

var (
	// defaultOwnerAddress defines a reusable bech32 address for testing purposes
	defaultOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// defaultPortID defines a reusable port identifier for testing purposes
	defaultPortID, _ = icatypes.NewControllerPortID(defaultOwnerAddress)

	// defaultICAVersion defines a reusable interchainaccounts version string for testing purposes
	defaultICAVersion = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
)

// NewIncentivizedICAPath creates and returns a new ibctesting path configured for a fee enabled interchain accounts channel
func NewIncentivizedICAPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)

	feeMetadata := types.Metadata{
		FeeVersion: types.Version,
		AppVersion: defaultICAVersion,
	}

	feeICAVersion := string(types.ModuleCdc.MustMarshalJSON(&feeMetadata))

	path.SetChannelOrdered()
	path.EndpointA.ChannelConfig.Version = feeICAVersion
	path.EndpointB.ChannelConfig.Version = feeICAVersion
	path.EndpointA.ChannelConfig.PortID = defaultPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID

	return path
}

// SetupPath performs the InterchainAccounts channel creation handshake using an ibctesting path
func SetupPath(path *ibctesting.Path, owner string) error {
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

// RegisterInterchainAccount invokes the the InterchainAccounts entrypoint, routes a new MsgChannelOpenInit to the appropriate handler,
// commits state changes and updates the testing endpoint accordingly
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
func (s *FeeTestSuite) TestFeeInterchainAccounts() {
	path := NewIncentivizedICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupPath(path, defaultOwnerAddress)
	s.Require().NoError(err)

	// assert the newly established channel is fee enabled on both ends
	s.Require().True(s.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
	s.Require().True(s.chainB.GetSimApp().IBCFeeKeeper.IsFeeEnabled(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))

	// register counterparty address on destination chainB as chainA.SenderAccounts[1] for recv fee distribution
	s.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress().String(), s.chainA.SenderAccounts[1].SenderAccount.GetAddress().String(), path.EndpointB.ChannelID)

	// escrow a packet fee for the next send sequence
	expectedFee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	msgPayPacketFee := types.NewMsgPayPacketFee(expectedFee, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, s.chainA.SenderAccount.GetAddress().String(), nil)

	// fetch the account balance before fees are escrowed and assert the difference below
	preEscrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	res, err := s.chainA.SendMsgs(msgPayPacketFee)
	s.Require().NotNil(res)
	s.Require().NoError(err)

	postEscrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	s.Require().Equal(postEscrowBalance.AddAmount(expectedFee.Total().AmountOf(sdk.DefaultBondDenom)), preEscrowBalance)

	packetID := channeltypes.NewPacketID(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)
	packetFees, found := s.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(s.chainA.GetContext(), packetID)
	s.Require().True(found)
	s.Require().Equal(expectedFee, packetFees.PacketFees[0].Fee)

	interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
	s.Require().True(found)

	// fund the interchain account on chainB
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000)))
	msgBankSend := &banktypes.MsgSend{
		FromAddress: s.chainB.SenderAccount.GetAddress().String(),
		ToAddress:   interchainAccountAddr,
		Amount:      coins,
	}

	res, err = s.chainB.SendMsgs(msgBankSend)
	s.Require().NotEmpty(res)
	s.Require().NoError(err)

	// prepare a simple stakingtypes.MsgDelegate to be used as the interchain account msg executed on chainB
	validatorAddr := (sdk.ValAddress)(s.chainB.Vals.Validators[0].Address)
	msgDelegate := &stakingtypes.MsgDelegate{
		DelegatorAddress: interchainAccountAddr,
		ValidatorAddress: validatorAddr.String(),
		Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
	}

	data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msgDelegate})
	s.Require().NoError(err)

	icaPacketData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}

	// ensure chainB is allowed to execute stakingtypes.MsgDelegate
	params := icahosttypes.NewParams(true, []string{sdk.MsgTypeURL(msgDelegate)})
	s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)

	// build the interchain accounts packet
	packet := buildInterchainAccountsPacket(path, icaPacketData.GetBytes(), 1)

	// write packet commitment to state on chainA and commit state
	commitment := channeltypes.CommitPacket(s.chainA.GetSimApp().AppCodec(), packet)
	s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1, commitment)
	s.chainA.NextBlock()

	err = path.RelayPacket(packet)
	s.Require().NoError(err)

	// ensure escrowed fees are cleaned up
	packetFees, found = s.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(s.chainA.GetContext(), packetID)
	s.Require().False(found)
	s.Require().Empty(packetFees)

	// assert the value of the account balance after fee distribution
	// NOTE: the balance after fee distribution should be equal to the pre-escrow balance minus the recv fee
	// as chainA.SenderAccount is used as the msg signer and refund address for msgPayPacketFee above as well as the relyer account for acknowledgements in path.RelayPacket()
	postDistBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	s.Require().Equal(preEscrowBalance.SubAmount(defaultRecvFee.AmountOf(sdk.DefaultBondDenom)), postDistBalance)
}

func buildInterchainAccountsPacket(path *ibctesting.Path, data []byte, seq uint64) channeltypes.Packet {
	packet := channeltypes.NewPacket(
		data,
		1,
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		path.EndpointB.ChannelConfig.PortID,
		path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	return packet
}
