package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

var (
	validCoins   = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	validCoins2  = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(200)}}
	validCoins3  = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(300)}}
	invalidCoins = sdk.Coins{sdk.Coin{Denom: "invalidDenom", Amount: sdk.NewInt(100)}}
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

func (suite *KeeperTestSuite) TestFeeInEscrow() {
	ackFee := validCoins
	receiveFee := validCoins2
	timeoutFee := validCoins3
	fee := types.Fee{ReceiveFee: receiveFee, AckFee: ackFee, TimeoutFee: timeoutFee}

	// set some fees
	for i := 1; i < 6; i++ {
		packetId := types.NewPacketId(fmt.Sprintf("channel-1"), uint64(i))
		fee := types.NewIdentifiedPacketFee(packetId, fee, suite.chainA.SenderAccount.GetAddress().String(), []string{})
		suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeInEscrow(suite.chainA.GetContext(), fee)
	}

	// delete 1 fee
	packetId := types.NewPacketId("channel-1", 3)
	suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeInEscrow(suite.chainA.GetContext(), packetId)

	// iterate over remaining fees
	arr := []int64{}
	expectedArr := []int64{1, 2, 4, 5}
	suite.chainA.GetSimApp().IBCFeeKeeper.IterateChannelFeesInEscrow(suite.chainA.GetContext(), types.PortKey, "channel-1", func(identifiedFee types.IdentifiedPacketFee) (stop bool) {
		arr = append(arr, int64(identifiedFee.PacketId.Sequence))
		return false
	})
	suite.Require().Equal(expectedArr, arr, "did not retrieve expected fees during iteration")
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
