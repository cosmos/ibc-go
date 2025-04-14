package keeper_test

import (
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

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
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

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))

	queryHelper := baseapp.NewQueryServerTestHelper(suite.chainA.GetContext(), suite.chainA.GetSimApp().InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.chainA.GetSimApp().TransferKeeper)
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestNewKeeper() {
	testCases := []struct {
		name          string
		instantiateFn func()
		panicMsg      string
	}{
		{"success", func() {
			keeper.NewKeeper(
				suite.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(types.StoreKey)),
				suite.chainA.GetSimApp().GetSubspace(types.ModuleName),
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().MsgServiceRouter(),
				suite.chainA.GetSimApp().AccountKeeper,
				suite.chainA.GetSimApp().BankKeeper,
				suite.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
			)
		}, ""},
		{"failure: transfer module account does not exist", func() {
			keeper.NewKeeper(
				suite.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(types.StoreKey)),
				suite.chainA.GetSimApp().GetSubspace(types.ModuleName),
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().MsgServiceRouter(),
				authkeeper.AccountKeeper{}, // empty account keeper
				suite.chainA.GetSimApp().BankKeeper,
				suite.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
			)
		}, "the IBC transfer module account has not been set"},
		{"failure: empty authority", func() {
			keeper.NewKeeper(
				suite.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(types.StoreKey)),
				suite.chainA.GetSimApp().GetSubspace(types.ModuleName),
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().MsgServiceRouter(),
				suite.chainA.GetSimApp().AccountKeeper,
				suite.chainA.GetSimApp().BankKeeper,
				"", // authority
			)
		}, "authority must be non-empty"},
	}

	for _, tc := range testCases {

		suite.SetupTest()

		suite.Run(tc.name, func() {
			if tc.panicMsg == "" {
				suite.Require().NotPanics(
					tc.instantiateFn,
				)
			} else {
				suite.Require().PanicsWithError(
					tc.panicMsg,
					tc.instantiateFn,
				)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSetGetTotalEscrowForDenom() {
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
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			expAmount = sdkmath.NewInt(100)
			ctx := suite.chainA.GetContext()

			tc.malleate()

			if tc.expPass {
				suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(ctx, sdk.NewCoin(denom, expAmount))
				total := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(ctx, denom)
				suite.Require().Equal(expAmount, total.Amount)

				storeKey := suite.chainA.GetSimApp().GetKey(types.ModuleName)
				store := ctx.KVStore(storeKey)
				key := types.TotalEscrowForDenomKey(denom)
				if expAmount.IsZero() {
					suite.Require().False(store.Has(key))
				} else {
					suite.Require().True(store.Has(key))
				}
			} else {
				suite.Require().PanicsWithError("negative coin amount: -1", func() {
					suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(ctx, sdk.NewCoin(denom, expAmount))
				})
				total := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(ctx, denom)
				suite.Require().Equal(sdkmath.ZeroInt(), total.Amount)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetAllDenomEscrows() {
	var (
		store storetypes.KVStore
		cdc   codec.Codec
	)

	testCases := []struct {
		name     string
		malleate func()
		expected sdk.Coins
	}{
		{
			"success",
			func() {
				denom := "uatom" //nolint:goconst
				amount := sdkmath.NewInt(100)
				bz := cdc.MustMarshal(&sdk.IntProto{Int: amount})
				store.Set(types.TotalEscrowForDenomKey(denom), bz)
			},
			sdk.NewCoins(sdk.NewInt64Coin("uatom", 100)),
		},
		{
			"success: multiple denoms",
			func() {
				store.Set(types.TotalEscrowForDenomKey("uatom"), cdc.MustMarshal(&sdk.IntProto{Int: sdkmath.NewInt(100)}))
				store.Set(types.TotalEscrowForDenomKey("bar/foo"), cdc.MustMarshal(&sdk.IntProto{Int: sdkmath.NewInt(50)}))
			},
			sdk.NewCoins(
				sdk.NewInt64Coin("uatom", 100),
				sdk.NewInt64Coin("bar/foo", 50),
			),
		},
		{
			"success: denom with non-alphanumeric characters",
			func() {
				store.Set(types.TotalEscrowForDenomKey("ibc/123-456"), cdc.MustMarshal(&sdk.IntProto{Int: sdkmath.NewInt(100)}))
			},
			sdk.NewCoins(sdk.NewInt64Coin("ibc/123-456", 100)),
		},
		{
			"failure: empty denom",
			func() {
				store.Set(types.TotalEscrowForDenomKey(""), cdc.MustMarshal(&sdk.IntProto{Int: sdkmath.ZeroInt()}))
			},
			sdk.Coins{},
		},
		{
			"failure: wrong prefix key",
			func() {
				key := []byte("wrong-prefix/uatom")
				store.Set(key, cdc.MustMarshal(&sdk.IntProto{Int: sdkmath.ZeroInt()}))
			},
			sdk.Coins{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			ctx := suite.chainA.GetContext()
			storeKey := suite.chainA.GetSimApp().GetKey(types.ModuleName)
			store = ctx.KVStore(storeKey)
			cdc = suite.chainA.App.AppCodec()

			tc.malleate()

			actual := suite.chainA.GetSimApp().TransferKeeper.GetAllTotalEscrowed(ctx)
			suite.Require().ElementsMatch(tc.expected, actual)
		})
	}
}

func (suite *KeeperTestSuite) TestParams() {
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
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			ctx := suite.chainA.GetContext()
			if tc.panicMsg == "" {
				suite.chainA.GetSimApp().TransferKeeper.SetParams(ctx, tc.input)
				expected := tc.input
				p := suite.chainA.GetSimApp().TransferKeeper.GetParams(ctx)
				suite.Require().Equal(expected, p)
			} else {
				suite.Require().PanicsWithError(tc.panicMsg, func() {
					suite.chainA.GetSimApp().TransferKeeper.SetParams(ctx, tc.input)
				})
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUnsetParams() {
	suite.SetupTest()

	ctx := suite.chainA.GetContext()
	store := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetKey(types.ModuleName))
	store.Delete([]byte(types.ParamsKey))

	suite.Require().Panics(func() {
		suite.chainA.GetSimApp().TransferKeeper.GetParams(ctx)
	})
}

func (suite *KeeperTestSuite) TestWithICS4Wrapper() {
	suite.SetupTest()

	// test if the ics4 wrapper is the channel keeper initially
	ics4Wrapper := suite.chainA.GetSimApp().TransferKeeper.GetICS4Wrapper()

	_, isChannelKeeper := ics4Wrapper.(*channelkeeper.Keeper)
	suite.Require().True(isChannelKeeper)
	suite.Require().IsType((*channelkeeper.Keeper)(nil), ics4Wrapper)

	// set the ics4 wrapper to the channel keeper
	suite.chainA.GetSimApp().TransferKeeper.WithICS4Wrapper(nil)
	ics4Wrapper = suite.chainA.GetSimApp().TransferKeeper.GetICS4Wrapper()
	suite.Require().Nil(ics4Wrapper)
}

func (suite *KeeperTestSuite) TestIsBlockedAddr() {
	suite.SetupTest()

	testCases := []struct {
		name     string
		addr     sdk.AccAddress
		expBlock bool
	}{
		{
			"transfer module account address",
			suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName),
			false,
		},
		{
			"regular address",
			suite.chainA.SenderAccount.GetAddress(),
			false,
		},
		{
			"blocked address",
			suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(minttypes.ModuleName),
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.expBlock, suite.chainA.GetSimApp().TransferKeeper.IsBlockedAddr(tc.addr))
		})
	}
}
