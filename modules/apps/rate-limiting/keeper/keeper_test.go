package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	ratelimittypes "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestNewKeeper() {
	testCases := []struct {
		name          string
		instantiateFn func()
		panicMsg      string
	}{
		{
			name: "success",
			instantiateFn: func() {
				keeper.NewKeeper(
					s.chainA.GetSimApp().AppCodec(),
					runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ratelimittypes.StoreKey)),
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					s.chainA.GetSimApp().IBCKeeper.ClientKeeper, // Add clientKeeper
					s.chainA.GetSimApp().BankKeeper,
					s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(),
				)
			},
			panicMsg: "",
		},
		{
			name: "failure: empty authority",
			instantiateFn: func() {
				keeper.NewKeeper(
					s.chainA.GetSimApp().AppCodec(),
					runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ratelimittypes.StoreKey)),
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					s.chainA.GetSimApp().IBCKeeper.ClientKeeper, // clientKeeper
					s.chainA.GetSimApp().BankKeeper,
					"", // empty authority
				)
			},
			panicMsg: "authority must be non-empty",
		},
	}

	for _, tc := range testCases {
		s.SetupTest()

		s.Run(tc.name, func() {
			if tc.panicMsg == "" {
				s.Require().NotPanics(tc.instantiateFn)
			} else {
				s.Require().PanicsWithError(tc.panicMsg, tc.instantiateFn)
			}
		})
	}
}
