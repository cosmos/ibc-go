package keeper_test

import (
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	minttypes "cosmossdk.io/x/mint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
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
				runtime.NewEnvironment(runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(types.StoreKey)), log.NewNopLogger()),
				suite.chainA.GetSimApp().GetSubspace(types.ModuleName),
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().AuthKeeper,
				suite.chainA.GetSimApp().BankKeeper,
				suite.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
			)
		}, ""},
		{"failure: transfer module account does not exist", func() {
			keeper.NewKeeper(
				suite.chainA.GetSimApp().AppCodec(),
				runtime.NewEnvironment(runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(types.StoreKey)), log.NewNopLogger()),
				suite.chainA.GetSimApp().GetSubspace(types.ModuleName),
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				authkeeper.AccountKeeper{}, // empty account keeper
				suite.chainA.GetSimApp().BankKeeper,
				suite.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
			)
		}, "the IBC transfer module account has not been set"},
		{"failure: empty authority", func() {
			keeper.NewKeeper(
				suite.chainA.GetSimApp().AppCodec(),
				runtime.NewEnvironment(runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(types.StoreKey)), log.NewNopLogger()),
				suite.chainA.GetSimApp().GetSubspace(types.ModuleName),
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				suite.chainA.GetSimApp().AuthKeeper,
				suite.chainA.GetSimApp().BankKeeper,
				"", // authority
			)
		}, "authority must be non-empty"},
	}

	for _, tc := range testCases {
		tc := tc
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
		tc := tc

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
		store           storetypes.KVStore
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

				bz, err := amount.Marshal()
				suite.Require().NoError(err)
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

				bz, err := amount.Marshal()
				suite.Require().NoError(err)
				store.Set(types.TotalEscrowForDenomKey(denom), bz)

				denom = "bar/foo"
				amount = sdkmath.NewInt(50)
				expDenomEscrows = append(expDenomEscrows, sdk.NewCoin(denom, amount))

				bz, err = amount.Marshal()
				suite.Require().NoError(err)
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

				bz, err := amount.Marshal()
				suite.Require().NoError(err)
				store.Set(types.TotalEscrowForDenomKey(denom), bz)
			},
			true,
		},
		{
			"failure: empty denom",
			func() {
				denom := ""
				amount := sdkmath.ZeroInt()

				bz, err := amount.Marshal()
				suite.Require().NoError(err)
				store.Set(types.TotalEscrowForDenomKey(denom), bz)
			},
			false,
		},
		{
			"failure: wrong prefix key",
			func() {
				denom := "uatom"
				amount := sdkmath.ZeroInt()

				bz, err := amount.Marshal()
				suite.Require().NoError(err)
				store.Set([]byte(fmt.Sprintf("wrong-prefix/%s", denom)), bz)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			expDenomEscrows = sdk.Coins{}
			ctx := suite.chainA.GetContext()

			storeKey := suite.chainA.GetSimApp().GetKey(types.ModuleName)
			store = ctx.KVStore(storeKey)

			tc.malleate()

			denomEscrows := suite.chainA.GetSimApp().TransferKeeper.GetAllTotalEscrowed(ctx)

			if tc.expPass {
				suite.Require().Len(expDenomEscrows, len(denomEscrows))
				suite.Require().ElementsMatch(expDenomEscrows, denomEscrows)
			} else {
				suite.Require().Empty(denomEscrows)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetAllForwardedPackets() {
	suite.SetupTest()

	// Store forward packets on transfer/channel-1 and transfer/channel-2
	for _, channelID := range []string{"channel-1", "channel-2"} {
		// go across '10' to test numerical order
		for sequence := uint64(5); sequence <= 15; sequence++ {
			packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, ibctesting.TransferPort, channelID, "", "", clienttypes.ZeroHeight(), 0)
			suite.chainA.GetSimApp().TransferKeeper.SetForwardedPacket(suite.chainA.GetContext(), ibctesting.TransferPort, channelID, sequence, packet)
		}
	}

	packets := suite.chainA.GetSimApp().TransferKeeper.GetAllForwardedPackets(suite.chainA.GetContext())
	// Assert each packets is as expected
	i := 0
	for _, channelID := range []string{"channel-1", "channel-2"} {
		for sequence := uint64(5); sequence <= 15; sequence++ {
			forwardedPacket := packets[i]

			expForwardKey := channeltypes.NewPacketID(ibctesting.TransferPort, channelID, sequence)
			suite.Require().Equal(forwardedPacket.ForwardKey, expForwardKey)

			expPacket := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, ibctesting.TransferPort, channelID, "", "", clienttypes.ZeroHeight(), 0)
			suite.Require().Equal(forwardedPacket.Packet, expPacket)

			i++
		}
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
		tc := tc

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
	suite.Require().False(isChannelKeeper)

	// set the ics4 wrapper to the channel keeper
	suite.chainA.GetSimApp().TransferKeeper.WithICS4Wrapper(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper)
	ics4Wrapper = suite.chainA.GetSimApp().TransferKeeper.GetICS4Wrapper()

	suite.Require().IsType((*channelkeeper.Keeper)(nil), ics4Wrapper)
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
			suite.chainA.GetSimApp().AuthKeeper.GetModuleAddress(types.ModuleName),
			false,
		},
		{
			"regular address",
			suite.chainA.SenderAccount.GetAddress(),
			false,
		},
		{
			"blocked address",
			suite.chainA.GetSimApp().AuthKeeper.GetModuleAddress(minttypes.ModuleName),
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.expBlock, suite.chainA.GetSimApp().TransferKeeper.IsBlockedAddr(tc.addr))
		})
	}
}
