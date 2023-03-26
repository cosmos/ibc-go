package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))

	queryHelper := baseapp.NewQueryServerTestHelper(suite.chainA.GetContext(), suite.chainA.GetSimApp().InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.chainA.GetSimApp().TransferKeeper)
}

func NewTransferPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointA.ChannelConfig.Version = types.Version
	path.EndpointB.ChannelConfig.Version = types.Version

	return path
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestSetGetTotalEscrowForDenom() {
	var amount math.Int

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: with escrow amount > 2^63",
			func() {
				amount, _ = math.NewIntFromString("100000000000000000000")
			},
			true,
		},
		{
			"failure: setter panics with negative escrow amount",
			func() {
				amount = math.NewInt(-1)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			amount = math.ZeroInt()
			ctx := suite.chainA.GetContext()

			tc.malleate()

			if tc.expPass {
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(ctx, "atom", amount)
				total := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(ctx, "atom")
				suite.Require().Equal(amount, total)
			} else {
				suite.Require().PanicsWithValue("amount cannot be negative: -1", func() {
					suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(ctx, "atom", amount)
				})
				total := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(ctx, "atom")
				suite.Require().Equal(math.ZeroInt(), total)
			}
		})
	}
}
