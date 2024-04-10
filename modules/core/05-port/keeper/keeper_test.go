package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v8/testing/simapp"
)

var (
	validPort   = "validportid"
	invalidPort = "(invalidPortID)"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	ctx    sdk.Context
	keeper *keeper.Keeper
}

func (suite *KeeperTestSuite) SetupTest() {
	isCheckTx := false
	app := simapp.Setup(suite.T(), isCheckTx)

	suite.ctx = app.BaseApp.NewContext(isCheckTx)
	suite.keeper = app.IBCKeeper.PortKeeper
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestBind() {
	// Test that invalid portID causes panic
	require.Panics(suite.T(), func() { suite.keeper.BindPort(suite.ctx, invalidPort) }, "Did not panic on invalid portID")

	// Test that valid BindPort returns capability key
	capKey := suite.keeper.BindPort(suite.ctx, validPort)
	require.NotNil(suite.T(), capKey, "capabilityKey is nil on valid BindPort")

	isBound := suite.keeper.IsBound(suite.ctx, validPort)
	require.True(suite.T(), isBound, "port is bound successfully")

	isNotBound := suite.keeper.IsBound(suite.ctx, "not-a-port")
	require.False(suite.T(), isNotBound, "port is not bound")

	// Test that rebinding the same portid causes panic
	require.Panics(suite.T(), func() { suite.keeper.BindPort(suite.ctx, validPort) }, "did not panic on re-binding the same port")
}

func (suite *KeeperTestSuite) TestAuthenticate() {
	capKey := suite.keeper.BindPort(suite.ctx, validPort)

	// Require that passing in invalid portID causes panic
	require.Panics(suite.T(), func() { suite.keeper.Authenticate(suite.ctx, capKey, invalidPort) }, "did not panic on invalid portID")

	// Valid authentication should return true
	auth := suite.keeper.Authenticate(suite.ctx, capKey, validPort)
	require.True(suite.T(), auth, "valid authentication failed")

	// Test that authenticating against incorrect portid fails
	auth = suite.keeper.Authenticate(suite.ctx, capKey, "wrongportid")
	require.False(suite.T(), auth, "invalid authentication failed")

	// Test that authenticating port against different valid
	// capability key fails
	capKey2 := suite.keeper.BindPort(suite.ctx, "otherportid")
	auth = suite.keeper.Authenticate(suite.ctx, capKey2, validPort)
	require.False(suite.T(), auth, "invalid authentication for different capKey failed")
}

func (suite *KeeperTestSuite) TestGetRoute() {
	testCases := []struct {
		msg     string
		module  string
		expPass bool
	}{
		{
			"success",
			fmt.Sprintf("%s-%d", exported.Tendermint, 0),
			true,
		},
		{
			"failure - route does not exist",
			"invalid-route",
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			cdc := suite.chainA.App.AppCodec()
			storeKey := storetypes.NewKVStoreKey("store-key")
			tmLightClientModule := ibctm.NewLightClientModule(cdc, authtypes.NewModuleAddress(govtypes.ModuleName).String())
			router := clienttypes.NewRouter(storeKey)
			router.AddRoute(exported.Tendermint, &tmLightClientModule)

			route, ok := suite.keeper.GetRoute(tc.module)
			if tc.expPass {
				suite.Require().True(ok)
				suite.Require().NotNil(route)
				suite.Require().IsType(&ibctm.LightClientModule{}, route)
			} else {
				suite.Require().False(ok)
				suite.Require().Nil(route)
			}
		})
	}
}
