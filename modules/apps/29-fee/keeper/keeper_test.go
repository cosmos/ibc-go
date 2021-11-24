package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	path        *ibctesting.Path
	queryClient types.QueryClient
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	feeTransferVersion := channeltypes.MergeChannelVersions(types.Version, transfertypes.Version)
	path.EndpointA.ChannelConfig.Version = feeTransferVersion
	path.EndpointB.ChannelConfig.Version = feeTransferVersion
	path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.PortID
	suite.path = path

	queryHelper := baseapp.NewQueryServerTestHelper(suite.chainA.GetContext(), suite.chainA.GetSimApp().InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.chainA.GetSimApp().IBCFeeKeeper)
	suite.queryClient = types.NewQueryClient(queryHelper)
}

func SetupFeePath(path *ibctesting.Path) error {
	if err := path.EndpointA.ChanOpenInit(); err != nil {
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

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestGetAllIdentifiedPacketFees() {
	// escrow a fee
	refundAcc := suite.chainA.SenderAccount.GetAddress()
	ackFee := validCoins
	receiveFee := validCoins2
	timeoutFee := validCoins3
	packetId := &channeltypes.PacketId{ChannelId: ibctesting.FirstChannelID, PortId: types.PortID, Sequence: uint64(1)}
	fee := types.Fee{ackFee, receiveFee, timeoutFee}
	identifiedPacketFee := &types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, RefundAddress: refundAcc.String(), Relayers: []string{}}

	// escrow the packet fee
	err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedPacketFee)
	suite.Require().NoError(err)

	expectedFees := []*types.IdentifiedPacketFee{
		{
			PacketId:      packetId,
			Fee:           fee,
			RefundAddress: refundAcc.String(),
			Relayers:      nil,
		},
	}

	fees := suite.chainA.GetSimApp().IBCFeeKeeper.GetAllIdentifiedPacketFees(suite.chainA.GetContext())
	suite.Require().Len(fees, len(expectedFees))
	suite.Require().Equal(fees, expectedFees)
}

func (suite *KeeperTestSuite) TestGetAllFeeEnabledChannels() {
	suite.SetupTest() // reset

	validPortId := "ibcmoduleport"
	// set two channels enabled
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), transfertypes.PortID, ibctesting.FirstChannelID)
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), validPortId, ibctesting.FirstChannelID)

	expectedCh := []*types.FeeEnabledChannel{
		{
			PortId:    validPortId,
			ChannelId: ibctesting.FirstChannelID,
		},
		{
			PortId:    transfertypes.PortID,
			ChannelId: ibctesting.FirstChannelID,
		},
	}

	ch := suite.chainA.GetSimApp().IBCFeeKeeper.GetAllFeeEnabledChannels(suite.chainA.GetContext())
	suite.Require().Len(ch, len(expectedCh))
	suite.Require().Equal(ch, expectedCh)
}

func (suite *KeeperTestSuite) TestGetAllRelayerAddresses() {
	suite.SetupTest() // reset

	sender := suite.chainA.SenderAccount.GetAddress().String()
	counterparty := suite.chainB.SenderAccount.GetAddress().String()

	suite.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainA.GetContext(), sender, counterparty)

	expectedAddr := []*types.RegisteredRelayerAddress{
		{
			Address:             sender,
			CounterpartyAddress: counterparty,
		},
	}

	addr := suite.chainA.GetSimApp().IBCFeeKeeper.GetAllRelayerAddresses(suite.chainA.GetContext())
	suite.Require().Len(addr, len(expectedAddr))
	suite.Require().Equal(addr, expectedAddr)
}
