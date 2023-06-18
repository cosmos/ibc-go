package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))

	queryHelper := baseapp.NewQueryServerTestHelper(s.chainA.GetContext(), s.chainA.GetSimApp().InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, s.chainA.GetSimApp().TransferKeeper)
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

func (s *KeeperTestSuite) TestSetGetTotalEscrowForDenom() {
	const denom = "atom"
	var expAmount sdkmath.Int

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: with non-zero escrow amount",
			func() {},
			true,
		},
		{
			"success: with escrow amount > 2^63",
			func() {
				expAmount, _ = sdkmath.NewIntFromString("100000000000000000000")
			},
			true,
		},
		{
			"success: escrow amount 0 is not stored",
			func() {
				expAmount = sdkmath.ZeroInt()
			},
			true,
		},
		{
			"failure: setter panics with negative escrow amount",
			func() {
				expAmount = sdkmath.NewInt(-1)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			expAmount = sdkmath.NewInt(100)
			ctx := s.chainA.GetContext()

			tc.malleate()

			if tc.expPass {
				s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(ctx, sdk.NewCoin(denom, expAmount))
				total := s.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(ctx, denom)
				s.Require().Equal(expAmount, total.Amount)

				storeKey := s.chainA.GetSimApp().GetKey(types.ModuleName)
				store := ctx.KVStore(storeKey)
				key := types.TotalEscrowForDenomKey(denom)
				if expAmount.IsZero() {
					s.Require().False(store.Has(key))
				} else {
					s.Require().True(store.Has(key))
				}
			} else {
				s.Require().PanicsWithError("negative coin amount: -1", func() {
					s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(ctx, sdk.NewCoin(denom, expAmount))
				})
				total := s.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(ctx, denom)
				s.Require().Equal(sdkmath.ZeroInt(), total.Amount)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetAllDenomEscrows() {
	var (
		store           storetypes.KVStore
		cdc             codec.Codec
		expDenomEscrows sdk.Coins
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				denom := "uatom"
				amount := sdkmath.NewInt(100)
				expDenomEscrows = append(expDenomEscrows, sdk.NewCoin(denom, amount))

				bz := cdc.MustMarshal(&sdk.IntProto{Int: amount})
				store.Set(types.TotalEscrowForDenomKey(denom), bz)
			},
			true,
		},
		{
			"success: multiple denoms",
			func() {
				denom := "uatom"
				amount := sdkmath.NewInt(100)
				expDenomEscrows = append(expDenomEscrows, sdk.NewCoin(denom, amount))

				bz := cdc.MustMarshal(&sdk.IntProto{Int: amount})
				store.Set(types.TotalEscrowForDenomKey(denom), bz)

				denom = "bar/foo"
				amount = sdkmath.NewInt(50)
				expDenomEscrows = append(expDenomEscrows, sdk.NewCoin(denom, amount))

				bz = cdc.MustMarshal(&sdk.IntProto{Int: amount})
				store.Set(types.TotalEscrowForDenomKey(denom), bz)
			},
			true,
		},
		{
			"success: denom with non-alphanumeric characters",
			func() {
				denom := "ibc/123-456"
				amount := sdkmath.NewInt(100)
				expDenomEscrows = append(expDenomEscrows, sdk.NewCoin(denom, amount))

				bz := cdc.MustMarshal(&sdk.IntProto{Int: amount})
				store.Set(types.TotalEscrowForDenomKey(denom), bz)
			},
			true,
		},
		{
			"failure: empty denom",
			func() {
				denom := ""
				amount := sdkmath.ZeroInt()

				bz := cdc.MustMarshal(&sdk.IntProto{Int: amount})
				store.Set(types.TotalEscrowForDenomKey(denom), bz)
			},
			false,
		},
		{
			"failure: wrong prefix key",
			func() {
				denom := "uatom"
				amount := sdkmath.ZeroInt()

				bz := cdc.MustMarshal(&sdk.IntProto{Int: amount})
				store.Set([]byte(fmt.Sprintf("wrong-prefix/%s", denom)), bz)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			expDenomEscrows = sdk.Coins{}
			ctx := s.chainA.GetContext()

			storeKey := s.chainA.GetSimApp().GetKey(types.ModuleName)
			store = ctx.KVStore(storeKey)
			cdc = s.chainA.App.AppCodec()

			tc.malleate()

			denomEscrows := s.chainA.GetSimApp().TransferKeeper.GetAllTotalEscrowed(ctx)

			if tc.expPass {
				s.Require().Len(expDenomEscrows, len(denomEscrows))
				s.Require().ElementsMatch(expDenomEscrows, denomEscrows)
			} else {
				s.Require().Empty(denomEscrows)
			}
		})
	}
}

func (s *KeeperTestSuite) TestParams() {
	testCases := []struct {
		name    string
		input   types.Params
		expPass bool
	}{
		// it is not possible to set invalid booleans
		{"success: set params false-false", types.NewParams(false, false), true},
		{"success: set params false-true", types.NewParams(false, true), true},
		{"success: set params true-false", types.NewParams(true, false), true},
		{"success: set params true-true", types.NewParams(true, true), true},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()
			if tc.expPass {
				s.chainA.GetSimApp().TransferKeeper.SetParams(ctx, tc.input)
				expected := tc.input
				p := s.chainA.GetSimApp().TransferKeeper.GetParams(ctx)
				s.Require().Equal(expected, p)
			} else {
				s.Require().Panics(func() {
					s.chainA.GetSimApp().TransferKeeper.SetParams(ctx, tc.input)
				})
			}
		})
	}
}

func (s *KeeperTestSuite) TestUnsetParams() {
	s.SetupTest()

	ctx := s.chainA.GetContext()
	store := s.chainA.GetContext().KVStore(s.chainA.GetSimApp().GetKey(types.ModuleName))
	store.Delete([]byte(types.ParamsKey))

	s.Require().Panics(func() {
		s.chainA.GetSimApp().TransferKeeper.GetParams(ctx)
	})
}
