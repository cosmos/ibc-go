package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	ibcmock "github.com/cosmos/ibc-go/v3/testing/mock"
)

var (
	defaultReceiveFee = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	defaultAckFee     = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(200)}}
	defaultTimeoutFee = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(300)}}
	invalidCoins      = sdk.Coins{sdk.Coin{Denom: "invalidDenom", Amount: sdk.NewInt(100)}}
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

	queryClient types.QueryClient
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	mockFeeVersion := string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version}))
	path.EndpointA.ChannelConfig.Version = mockFeeVersion
	path.EndpointB.ChannelConfig.Version = mockFeeVersion
	path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
	path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
	suite.path = path

	path = ibctesting.NewPath(suite.chainA, suite.chainC)
	path.EndpointA.ChannelConfig.Version = mockFeeVersion
	path.EndpointB.ChannelConfig.Version = mockFeeVersion
	path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
	path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
	suite.pathAToC = path

	queryHelper := baseapp.NewQueryServerTestHelper(suite.chainA.GetContext(), suite.chainA.GetSimApp().InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.chainA.GetSimApp().IBCFeeKeeper)
	suite.queryClient = types.NewQueryClient(queryHelper)
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

func (suite *KeeperTestSuite) TestFeesInEscrow() {
	suite.coordinator.Setup(suite.path)

	// escrow five fees for packet sequence 1
	packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, 1)
	fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)

	for i := 1; i < 6; i++ {
		packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), nil)
		suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
	}

	// retrieve the fees in escrow and assert the length of PacketFees
	feesInEscrow, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(suite.chainA.GetContext(), packetID)
	suite.Require().True(found)
	suite.Require().Len(feesInEscrow.PacketFees, 5, fmt.Sprintf("expected length 5, but got %d", len(feesInEscrow.PacketFees)))

	// delete fees for packet sequence 1
	suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeesInEscrow(suite.chainA.GetContext(), packetID)
	hasFeesInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(suite.chainA.GetContext(), packetID)
	suite.Require().False(hasFeesInEscrow)
}

func (suite *KeeperTestSuite) TestIsLocked() {
	ctx := suite.chainA.GetContext()
	suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(ctx))

	lockFeeModule(suite.chainA)

	suite.Require().True(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(ctx))
}

func (suite *KeeperTestSuite) TestDisableAllChannels() {
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), "port1", "channel1")
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), "port2", "channel2")
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), "port3", "channel3")

	suite.chainA.GetSimApp().IBCFeeKeeper.DisableAllChannels(suite.chainA.GetContext())

	suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), "port1", "channel1"),
		"fee is still enabled on channel-1 after DisableAllChannels call")
	suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), "port2", "channel2"),
		"fee is still enabled on channel-2 after DisableAllChannels call")
	suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), "port3", "channel3"),
		"fee is still enabled on channel-3 after DisableAllChannels call")
}

func (suite *KeeperTestSuite) TestGetAllIdentifiedPacketFees() {
	suite.coordinator.Setup(suite.path)

	// escrow a fee
	refundAcc := suite.chainA.SenderAccount.GetAddress()
	packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, 1)
	fee := types.Fee{
		AckFee:     defaultAckFee,
		RecvFee:    defaultReceiveFee,
		TimeoutFee: defaultTimeoutFee,
	}

	// escrow the packet fee
	packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))

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

	identifiedFees := suite.chainA.GetSimApp().IBCFeeKeeper.GetAllIdentifiedPacketFees(suite.chainA.GetContext())
	suite.Require().Len(identifiedFees, len(expectedFees))
	suite.Require().Equal(identifiedFees, expectedFees)
}

func (suite *KeeperTestSuite) TestGetAllFeeEnabledChannels() {
	validPortId := "ibcmoduleport"
	// set two channels enabled
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), validPortId, ibctesting.FirstChannelID)

	expectedCh := []types.FeeEnabledChannel{
		{
			PortId:    validPortId,
			ChannelId: ibctesting.FirstChannelID,
		},
		{
			PortId:    ibctesting.MockFeePort,
			ChannelId: ibctesting.FirstChannelID,
		},
	}

	ch := suite.chainA.GetSimApp().IBCFeeKeeper.GetAllFeeEnabledChannels(suite.chainA.GetContext())
	suite.Require().Len(ch, len(expectedCh))
	suite.Require().Equal(ch, expectedCh)
}

func (suite *KeeperTestSuite) TestGetAllRelayerAddresses() {
	sender := suite.chainA.SenderAccount.GetAddress().String()
	counterparty := suite.chainB.SenderAccount.GetAddress().String()

	suite.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainA.GetContext(), sender, counterparty, ibctesting.FirstChannelID)

	expectedAddr := []types.RegisteredRelayerAddress{
		{
			Address:             sender,
			CounterpartyAddress: counterparty,
			ChannelId:           ibctesting.FirstChannelID,
		},
	}

	addr := suite.chainA.GetSimApp().IBCFeeKeeper.GetAllRelayerAddresses(suite.chainA.GetContext())
	suite.Require().Len(addr, len(expectedAddr))
	suite.Require().Equal(addr, expectedAddr)
}
