package fee_test

import (
	"testing"

	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
	"github.com/stretchr/testify/suite"
)

type FeeTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain

	path     *ibctesting.Path
	pathAToC *ibctesting.Path
}

func (s *FeeTestSuite) SetupTest() {
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

func TestIBCFeeTestSuite(t *testing.T) {
	suite.Run(t, new(FeeTestSuite))
}

func (s *FeeTestSuite) CreateMockPacket() channeltypes.Packet {
	return channeltypes.NewPacket(
		ibcmock.MockPacketData,
		s.chainA.SenderAccount.GetSequence(),
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID,
		s.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)
}

// helper function
func lockFeeModule(chain *ibctesting.TestChain) {
	ctx := chain.GetContext()
	storeKey := chain.GetSimApp().GetKey(types.ModuleName)
	store := ctx.KVStore(storeKey)
	store.Set(types.KeyLocked(), []byte{1})
}
