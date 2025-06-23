package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/05-port/keeper"
	"github.com/cosmos/ibc-go/v10/testing/simapp"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	ctx    sdk.Context
	keeper *keeper.Keeper
}

func (s *KeeperTestSuite) SetupTest() {
	isCheckTx := false
	app := simapp.Setup(s.T(), isCheckTx)

	s.ctx = app.NewContext(isCheckTx)
	s.keeper = app.IBCKeeper.PortKeeper
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}
