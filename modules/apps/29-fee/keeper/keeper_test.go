package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

var (
	defaultRecvFee    = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}}
	defaultAckFee     = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(200)}}
	defaultTimeoutFee = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(300)}}
	invalidCoins      = sdk.Coins{sdk.Coin{Denom: "invalidDenom", Amount: sdkmath.NewInt(100)}}
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain

	path     *ibctesting.Path
	pathAToC *ibctesting.Path
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))

	path := ibctesting.NewPath(s.chainA, s.chainB)
	mockFeeVersion := string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version}))
	path.EndpointA.ChannelConfig.Version = mockFeeVersion
	path.EndpointB.ChannelConfig.Version = mockFeeVersion
	path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
	path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
	s.path = path

	path = ibctesting.NewPath(s.chainA, s.chainC)
	path.EndpointA.ChannelConfig.Version = mockFeeVersion
	path.EndpointB.ChannelConfig.Version = mockFeeVersion
	path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
	path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
	s.pathAToC = path
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// helper function
func lockFeeModule(chain *ibctesting.TestChain) {
	ctx := chain.GetContext()
	storeKey := chain.GetSimApp().GetKey(types.ModuleName)
	store := ctx.KVStore(storeKey)
	store.Set(types.KeyLocked(), []byte{1})
}

func (s *KeeperTestSuite) TestEscrowAccountHasBalance() {
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

	s.Require().False(s.chainA.GetSimApp().IBCFeeKeeper.EscrowAccountHasBalance(s.chainA.GetContext(), fee.Total()))

	// set fee in escrow account
	err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), types.ModuleName, fee.Total())
	s.Require().Nil(err)

	s.Require().True(s.chainA.GetSimApp().IBCFeeKeeper.EscrowAccountHasBalance(s.chainA.GetContext(), fee.Total()))

	// increase ack fee
	fee.AckFee = fee.AckFee.Add(defaultAckFee...)
	s.Require().False(s.chainA.GetSimApp().IBCFeeKeeper.EscrowAccountHasBalance(s.chainA.GetContext(), fee.Total()))
}

func (s *KeeperTestSuite) TestGetSetPayeeAddress() {
	s.coordinator.Setup(s.path)

	payeeAddr, found := s.chainA.GetSimApp().IBCFeeKeeper.GetPayeeAddress(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress().String(), s.path.EndpointA.ChannelID)
	s.Require().False(found)
	s.Require().Empty(payeeAddr)

	s.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
		s.chainA.GetContext(),
		s.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(),
		s.chainA.SenderAccounts[1].SenderAccount.GetAddress().String(),
		s.path.EndpointA.ChannelID,
	)

	payeeAddr, found = s.chainA.GetSimApp().IBCFeeKeeper.GetPayeeAddress(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress().String(), s.path.EndpointA.ChannelID)
	s.Require().True(found)
	s.Require().Equal(s.chainA.SenderAccounts[1].SenderAccount.GetAddress().String(), payeeAddr)
}

func (s *KeeperTestSuite) TestFeesInEscrow() {
	s.coordinator.Setup(s.path)

	// escrow five fees for packet sequence 1
	packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

	packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), nil)
	packetFees := []types.PacketFee{packetFee, packetFee, packetFee, packetFee, packetFee}

	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

	// retrieve the fees in escrow and assert the length of PacketFees
	feesInEscrow, found := s.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(s.chainA.GetContext(), packetID)
	s.Require().True(found)
	s.Require().Len(feesInEscrow.PacketFees, 5, fmt.Sprintf("expected length 5, but got %d", len(feesInEscrow.PacketFees)))

	// delete fees for packet sequence 1
	s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeesInEscrow(s.chainA.GetContext(), packetID)
	hasFeesInEscrow := s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID)
	s.Require().False(hasFeesInEscrow)
}

func (s *KeeperTestSuite) TestIsLocked() {
	ctx := s.chainA.GetContext()
	s.Require().False(s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(ctx))

	lockFeeModule(s.chainA)

	s.Require().True(s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(ctx))
}

func (s *KeeperTestSuite) TestGetIdentifiedPacketFeesForChannel() {
	s.coordinator.Setup(s.path)

	// escrow a fee
	refundAcc := s.chainA.SenderAccount.GetAddress()
	packetID1 := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
	packetID2 := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 2)
	packetID5 := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 51)

	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

	// escrow the packet fee
	packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID1, types.NewPacketFees([]types.PacketFee{packetFee}))
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID2, types.NewPacketFees([]types.PacketFee{packetFee}))
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID5, types.NewPacketFees([]types.PacketFee{packetFee}))

	// set fees in escrow for packetIDs on different channel
	diffChannel := "channel-1"
	diffPacketID1 := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, diffChannel, 1)
	diffPacketID2 := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, diffChannel, 2)
	diffPacketID5 := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, diffChannel, 5)
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), diffPacketID1, types.NewPacketFees([]types.PacketFee{packetFee}))
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), diffPacketID2, types.NewPacketFees([]types.PacketFee{packetFee}))
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), diffPacketID5, types.NewPacketFees([]types.PacketFee{packetFee}))

	expectedFees := []types.IdentifiedPacketFees{
		{
			PacketId: packetID1,
			PacketFees: []types.PacketFee{
				{
					Fee:           fee,
					RefundAddress: refundAcc.String(),
					Relayers:      nil,
				},
			},
		},
		{
			PacketId: packetID2,
			PacketFees: []types.PacketFee{
				{
					Fee:           fee,
					RefundAddress: refundAcc.String(),
					Relayers:      nil,
				},
			},
		},
		{
			PacketId: packetID5,
			PacketFees: []types.PacketFee{
				{
					Fee:           fee,
					RefundAddress: refundAcc.String(),
					Relayers:      nil,
				},
			},
		},
	}

	identifiedFees := s.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	s.Require().Len(identifiedFees, len(expectedFees))
	s.Require().Equal(identifiedFees, expectedFees)
}

func (s *KeeperTestSuite) TestGetAllIdentifiedPacketFees() {
	s.coordinator.Setup(s.path)

	// escrow a fee
	refundAcc := s.chainA.SenderAccount.GetAddress()
	packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

	// escrow the packet fee
	packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))

	expectedFees := []types.IdentifiedPacketFees{
		{
			PacketId: packetID,
			PacketFees: []types.PacketFee{
				{
					Fee:           fee,
					RefundAddress: refundAcc.String(),
					Relayers:      nil,
				},
			},
		},
	}

	identifiedFees := s.chainA.GetSimApp().IBCFeeKeeper.GetAllIdentifiedPacketFees(s.chainA.GetContext())
	s.Require().Len(identifiedFees, len(expectedFees))
	s.Require().Equal(identifiedFees, expectedFees)
}

func (s *KeeperTestSuite) TestGetAllFeeEnabledChannels() {
	validPortID := "ibcmoduleport"
	// set two channels enabled
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), validPortID, ibctesting.FirstChannelID)

	expectedCh := []types.FeeEnabledChannel{
		{
			PortId:    validPortID,
			ChannelId: ibctesting.FirstChannelID,
		},
		{
			PortId:    ibctesting.MockFeePort,
			ChannelId: ibctesting.FirstChannelID,
		},
	}

	ch := s.chainA.GetSimApp().IBCFeeKeeper.GetAllFeeEnabledChannels(s.chainA.GetContext())
	s.Require().Len(ch, len(expectedCh))
	s.Require().Equal(ch, expectedCh)
}

func (s *KeeperTestSuite) TestGetAllPayees() {
	var expectedPayees []types.RegisteredPayee

	for i := 0; i < 3; i++ {
		s.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
			s.chainA.GetContext(),
			s.chainA.SenderAccounts[i].SenderAccount.GetAddress().String(),
			s.chainB.SenderAccounts[i].SenderAccount.GetAddress().String(),
			ibctesting.FirstChannelID,
		)

		registeredPayee := types.RegisteredPayee{
			Relayer:   s.chainA.SenderAccounts[i].SenderAccount.GetAddress().String(),
			Payee:     s.chainB.SenderAccounts[i].SenderAccount.GetAddress().String(),
			ChannelId: ibctesting.FirstChannelID,
		}

		expectedPayees = append(expectedPayees, registeredPayee)
	}

	registeredPayees := s.chainA.GetSimApp().IBCFeeKeeper.GetAllPayees(s.chainA.GetContext())
	s.Require().Len(registeredPayees, len(expectedPayees))
	s.Require().ElementsMatch(expectedPayees, registeredPayees)
}

func (s *KeeperTestSuite) TestGetAllCounterpartyPayees() {
	relayerAddr := s.chainA.SenderAccount.GetAddress().String()
	counterpartyPayee := s.chainB.SenderAccount.GetAddress().String()

	s.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(s.chainA.GetContext(), relayerAddr, counterpartyPayee, ibctesting.FirstChannelID)

	expectedCounterpartyPayee := []types.RegisteredCounterpartyPayee{
		{
			Relayer:           relayerAddr,
			CounterpartyPayee: counterpartyPayee,
			ChannelId:         ibctesting.FirstChannelID,
		},
	}

	counterpartyPayeeAddr := s.chainA.GetSimApp().IBCFeeKeeper.GetAllCounterpartyPayees(s.chainA.GetContext())
	s.Require().Len(counterpartyPayeeAddr, len(expectedCounterpartyPayee))
	s.Require().Equal(counterpartyPayeeAddr, expectedCounterpartyPayee)
}
