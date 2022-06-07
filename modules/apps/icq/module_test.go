package icq_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	icq "github.com/cosmos/ibc-go/v3/modules/apps/icq"
	types "github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	"github.com/cosmos/ibc-go/v3/testing/simapp"
)

type InterchainQueriesTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
}

func TestICQTestSuite(t *testing.T) {
	suite.Run(t, new(InterchainQueriesTestSuite))
}

func (suite *InterchainQueriesTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
}

func (suite *InterchainQueriesTestSuite) TestInitModule() {
	// setup and basic testing
	app := simapp.NewSimApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, map[int64]bool{}, simapp.DefaultNodeHome, 5, simapp.MakeTestEncodingConfig(), simapp.EmptyAppOptions{})
	appModule, ok := app.GetModuleManager().Modules[types.ModuleName].(icq.AppModule)
	suite.Require().True(ok)

	header := tmproto.Header{
		ChainID: "testchain",
		Height:  1,
		Time:    suite.coordinator.CurrentTime.UTC(),
	}

	ctx := app.GetBaseApp().NewContext(true, header)

	// ensure params are not set
	suite.Require().Panics(func() {
		app.ICQKeeper.GetParams(ctx)
	})

	params := types.DefaultParams()
	params.HostEnabled = true
	expAllowMessages := []string{"sdk.Msg"}
	params.AllowQueries = expAllowMessages
	suite.Require().False(app.IBCKeeper.PortKeeper.IsBound(ctx, types.PortID))

	testCases := []struct {
		name         string
		malleate     func()
		expQueryPass bool
	}{
		{
			"query module is set", func() {
				var ok bool
				appModule, ok = app.GetModuleManager().Modules[types.ModuleName].(icq.AppModule)
				suite.Require().True(ok)
			}, true,
		},
		{
			"query module is not set", func() {
				// appModule = icq.NewAppModule() // need to set NewAppModule to take `*keeper.Keeper` instead of `keeper.Keeper`
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// reset app state
			app = simapp.NewSimApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, map[int64]bool{}, simapp.DefaultNodeHome, 5, simapp.MakeTestEncodingConfig(), simapp.EmptyAppOptions{})
			header := tmproto.Header{
				ChainID: "testchain",
				Height:  1,
				Time:    suite.coordinator.CurrentTime.UTC(),
			}

			ctx := app.GetBaseApp().NewContext(true, header)

			tc.malleate()

			suite.Require().NotPanics(func() {
				appModule.InitModule(ctx, params)
			})

			if tc.expQueryPass {
				params = app.ICQKeeper.GetParams(ctx)
				suite.Require().True(params.HostEnabled)
				suite.Require().Equal(expAllowMessages, params.AllowQueries)
				suite.Require().True(app.IBCKeeper.PortKeeper.IsBound(ctx, types.PortID))
			}

		})
	}

}
