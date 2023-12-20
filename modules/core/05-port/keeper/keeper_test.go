package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
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
	// Test that invalid portID causes error
	_, err := suite.keeper.BindPort(suite.ctx, invalidPort)
	require.Error(suite.T(), err, "did not error on invalid portID")

	// Test that valid BindPort returns capability key
	capKey, err := suite.keeper.BindPort(suite.ctx, validPort)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), capKey, "capabilityKey is nil on valid BindPort")

	isBound := suite.keeper.IsBound(suite.ctx, validPort)
	require.True(suite.T(), isBound, "port is bound successfully")

	isNotBound := suite.keeper.IsBound(suite.ctx, "not-a-port")
	require.False(suite.T(), isNotBound, "port is not bound")

	// Test that rebinding the same portid causes error
	_, err = suite.keeper.BindPort(suite.ctx, validPort)
	require.Error(suite.T(), err, "did not panic on re-binding the same port")
}

func (suite *KeeperTestSuite) TestAuthenticate() {
	capKey, err := suite.keeper.BindPort(suite.ctx, validPort)
	require.NoError(suite.T(), err)

	// Require that passing in invalid portID causes panic
	auth, err := suite.keeper.Authenticate(suite.ctx, capKey, invalidPort)
	require.Error(suite.T(), err, "did not error on invalid portID")
	require.False(suite.T(), auth, "invalid authentication failed")

	// Valid authentication should return true
	auth, err = suite.keeper.Authenticate(suite.ctx, capKey, validPort)
	require.NoError(suite.T(), err)
	require.True(suite.T(), auth, "valid authentication failed")

	// Test that authenticating against incorrect portid fails
	auth, err = suite.keeper.Authenticate(suite.ctx, capKey, "wrongportid")
	require.NoError(suite.T(), err)
	require.False(suite.T(), auth, "invalid authentication failed")

	// Test that authenticating port against different valid
	// capability key fails
	capKey2, err := suite.keeper.BindPort(suite.ctx, "otherportid")
	require.NoError(suite.T(), err)
	auth, err = suite.keeper.Authenticate(suite.ctx, capKey2, validPort)
	require.NoError(suite.T(), err)
	require.False(suite.T(), auth, "invalid authentication for different capKey failed")
}
