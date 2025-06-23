package keeper_test

import (
	"errors"
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	packetforward "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type KeeperTestSuite struct {
	testifysuite.Suite

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

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestNewKeeper() {
	testCases := []struct {
		name          string
		instantiateFn func()
		panicMsg      string
	}{
		{"success", func() {
			keeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(types.StoreKey)),
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				s.chainA.GetSimApp().MsgServiceRouter(),
				s.chainA.GetSimApp().AccountKeeper,
				s.chainA.GetSimApp().BankKeeper,
				s.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
			)
		}, ""},
		{"failure: transfer module account does not exist", func() {
			keeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(types.StoreKey)),
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				s.chainA.GetSimApp().MsgServiceRouter(),
				authkeeper.AccountKeeper{}, // empty account keeper
				s.chainA.GetSimApp().BankKeeper,
				s.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
			)
		}, "the IBC transfer module account has not been set"},
		{"failure: empty authority", func() {
			keeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(types.StoreKey)),
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				s.chainA.GetSimApp().MsgServiceRouter(),
				s.chainA.GetSimApp().AccountKeeper,
				s.chainA.GetSimApp().BankKeeper,
				"", // authority
			)
		}, "authority must be non-empty"},
	}

	for _, tc := range testCases {
		s.SetupTest()

		s.Run(tc.name, func() {
			if tc.panicMsg == "" {
				s.Require().NotPanics(
					tc.instantiateFn,
				)
			} else {
				s.Require().PanicsWithError(
					tc.panicMsg,
					tc.instantiateFn,
				)
			}
		})
	}
}

func (s *KeeperTestSuite) TestSetGetTotalEscrowForDenom() {
	const denom = "atom"
	var expAmount sdkmath.Int

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: with non-zero escrow amount",
			func() {},
			nil,
		},
		{
			"success: with escrow amount > 2^63",
			func() {
				expAmount, _ = sdkmath.NewIntFromString("100000000000000000000")
			},
			nil,
		},
		{
			"success: escrow amount 0 is not stored",
			func() {
				expAmount = sdkmath.ZeroInt()
			},
			nil,
		},
		{
			"failure: setter panics with negative escrow amount",
			func() {
				expAmount = sdkmath.NewInt(-1)
			},
			errors.New("amount cannot be negative: -1"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			expAmount = sdkmath.NewInt(100)
			ctx := s.chainA.GetContext()

			tc.malleate()

			coin := sdk.Coin{
				Denom:  denom,
				Amount: expAmount,
			}

			if tc.expError == nil {
				s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(ctx, coin)
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
				s.Require().PanicsWithError(tc.expError.Error(), func() {
					s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(ctx, coin)
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
				denom := "uatom" //nolint:goconst
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
				store.Set(fmt.Appendf(nil, "wrong-prefix/%s", denom), bz)
			},
			false,
		},
	}

	for _, tc := range testCases {
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
		name     string
		input    types.Params
		panicMsg string
	}{
		// it is not possible to set invalid booleans
		{"success: set params false-false", types.NewParams(false, false), ""},
		{"success: set params false-true", types.NewParams(false, true), ""},
		{"success: set params true-false", types.NewParams(true, false), ""},
		{"success: set params true-true", types.NewParams(true, true), ""},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()
			if tc.panicMsg == "" {
				s.chainA.GetSimApp().TransferKeeper.SetParams(ctx, tc.input)
				expected := tc.input
				p := s.chainA.GetSimApp().TransferKeeper.GetParams(ctx)
				s.Require().Equal(expected, p)
			} else {
				s.Require().PanicsWithError(tc.panicMsg, func() {
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

func (s *KeeperTestSuite) TestWithICS4Wrapper() {
	s.SetupTest()

	// test if the ics4 wrapper is the pfm keeper initially
	ics4Wrapper := s.chainA.GetSimApp().TransferKeeper.GetICS4Wrapper()

	_, isPFMKeeper := ics4Wrapper.(*packetforward.IBCMiddleware)
	s.Require().True(isPFMKeeper)
	s.Require().IsType((*packetforward.IBCMiddleware)(nil), ics4Wrapper)

	// set the ics4 wrapper to the channel keeper
	s.chainA.GetSimApp().TransferKeeper.WithICS4Wrapper(nil)
	ics4Wrapper = s.chainA.GetSimApp().TransferKeeper.GetICS4Wrapper()
	s.Require().Nil(ics4Wrapper)
}

func (s *KeeperTestSuite) TestIsBlockedAddr() {
	s.SetupTest()

	testCases := []struct {
		name     string
		addr     sdk.AccAddress
		expBlock bool
	}{
		{
			"transfer module account address",
			s.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName),
			false,
		},
		{
			"regular address",
			s.chainA.SenderAccount.GetAddress(),
			false,
		},
		{
			"blocked address",
			s.chainA.GetSimApp().AccountKeeper.GetModuleAddress(minttypes.ModuleName),
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.expBlock, s.chainA.GetSimApp().TransferKeeper.IsBlockedAddr(tc.addr))
		})
	}
}
