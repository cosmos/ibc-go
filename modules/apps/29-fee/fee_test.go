package fee_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

type FeeTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	path *ibctesting.Path
}

func (suite *FeeTestSuite) SetupTest() {
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
}

func TestIBCFeeTestSuite(t *testing.T) {
	suite.Run(t, new(FeeTestSuite))
}

func (suite *FeeTestSuite) CreateICS20Packet(coin sdk.Coin) channeltypes.Packet {

	fungibleTokenPacket := transfertypes.NewFungibleTokenPacketData(
		coin.Denom,
		sdk.NewInt(100).Uint64(),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
	)

	return channeltypes.NewPacket(
		fungibleTokenPacket.GetBytes(),
		suite.chainA.SenderAccount.GetSequence(),
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)
}
