package keeper_test

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v7/modules/core/05-port/keeper"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
)

var (
	validPort   = "validportid"
	invalidPort = "(invalidPortID)"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	keeper *keeper.Keeper
}

func (s *KeeperTestSuite) SetupTest() {
	isCheckTx := false
	app := simapp.Setup()

	s.ctx = app.BaseApp.NewContext(isCheckTx, tmproto.Header{})
	s.keeper = &app.IBCKeeper.PortKeeper
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestBind() {
	// Test that invalid portID causes panic
	require.Panics(s.T(), func() { s.keeper.BindPort(s.ctx, invalidPort) }, "Did not panic on invalid portID")

	// Test that valid BindPort returns capability key
	capKey := s.keeper.BindPort(s.ctx, validPort)
	require.NotNil(s.T(), capKey, "capabilityKey is nil on valid BindPort")

	isBound := s.keeper.IsBound(s.ctx, validPort)
	require.True(s.T(), isBound, "port is bound successfully")

	isNotBound := s.keeper.IsBound(s.ctx, "not-a-port")
	require.False(s.T(), isNotBound, "port is not bound")

	// Test that rebinding the same portid causes panic
	require.Panics(s.T(), func() { s.keeper.BindPort(s.ctx, validPort) }, "did not panic on re-binding the same port")
}

func (s *KeeperTestSuite) TestAuthenticate() {
	capKey := s.keeper.BindPort(s.ctx, validPort)

	// Require that passing in invalid portID causes panic
	require.Panics(s.T(), func() { s.keeper.Authenticate(s.ctx, capKey, invalidPort) }, "did not panic on invalid portID")

	// Valid authentication should return true
	auth := s.keeper.Authenticate(s.ctx, capKey, validPort)
	require.True(s.T(), auth, "valid authentication failed")

	// Test that authenticating against incorrect portid fails
	auth = s.keeper.Authenticate(s.ctx, capKey, "wrongportid")
	require.False(s.T(), auth, "invalid authentication failed")

	// Test that authenticating port against different valid
	// capability key fails
	capKey2 := s.keeper.BindPort(s.ctx, "otherportid")
	auth = s.keeper.Authenticate(s.ctx, capKey2, validPort)
	require.False(s.T(), auth, "invalid authentication for different capKey failed")
}
